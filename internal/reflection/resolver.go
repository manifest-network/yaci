package reflection

import (
	"context"
	"fmt"

	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// CustomResolver implements the Resolver interface required by protojson.
type CustomResolver struct {
	files       *protoregistry.Files
	refClient   reflection.ServerReflectionClient
	ctx         context.Context
	seenSymbols map[string]bool
}

// NewCustomResolver creates a new instance of CustomResolver.
func NewCustomResolver(files *protoregistry.Files, refClient reflection.ServerReflectionClient, ctx context.Context) *CustomResolver {
	return &CustomResolver{
		files:       files,
		refClient:   refClient,
		ctx:         ctx,
		seenSymbols: make(map[string]bool),
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
		return nil, err
	}

	// Try to find the message again after fetching
	desc, err := r.files.FindDescriptorByName(name)
	if err != nil {
		return nil, err
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
func (r *CustomResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

// FindExtensionByNumber is not implemented.
func (r *CustomResolver) FindExtensionByNumber(message protoreflect.FullName, fieldNumber protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

func (r *CustomResolver) fetchDescriptorBySymbol(symbol string) error {
	if r.seenSymbols[symbol] {
		return nil
	}
	r.seenSymbols[symbol] = true

	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: symbol,
		},
	}

	stream, err := r.refClient.ServerReflectionInfo(r.ctx)
	if err != nil {
		return err
	}

	if err := stream.Send(req); err != nil {
		return err
	}

	resp, err := stream.Recv()
	if err != nil {
		return err
	}

	fdResponse, ok := resp.MessageResponse.(*reflection.ServerReflectionResponse_FileDescriptorResponse)
	if !ok {
		return fmt.Errorf("unexpected response: %v", resp)
	}

	// Build and register the new descriptors
	for _, fdBytes := range fdResponse.FileDescriptorResponse.FileDescriptorProto {
		fdProto := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(fdBytes, fdProto); err != nil {
			return err
		}

		name := fdProto.GetName()
		if _, err := r.files.FindFileByPath(name); err == nil {
			// Already registered
			continue
		}

		// Recursively fetch dependencies
		for _, dep := range fdProto.Dependency {
			if _, err := r.files.FindFileByPath(dep); err != nil {
				if err := r.fetchDescriptorByName(dep); err != nil {
					return err
				}
			}
		}

		fd, err := protodesc.NewFile(fdProto, r.files)
		if err != nil {
			return err
		}

		if err := r.files.RegisterFile(fd); err != nil {
			return err
		}
	}

	return nil
}

func (r *CustomResolver) fetchDescriptorByName(name string) error {
	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileByFilename{
			FileByFilename: name,
		},
	}

	stream, err := r.refClient.ServerReflectionInfo(r.ctx)
	if err != nil {
		return err
	}

	if err := stream.Send(req); err != nil {
		return err
	}

	resp, err := stream.Recv()
	if err != nil {
		return err
	}

	fdResponse, ok := resp.MessageResponse.(*reflection.ServerReflectionResponse_FileDescriptorResponse)
	if !ok {
		return fmt.Errorf("unexpected response: %v", resp)
	}

	// Build and register the new descriptors
	for _, fdBytes := range fdResponse.FileDescriptorResponse.FileDescriptorProto {
		fdProto := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(fdBytes, fdProto); err != nil {
			return err
		}

		name := fdProto.GetName()
		if _, err := r.files.FindFileByPath(name); err == nil {
			// Already registered
			continue
		}

		// Recursively fetch dependencies
		for _, dep := range fdProto.Dependency {
			if _, err := r.files.FindFileByPath(dep); err != nil {
				if err := r.fetchDescriptorByName(dep); err != nil {
					return err
				}
			}
		}

		fd, err := protodesc.NewFile(fdProto, r.files)
		if err != nil {
			return err
		}

		if err := r.files.RegisterFile(fd); err != nil {
			return err
		}
	}

	return nil
}
