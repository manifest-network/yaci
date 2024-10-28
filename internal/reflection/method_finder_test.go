package reflection_test

import (
	"testing"

	"github.com/liftedinit/yaci/internal/reflection"
	"github.com/liftedinit/yaci/internal/testutil"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestFindMethodDescriptor(t *testing.T) {
	files := new(protoregistry.Files)

	// Register the sorted descriptors
	for _, fdProto := range testutil.MockFileDescriptorSet.File {
		fd, err := protodesc.NewFile(fdProto, files)
		assert.NoError(t, err)

		err = files.RegisterFile(fd)
		assert.NoError(t, err)
	}

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
