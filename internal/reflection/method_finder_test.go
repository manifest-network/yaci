package reflection_test

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/liftedinit/yaci/internal/reflection"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestFindMethodDescriptor(t *testing.T) {
	// Create mock file descriptor
	fdProto := &descriptorpb.FileDescriptorProto{
		Name: proto.String("test.proto"),
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("TestService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("TestMethod"),
						InputType:  proto.String("TestInput"),
						OutputType: proto.String("TestOutput"),
					},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("TestInput"),
			},
			{
				Name: proto.String("TestOutput"),
			},
		},
	}

	// Convert to FileDescriptor
	fd, err := protodesc.NewFile(fdProto, nil)
	assert.NoError(t, err)

	// Create a registry and register the file descriptor
	files := new(protoregistry.Files)
	err = files.RegisterFile(fd)
	assert.NoError(t, err)

	// Test finding the method descriptor
	methodDesc, err := reflection.FindMethodDescriptor(files, "TestService", "TestMethod")
	assert.NoError(t, err)
	assert.NotNil(t, methodDesc)
	assert.Equal(t, "TestMethod", string(methodDesc.Name()))

	// Test method not found
	_, err = reflection.FindMethodDescriptor(files, "TestService", "NonExistentMethod")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "method NonExistentMethod not found in service TestService")

	// Test service not found
	_, err = reflection.FindMethodDescriptor(files, "NonExistentService", "TestMethod")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "method TestMethod not found in service NonExistentService")
}
