package reflection_test

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/liftedinit/yaci/internal/reflection"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestBuildFileDescriptorSet(t *testing.T) {
	cases := []struct {
		name        string
		descriptors []*descriptorpb.FileDescriptorProto
		error       string
	}{
		{
			name: "valid descriptors",
			descriptors: []*descriptorpb.FileDescriptorProto{
				{
					Name:       proto.String("file1.proto"),
					Dependency: []string{"file2.proto"},
				},
				{
					Name: proto.String("file2.proto"),
				},
			},
		},
		{
			name: "circular dependency",
			descriptors: []*descriptorpb.FileDescriptorProto{
				{
					Name:       proto.String("file1.proto"),
					Dependency: []string{"file2.proto"},
				},
				{
					Name:       proto.String("file2.proto"),
					Dependency: []string{"file1.proto"},
				},
			},
			error: "circular dependency detected",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files, err := reflection.BuildFileDescriptorSet(tc.descriptors)
			if tc.error != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.error)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, files)
				for _, fd := range tc.descriptors {
					fd, err := files.FindFileByPath(*fd.Name)
					assert.NoError(t, err)
					assert.NotNil(t, fd)
				}
			}
		})
	}
}
