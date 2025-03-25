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
	fdMap := make(map[string]*descriptorpb.FileDescriptorProto, len(descriptors))
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
		patchABCIDescriptor(fdProto)

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

// patchABCIDescriptor applies specific patches to known problematic descriptors
func patchABCIDescriptor(descriptor *descriptorpb.FileDescriptorProto) {
	// Fix for cosmos/base/abci/v1beta1/abci.proto
	// The `raw_log` field can contain invalid UTF-8, so we change its type from string to bytes
	// See https://github.com/cosmos/cosmos-sdk/issues/22414
	if descriptor.GetName() == "cosmos/base/abci/v1beta1/abci.proto" {
		for _, msgType := range descriptor.GetMessageType() {
			for _, field := range msgType.GetField() {
				if field.GetName() == "raw_log" && field.GetType() == descriptorpb.FieldDescriptorProto_TYPE_STRING {
					field.Type = descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum()
				}
			}
		}
	}
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
