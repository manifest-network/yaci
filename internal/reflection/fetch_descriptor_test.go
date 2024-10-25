package reflection_test

import (
	"context"
	"net"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/liftedinit/yaci/internal/reflection"
	"github.com/stretchr/testify/assert"
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

func bufDialer(context.Context, string) (net.Conn, error) {
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
			resp := &reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_ListServicesResponse{
					ListServicesResponse: &reflectionpb.ListServiceResponse{
						Service: []*reflectionpb.ServiceResponse{
							{Name: "TestService"},
						},
					},
				},
			}
			stream.Send(resp)
		case *reflectionpb.ServerReflectionRequest_FileByFilename:
			resp := &reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{
					FileDescriptorResponse: &reflectionpb.FileDescriptorResponse{
						FileDescriptorProto: [][]byte{
							mustMarshal(&descriptorpb.FileDescriptorProto{
								Name: proto.String("description.proto"),
								MessageType: []*descriptorpb.DescriptorProto{
									{Name: proto.String("DependencyMessage")},
								},
							}),
						},
					},
				},
			}
			stream.Send(resp)
		case *reflectionpb.ServerReflectionRequest_FileContainingSymbol:
			resp := &reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{
					FileDescriptorResponse: &reflectionpb.FileDescriptorResponse{
						FileDescriptorProto: [][]byte{
							mustMarshal(&descriptorpb.FileDescriptorProto{
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
									{Name: proto.String("TestInput")},
									{Name: proto.String("TestOutput")},
								},
								Dependency: []string{"dependency.proto"},
							}),
						},
					},
				},
			}
			stream.Send(resp)
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

func TestFetchAllDescriptors(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	assert.NoError(t, err)
	defer conn.Close()

	descriptors, err := reflection.FetchAllDescriptors(ctx, conn, 3)
	assert.NoError(t, err)
	assert.NotNil(t, descriptors)
	assert.NotEmpty(t, descriptors)
}
