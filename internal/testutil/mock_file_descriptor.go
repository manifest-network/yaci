package testutil

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	TestProtoName         = "test.proto"
	TestServiceName       = "TestService"
	TestMethodName        = "TestMethod"
	TestInputName         = "TestInput"
	TestOutputName        = "TestOutput"
	DependencyProtoName   = "dependency.proto"
	DependencyMessageName = "DependencyMessage"
)

var (
	MockFileDescriptor = &descriptorpb.FileDescriptorProto{
		Name: proto.String(TestProtoName),
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String(TestServiceName),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String(TestMethodName),
						InputType:  proto.String(TestInputName),
						OutputType: proto.String(TestOutputName),
					},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String(TestInputName)},
			{Name: proto.String(TestOutputName)},
		},
		Dependency: []string{DependencyProtoName},
	}

	MockDependencyFileDescriptor = &descriptorpb.FileDescriptorProto{
		Name: proto.String(DependencyProtoName),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String(DependencyMessageName)},
		},
	}

	MockFileDescriptorSet = &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			MockDependencyFileDescriptor,
			MockFileDescriptor,
		},
	}
)
