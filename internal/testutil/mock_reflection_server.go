package testutil

import (
	"context"
	"net"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/descriptorpb"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	reflectionpb.RegisterServerReflectionServer(s, &mockServerReflection{})
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
}

func MockDialer(_ context.Context, _ string) (net.Conn, error) {
	return lis.Dial()
}

type mockServerReflection struct {
	reflectionpb.UnimplementedServerReflectionServer
}

func (s *mockServerReflection) ServerReflectionInfo(stream reflectionpb.ServerReflection_ServerReflectionInfoServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		switch req.MessageRequest.(type) {
		case *reflectionpb.ServerReflectionRequest_ListServices:
			stream.Send(createListServicesResponse(TestServiceName))
		case *reflectionpb.ServerReflectionRequest_FileByFilename:
			stream.Send(createFileDescriptorResponse(MockDependencyFileDescriptor))
		case *reflectionpb.ServerReflectionRequest_FileContainingSymbol:
			stream.Send(createFileContainingSymbolResponse(MockFileDescriptor))
		}
	}
}

func mustMarshal(pb proto.Message) []byte {
	data, err := proto.Marshal(pb)
	if err != nil {
		panic(err)
	}
	return data
}

func createListServicesResponse(serviceName string) *reflectionpb.ServerReflectionResponse {
	return &reflectionpb.ServerReflectionResponse{
		MessageResponse: &reflectionpb.ServerReflectionResponse_ListServicesResponse{
			ListServicesResponse: &reflectionpb.ListServiceResponse{
				Service: []*reflectionpb.ServiceResponse{
					{Name: serviceName},
				},
			},
		},
	}
}

func createFileDescriptorResponse(fd *descriptorpb.FileDescriptorProto) *reflectionpb.ServerReflectionResponse {
	return &reflectionpb.ServerReflectionResponse{
		MessageResponse: &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{
			FileDescriptorResponse: &reflectionpb.FileDescriptorResponse{
				FileDescriptorProto: [][]byte{
					mustMarshal(fd),
				},
			},
		},
	}
}

func createFileContainingSymbolResponse(fd *descriptorpb.FileDescriptorProto) *reflectionpb.ServerReflectionResponse {
	return &reflectionpb.ServerReflectionResponse{
		MessageResponse: &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{
			FileDescriptorResponse: &reflectionpb.FileDescriptorResponse{
				FileDescriptorProto: [][]byte{
					mustMarshal(fd),
				},
			},
		},
	}
}
