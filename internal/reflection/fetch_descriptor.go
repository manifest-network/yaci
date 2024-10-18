package reflection

import (
	"context"
	"fmt"

	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// FetchAllDescriptors retrieves all file descriptors supported by the server.
func FetchAllDescriptors(ctx context.Context, refClient reflection.ServerReflectionClient) ([]*descriptorpb.FileDescriptorProto, error) {
	seenFiles := make(map[string]*descriptorpb.FileDescriptorProto)
	var result []*descriptorpb.FileDescriptorProto

	// List all services
	services, err := listServices(ctx, refClient)
	if err != nil {
		return nil, err
	}

	// For each service, fetch its file descriptors
	for _, service := range services {
		err := fetchFileDescriptors(ctx, refClient, service, seenFiles)
		if err != nil {
			return nil, err
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

	stream, err := refClient.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}

	if err := stream.Send(req); err != nil {
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	listServicesResp, ok := resp.MessageResponse.(*reflection.ServerReflectionResponse_ListServicesResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response: %v", resp)
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

	stream, err := refClient.ServerReflectionInfo(ctx)
	if err != nil {
		return err
	}

	if err := stream.Send(req); err != nil {
		return err
	}

	resp, err := stream.Recv()
	if err != nil {
		return err
	}

	fdResponse, ok := resp.MessageResponse.(*reflection.ServerReflectionResponse_FileDescriptorResponse)
	if !ok {
		return fmt.Errorf("unexpected response: %v", resp)
	}

	for _, fdBytes := range fdResponse.FileDescriptorResponse.FileDescriptorProto {
		fdProto := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(fdBytes, fdProto); err != nil {
			return err
		}

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
					return err
				}
			}
		}
	}

	return nil
}

func fetchFileByName(ctx context.Context, refClient reflection.ServerReflectionClient, name string, seen map[string]*descriptorpb.FileDescriptorProto) error {
	if _, exists := seen[name]; exists {
		return nil
	}

	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileByFilename{
			FileByFilename: name,
		},
	}

	stream, err := refClient.ServerReflectionInfo(ctx)
	if err != nil {
		return err
	}

	if err := stream.Send(req); err != nil {
		return err
	}

	resp, err := stream.Recv()
	if err != nil {
		return err
	}

	fdResponse, ok := resp.MessageResponse.(*reflection.ServerReflectionResponse_FileDescriptorResponse)
	if !ok {
		return fmt.Errorf("unexpected response: %v", resp)
	}

	for _, fdBytes := range fdResponse.FileDescriptorResponse.FileDescriptorProto {
		fdProto := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(fdBytes, fdProto); err != nil {
			return err
		}

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
					return err
				}
			}
		}
	}

	return nil
}
