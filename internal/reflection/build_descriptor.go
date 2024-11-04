package reflection

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// BuildFileDescriptorSet builds the file descriptor set from the descriptors.
func BuildFileDescriptorSet(descriptors []*descriptorpb.FileDescriptorProto) (*protoregistry.Files, error) {
	files := &protoregistry.Files{}

	// Build a map of file descriptors for dependency resolution
	fdMap := make(map[string]*descriptorpb.FileDescriptorProto)
	for _, fdProto := range descriptors {
		fdMap[fdProto.GetName()] = fdProto
	}

	// Perform a topological sort of the file descriptors
	sortedDescriptors, err := topologicalSort(fdMap)
	if err != nil {
		return nil, fmt.Errorf("failed to sort file descriptors: %w", err)
	}

	// Register the sorted descriptors
	for _, fdProto := range sortedDescriptors {
		// The Protocol Buffer specification mentions that the `string` type MUST always contain UTF-8 encoded or 7-bit ASCII text.
		// The `raw_log` field of `TxResponse` in `cosmos/base/abci/v1beta1/abci.proto` is of type `string` but can contain invalid UTF-8.
		//
		// We change the field type to `bytes` to avoid issues when unmarshalling the response.
		// See https://github.com/cosmos/cosmos-sdk/issues/22414
		//
		// Marshaling the response to JSON will still work as expected since the field is serialized as a base64-encoded string.
		if fdProto.GetName() == "cosmos/base/abci/v1beta1/abci.proto" {
			for _, msgType := range fdProto.GetMessageType() {
				for _, field := range msgType.GetField() {
					if field.GetName() == "raw_log" && field.GetType() == *descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum() {
						field.Type = descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum()
					}
				}
			}
		}

		fd, err := protodesc.NewFile(fdProto, files)
		if err != nil {
			return nil, fmt.Errorf("failed to create file descriptor: %w", err)
		}

		if err := files.RegisterFile(fd); err != nil {
			return nil, fmt.Errorf("failed to register file descriptor: %w", err)
		}
	}

	return files, nil
}

// topologicalSort sorts the file descriptors in dependency order.
func topologicalSort(fdMap map[string]*descriptorpb.FileDescriptorProto) ([]*descriptorpb.FileDescriptorProto, error) {
	visited := make(map[string]bool)
	tempMarked := make(map[string]bool)
	var sortedDescriptors []*descriptorpb.FileDescriptorProto

	var visit func(string) error
	visit = func(name string) error {
		if tempMarked[name] {
			return fmt.Errorf("circular dependency detected at %s", name)
		}
		if !visited[name] {
			tempMarked[name] = true
			fdProto := fdMap[name]
			if fdProto == nil {
				// Should not happen
				tempMarked[name] = false
				return fmt.Errorf("file descriptor not found: %s", name)
			}
			for _, dep := range fdProto.Dependency {
				if _, exists := fdMap[dep]; exists {
					if err := visit(dep); err != nil {
						return err
					}
				}
			}
			visited[name] = true
			tempMarked[name] = false
			sortedDescriptors = append(sortedDescriptors, fdProto)
		}
		return nil
	}

	for name := range fdMap {
		if !visited[name] {
			if err := visit(name); err != nil {
				return nil, err
			}
		}
	}

	return sortedDescriptors, nil
}
