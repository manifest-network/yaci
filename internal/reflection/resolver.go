package reflection

import (
	"context"
	"fmt"

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
}

// NewCustomResolver creates a new instance of CustomResolver.
func NewCustomResolver(files *protoregistry.Files, grpcConn *grpc.ClientConn, ctx context.Context, maxRetries uint) *CustomResolver {
	return &CustomResolver{
		files:       files,
		grpcConn:    grpcConn,
		ctx:         ctx,
		seenSymbols: make(map[string]bool),
		maxRetries:  maxRetries,
	}
}

// Files returns the registry's files.
func (r *CustomResolver) Files() *protoregistry.Files {
	return r.files
}

// FindMessageByName finds a message descriptor by its name.
func (r *CustomResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	// First, try to find the message in the existing registry
	desc, _ := r.files.FindDescriptorByName(name)

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
	desc, err := r.files.FindDescriptorByName(name)
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
	if r.seenSymbols[symbol] {
		return nil
	}
	r.seenSymbols[symbol] = true

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
		if _, err := r.files.FindFileByPath(name); err == nil {
			// Already registered
			continue
		}

		// Recursively fetch dependencies
		for _, dep := range fdProto.Dependency {
			if _, err := r.files.FindFileByPath(dep); err != nil {
				if err := r.fetchDescriptorByName(dep, maxRetries); err != nil {
					return errors.WithMessagef(err, "failed to fetch dependency %s", dep)
				}
			}
		}

		fd, err := protodesc.NewFile(fdProto, r.files)
		if err != nil {
			return errors.WithMessagef(err, "failed to create file descriptor for %s", name)
		}

		if err := r.files.RegisterFile(fd); err != nil {
			return errors.WithMessagef(err, "failed to register file %s", name)
		}
	}
	return nil
}
