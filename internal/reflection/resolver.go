package reflection

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// CustomResolver implements the Resolver interface required by protojson.
// It fetches file descriptors from a gRPC server using the reflection API and registers them in a protoregistry.Files.
// It also correctly resolves message types by fetching dependencies recursively.
type CustomResolver struct {
	files       *protoregistry.Files
	grpcConn    *grpc.ClientConn
	ctx         context.Context
	seenSymbols map[string]bool
	maxRetries  uint
	mu          sync.RWMutex
}

// NewCustomResolver creates a new instance of CustomResolver.
func NewCustomResolver(ctx context.Context, files *protoregistry.Files, grpcConn *grpc.ClientConn, maxRetries uint) *CustomResolver {
	return &CustomResolver{
		files:       files, // Note: The protoregistry.Files type is safe for concurrent use by multiple goroutines, but it is not safe to concurrently mutate the registry while also being used.
		grpcConn:    grpcConn,
		ctx:         ctx,
		seenSymbols: make(map[string]bool),
		maxRetries:  maxRetries,
	}
}

func (r *CustomResolver) FindMethodDescriptor(serviceName, methodName string) (protoreflect.MethodDescriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var methodDesc protoreflect.MethodDescriptor
	var found bool

	r.files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			svc := services.Get(i)
			if string(svc.FullName()) == serviceName {
				methods := svc.Methods()
				for j := 0; j < methods.Len(); j++ {
					m := methods.Get(j)
					if string(m.Name()) == methodName {
						methodDesc = m
						found = true
						return false
					}
				}
			}
		}
		return true
	})
	if !found {
		return nil, fmt.Errorf("method %s not found in service %s", methodName, serviceName)
	}
	return methodDesc, nil
}

// FindMessageByName finds a message descriptor by its name.
func (r *CustomResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	// First, try to find the message in the existing registry
	r.mu.RLock()
	desc, err := r.files.FindDescriptorByName(name)
	r.mu.RUnlock()

	if err == nil && desc != nil {
		return createMessageType(desc, name)
	}

	// If not found, attempt to fetch the descriptor via reflection
	if err := r.fetchDescriptorBySymbol(string(name)); err != nil {
		return nil, fmt.Errorf("failed to fetch descriptor for symbol %s: %w", name, err)
	}

	// Try to find the message again after fetching
	r.mu.RLock()
	desc, err = r.files.FindDescriptorByName(name)
	r.mu.RUnlock()

	if err != nil {
		return nil, fmt.Errorf("failed to find message by name %s: %w", name, err)
	}

	return createMessageType(desc, name)
}

func createMessageType(desc protoreflect.Descriptor, name protoreflect.FullName) (protoreflect.MessageType, error) {
	if desc == nil {
		return nil, fmt.Errorf("message %s not found", name)
	}

	msgDesc, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %s is not a message", name)
	}

	return dynamicpb.NewMessageType(msgDesc), nil
}

// FindMessageByURL finds a message descriptor by its URL.
func (r *CustomResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	return r.FindMessageByName(protoreflect.FullName(url[1:]))
}

// FindExtensionByName is not implemented.
func (r *CustomResolver) FindExtensionByName(_ protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

// FindExtensionByNumber is not implemented.
func (r *CustomResolver) FindExtensionByNumber(_ protoreflect.FullName, _ protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

func (r *CustomResolver) fetchDescriptorBySymbol(symbol string) error {
	r.mu.RLock()
	seen := r.seenSymbols[symbol]
	r.mu.RUnlock()

	if seen {
		return nil
	}

	r.mu.Lock()
	r.seenSymbols[symbol] = true
	r.mu.Unlock()

	// Create the request to fetch file descriptors containing the symbol
	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: symbol,
		},
	}

	fdProtos, err := fetchFileDescriptorsFromRequest(r.ctx, r.grpcConn, req, r.maxRetries)
	if err != nil {
		return fmt.Errorf("failed to fetch file descriptors containing symbol %s: %w", symbol, err)
	}

	return r.processFileDescriptors(fdProtos, r.maxRetries)
}

func (r *CustomResolver) fetchDescriptorByName(name string, maxRetries uint) error {
	// Create the request to fetch file descriptors by filename
	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileByFilename{
			FileByFilename: name,
		},
	}

	fdProtos, err := fetchFileDescriptorsFromRequest(r.ctx, r.grpcConn, req, maxRetries)
	if err != nil {
		return fmt.Errorf("failed to fetch file descriptors for file %s: %w", name, err)
	}

	return r.processFileDescriptors(fdProtos, maxRetries)
}

// processFileDescriptors processes the fetched file descriptors and recursively fetches their dependencies.
func (r *CustomResolver) processFileDescriptors(fdProtos []*descriptorpb.FileDescriptorProto, maxRetries uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, fdProto := range fdProtos {
		name := fdProto.GetName()
		_, err := r.files.FindFileByPath(name)
		if err == nil {
			// Already registered
			continue
		}

		if err := r.fetchDependencies(fdProto.Dependency, maxRetries); err != nil {
			return err
		}

		if err := r.registerFileDescriptor(fdProto); err != nil {
			return err
		}
	}

	return nil
}

func (r *CustomResolver) fetchDependencies(dependencies []string, maxRetries uint) error {
	for _, dep := range dependencies {
		// Skip if already registered
		if _, err := r.files.FindFileByPath(dep); err == nil {
			continue
		}

		// Unlock while fetching to avoid deadlocks with recursive calls
		r.mu.Unlock()
		err := r.fetchDescriptorByName(dep, maxRetries)
		r.mu.Lock()

		if err != nil {
			return fmt.Errorf("failed to fetch dependency %s: %w", dep, err)
		}
	}
	return nil
}

func (r *CustomResolver) registerFileDescriptor(fdProto *descriptorpb.FileDescriptorProto) error {
	name := fdProto.GetName()

	fd, err := protodesc.NewFile(fdProto, r.files)
	if err != nil {
		return fmt.Errorf("failed to create file descriptor for %s: %w", name, err)
	}

	if err := r.files.RegisterFile(fd); err != nil {
		return fmt.Errorf("failed to register file %s: %w", name, err)
	}

	return nil
}
