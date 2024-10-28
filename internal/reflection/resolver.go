package reflection

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// CustomResolver implements the Resolver interface required by protojson.
type CustomResolver struct {
	files       *protoregistry.Files
	grpcConn    *grpc.ClientConn
	ctx         context.Context
	seenSymbols map[string]bool
	maxRetries  uint
	mu          sync.Mutex
}

// NewCustomResolver creates a new instance of CustomResolver.
func NewCustomResolver(files *protoregistry.Files, grpcConn *grpc.ClientConn, ctx context.Context, maxRetries uint) *CustomResolver {
	return &CustomResolver{
		files:       files, // Note: The protoregistry.Files type is safe for concurrent use by multiple goroutines, but it is not safe to concurrently mutate the registry while also being used.
		grpcConn:    grpcConn,
		ctx:         ctx,
		seenSymbols: make(map[string]bool),
		maxRetries:  maxRetries,
	}
}

func (r *CustomResolver) FindMethodDescriptor(serviceName, methodName string) (protoreflect.MethodDescriptor, error) {
	var methodDesc protoreflect.MethodDescriptor
	var found bool
	r.mu.Lock()
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
	r.mu.Unlock()
	if !found {
		return nil, fmt.Errorf("method %s not found in service %s", methodName, serviceName)
	}
	return methodDesc, nil
}

// FindMessageByName finds a message descriptor by its name.
func (r *CustomResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	// First, try to find the message in the existing registry
	r.mu.Lock()
	desc, _ := r.files.FindDescriptorByName(name)
	r.mu.Unlock()

	if desc != nil {
		msgDesc, ok := desc.(protoreflect.MessageDescriptor)
		if !ok {
			return nil, fmt.Errorf("descriptor %s is not a message", name)
		}
		msgType := dynamicpb.NewMessageType(msgDesc)
		return msgType, nil
	}

	// If not found, attempt to fetch the descriptor via reflection
	if err := r.fetchDescriptorBySymbol(string(name)); err != nil {
		return nil, errors.WithMessagef(err, "failed to fetch descriptor for symbol %s", name)
	}

	// Try to find the message again after fetching
	r.mu.Lock()
	desc, err := r.files.FindDescriptorByName(name)
	r.mu.Unlock()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to find message by name %s", name)
	}

	if desc != nil {
		msgDesc, ok := desc.(protoreflect.MessageDescriptor)
		if !ok {
			return nil, fmt.Errorf("descriptor %s is not a message", name)
		}
		msgType := dynamicpb.NewMessageType(msgDesc)
		return msgType, nil
	}

	return nil, fmt.Errorf("message %s not found", name)
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
	r.mu.Lock()
	if r.seenSymbols[symbol] {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

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
		return errors.WithMessagef(err, "failed to fetch file descriptors containing symbol %s", symbol)
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
		return errors.WithMessagef(err, "failed to fetch file descriptors for file %s", name)
	}

	return r.processFileDescriptors(fdProtos, maxRetries)
}

// processFileDescriptors processes the fetched file descriptors and recursively fetches their dependencies.
func (r *CustomResolver) processFileDescriptors(fdProtos []*descriptorpb.FileDescriptorProto, maxRetries uint) error {
	for _, fdProto := range fdProtos {
		name := fdProto.GetName()
		r.mu.Lock()
		_, err := r.files.FindFileByPath(name)
		r.mu.Unlock()
		if err == nil {
			// Already registered
			continue
		}

		// Recursively fetch dependencies
		for _, dep := range fdProto.Dependency {
			r.mu.Lock()
			_, err := r.files.FindFileByPath(dep)
			r.mu.Unlock()
			if err != nil {
				if err := r.fetchDescriptorByName(dep, maxRetries); err != nil {
					return errors.WithMessagef(err, "failed to fetch dependency %s", dep)
				}
			}
		}

		fd, err := protodesc.NewFile(fdProto, r.files)
		if err != nil {
			return errors.WithMessagef(err, "failed to create file descriptor for %s", name)
		}

		r.mu.Lock()
		err = r.files.RegisterFile(fd)
		r.mu.Unlock()

		if err != nil {
			return errors.WithMessagef(err, "failed to register file %s", name)
		}
	}
	return nil
}
