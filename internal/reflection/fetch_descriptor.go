package reflection

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// FetchAllDescriptors retrieves all file descriptors supported by the server.
func FetchAllDescriptors(ctx context.Context, grpcClient *grpc.ClientConn, maxRetries uint) ([]*descriptorpb.FileDescriptorProto, error) {
	seenFiles := make(map[string]*descriptorpb.FileDescriptorProto)

	// List all services
	services, err := listServices(ctx, grpcClient, maxRetries)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	// For each service, fetch its file descriptors
	for _, service := range services {
		err := fetchFileDescriptors(ctx, grpcClient, service, seenFiles, maxRetries)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch file descriptors for service %s: %w", service, err)
		}
	}

	// Collect all descriptors
	result := make([]*descriptorpb.FileDescriptorProto, 0, len(seenFiles))
	for _, fd := range seenFiles {
		result = append(result, fd)
	}

	return result, nil
}

// listServices lists all services provided by the server via reflection.
func listServices(ctx context.Context, grpcClient *grpc.ClientConn, maxRetries uint) ([]string, error) {
	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_ListServices{
			ListServices: "*",
		},
	}

	resp, err := sendReflectionRequestWithRetry(ctx, grpcClient, req, maxRetries)
	if err != nil {
		return nil, fmt.Errorf("failed to list services via reflection: %w", err)
	}

	listServicesResp, ok := resp.MessageResponse.(*reflection.ServerReflectionResponse_ListServicesResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", resp.MessageResponse)
	}

	services := make([]string, 0, len(listServicesResp.ListServicesResponse.Service))
	for _, service := range listServicesResp.ListServicesResponse.Service {
		services = append(services, service.Name)
	}

	return services, nil
}

// fetchFileDescriptors fetches the file descriptors containing the given symbol and their dependencies.
func fetchFileDescriptors(ctx context.Context, grpcClient *grpc.ClientConn, symbol string, seen map[string]*descriptorpb.FileDescriptorProto, maxRetries uint) error {
	if _, exists := seen[symbol]; exists {
		return nil
	}

	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: symbol,
		},
	}

	fdProtos, err := fetchFileDescriptorsFromRequest(ctx, grpcClient, req, maxRetries)
	if err != nil {
		return fmt.Errorf("failed to fetch file descriptors containing symbol %s: %w", symbol, err)
	}

	return processFileDescriptors(ctx, grpcClient, fdProtos, seen, maxRetries)
}

// fetchFileByName fetches the file descriptor by filename and its dependencies.
func fetchFileByName(ctx context.Context, grpcClient *grpc.ClientConn, name string, seen map[string]*descriptorpb.FileDescriptorProto, maxRetries uint) error {
	if _, exists := seen[name]; exists {
		return nil
	}

	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileByFilename{
			FileByFilename: name,
		},
	}

	fdProtos, err := fetchFileDescriptorsFromRequest(ctx, grpcClient, req, maxRetries)
	if err != nil {
		return fmt.Errorf("failed to fetch file descriptors for file %s: %w", name, err)
	}

	return processFileDescriptors(ctx, grpcClient, fdProtos, seen, maxRetries)
}

// fetchFileDescriptorsFromRequest sends a reflection request and returns the file descriptors.
func fetchFileDescriptorsFromRequest(ctx context.Context, grpcClient *grpc.ClientConn, req *reflection.ServerReflectionRequest, maxRetries uint) ([]*descriptorpb.FileDescriptorProto, error) {
	resp, err := sendReflectionRequestWithRetry(ctx, grpcClient, req, maxRetries)
	if err != nil {
		return nil, err
	}

	fdResponse, ok := resp.MessageResponse.(*reflection.ServerReflectionResponse_FileDescriptorResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", resp.MessageResponse)
	}

	fdProtos := make([]*descriptorpb.FileDescriptorProto, 0, len(fdResponse.FileDescriptorResponse.FileDescriptorProto))
	for _, fdBytes := range fdResponse.FileDescriptorResponse.FileDescriptorProto {
		fdProto := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(fdBytes, fdProto); err != nil {
			return nil, fmt.Errorf("failed to unmarshal file descriptor: %w", err)
		}
		fdProtos = append(fdProtos, fdProto)
	}

	return fdProtos, nil
}

// processFileDescriptors processes the fetched file descriptors and recursively fetches their dependencies.
func processFileDescriptors(ctx context.Context, grpcClient *grpc.ClientConn, fdProtos []*descriptorpb.FileDescriptorProto, seen map[string]*descriptorpb.FileDescriptorProto, maxRetries uint) error {
	for _, fdProto := range fdProtos {
		name := fdProto.GetName()
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = fdProto

		// Recursively fetch dependencies
		for _, dep := range fdProto.Dependency {
			if _, exists := seen[dep]; !exists {
				err := fetchFileByName(ctx, grpcClient, dep, seen, maxRetries)
				if err != nil {
					return fmt.Errorf("failed to fetch dependency %s: %w", dep, err)
				}
			}
		}
	}
	return nil
}

func sendReflectionRequestWithRetry(ctx context.Context, grpcClient *grpc.ClientConn, req *reflection.ServerReflectionRequest, maxRetries uint) (*reflection.ServerReflectionResponse, error) {
	var resp *reflection.ServerReflectionResponse
	var err error
	for attempt := uint(1); attempt <= maxRetries; attempt++ {
		resp, err = sendReflectionRequest(ctx, grpcClient, req)
		if err == nil {
			return resp, nil
		}
		time.Sleep(time.Duration(2*attempt) * time.Second)
	}
	return nil, fmt.Errorf("failed to send reflection request after %d attempts: %w", maxRetries, err)
}

// sendReflectionRequest sends a reflection request and returns the response.
func sendReflectionRequest(ctx context.Context, grpcClient *grpc.ClientConn, req *reflection.ServerReflectionRequest) (*reflection.ServerReflectionResponse, error) {
	refClient := reflection.NewServerReflectionClient(grpcClient)
	stream, err := refClient.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reflection stream: %w", err)
	}
	defer stream.CloseSend()

	if err := stream.Send(req); err != nil {
		return nil, fmt.Errorf("failed to send reflection request: %w", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive reflection response: %w", err)
	}

	if err := checkErrorResponse(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// checkErrorResponse checks if the reflection response contains an error.
func checkErrorResponse(resp *reflection.ServerReflectionResponse) error {
	if errResp, ok := resp.MessageResponse.(*reflection.ServerReflectionResponse_ErrorResponse); ok {
		errInfo := errResp.ErrorResponse
		return fmt.Errorf("reflection error: %s (code: %d)", errInfo.ErrorMessage, errInfo.ErrorCode)
	}
	return nil
}
