package reflection

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/types/descriptorpb"
)

// FetchAllDescriptors retrieves all file descriptors supported by the server.
func FetchAllDescriptors(ctx context.Context, refClient reflection.ServerReflectionClient) ([]*descriptorpb.FileDescriptorProto, error) {
	seenFiles := make(map[string]*descriptorpb.FileDescriptorProto)
	var result []*descriptorpb.FileDescriptorProto

	// List all services
	services, err := listServices(ctx, refClient)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to list services")
	}

	// For each service, fetch its file descriptors
	for _, service := range services {
		err := fetchFileDescriptors(ctx, refClient, service, seenFiles)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to fetch file descriptors for service %s", service)
		}
	}

	// Collect all descriptors
	for _, fd := range seenFiles {
		result = append(result, fd)
	}

	return result, nil
}

// listServices lists all services provided by the server via reflection.
func listServices(ctx context.Context, refClient reflection.ServerReflectionClient) ([]string, error) {
	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_ListServices{
			ListServices: "*",
		},
	}

	resp, err := sendReflectionRequest(ctx, refClient, req)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to list services via reflection")
	}

	listServicesResp, ok := resp.MessageResponse.(*reflection.ServerReflectionResponse_ListServicesResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", resp.MessageResponse)
	}

	var services []string
	for _, service := range listServicesResp.ListServicesResponse.Service {
		services = append(services, service.Name)
	}

	return services, nil
}

// fetchFileDescriptors fetches the file descriptors containing the given symbol and their dependencies.
func fetchFileDescriptors(ctx context.Context, refClient reflection.ServerReflectionClient, symbol string, seen map[string]*descriptorpb.FileDescriptorProto) error {
	if _, exists := seen[symbol]; exists {
		return nil
	}

	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: symbol,
		},
	}

	fdProtos, err := fetchFileDescriptorsFromRequest(ctx, refClient, req)
	if err != nil {
		return errors.WithMessagef(err, "failed to fetch file descriptors containing symbol %s", symbol)
	}

	return processFileDescriptors(ctx, refClient, fdProtos, seen)
}

// fetchFileByName fetches the file descriptor by filename and its dependencies.
func fetchFileByName(ctx context.Context, refClient reflection.ServerReflectionClient, name string, seen map[string]*descriptorpb.FileDescriptorProto) error {
	if _, exists := seen[name]; exists {
		return nil
	}

	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileByFilename{
			FileByFilename: name,
		},
	}

	fdProtos, err := fetchFileDescriptorsFromRequest(ctx, refClient, req)
	if err != nil {
		return errors.WithMessagef(err, "failed to fetch file descriptors for file %s", name)
	}

	return processFileDescriptors(ctx, refClient, fdProtos, seen)
}

// fetchFileDescriptorsFromRequest sends a reflection request and returns the file descriptors.
func fetchFileDescriptorsFromRequest(ctx context.Context, refClient reflection.ServerReflectionClient, req *reflection.ServerReflectionRequest) ([]*descriptorpb.FileDescriptorProto, error) {
	resp, err := sendReflectionRequest(ctx, refClient, req)
	if err != nil {
		return nil, err
	}

	fdResponse, ok := resp.MessageResponse.(*reflection.ServerReflectionResponse_FileDescriptorResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", resp.MessageResponse)
	}

	var fdProtos []*descriptorpb.FileDescriptorProto
	for _, fdBytes := range fdResponse.FileDescriptorResponse.FileDescriptorProto {
		fdProto := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(fdBytes, fdProto); err != nil {
			return nil, errors.WithMessage(err, "failed to unmarshal file descriptor")
		}
		fdProtos = append(fdProtos, fdProto)
	}

	return fdProtos, nil
}

// processFileDescriptors processes the fetched file descriptors and recursively fetches their dependencies.
func processFileDescriptors(ctx context.Context, refClient reflection.ServerReflectionClient, fdProtos []*descriptorpb.FileDescriptorProto, seen map[string]*descriptorpb.FileDescriptorProto) error {
	for _, fdProto := range fdProtos {
		name := fdProto.GetName()
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = fdProto

		// Recursively fetch dependencies
		for _, dep := range fdProto.Dependency {
			if _, exists := seen[dep]; !exists {
				err := fetchFileByName(ctx, refClient, dep, seen)
				if err != nil {
					return errors.WithMessagef(err, "failed to fetch dependency %s", dep)
				}
			}
		}
	}
	return nil
}

// sendReflectionRequest sends a reflection request and returns the response.
func sendReflectionRequest(ctx context.Context, refClient reflection.ServerReflectionClient, req *reflection.ServerReflectionRequest) (*reflection.ServerReflectionResponse, error) {
	stream, err := refClient.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create reflection stream")
	}
	defer stream.CloseSend()

	if err := stream.Send(req); err != nil {
		return nil, errors.WithMessage(err, "failed to send reflection request")
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to receive reflection response")
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
