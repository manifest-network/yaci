package reflection

import (
	"context"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/liftedinit/cosmos-dump/internal/client"
	"github.com/pkg/errors"
	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/types/descriptorpb"
)

// FetchAllDescriptors retrieves all file descriptors supported by the server.
func FetchAllDescriptors(ctx context.Context, grpcPool *client.GRPCClientPool, maxConcurrency uint64) ([]*descriptorpb.FileDescriptorProto, error) {
	seenFiles := make(map[string]*descriptorpb.FileDescriptorProto)
	var result []*descriptorpb.FileDescriptorProto

	// List all services
	services, err := listServices(ctx, grpcPool)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to list services")
	}

	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	errCh := make(chan error, len(services))

	for _, service := range services {
		service := service

		wg.Add(1)
		sem <- struct{}{} // Acquire a token

		go func() {
			defer wg.Done()
			defer func() { <-sem }() // Release the token

			err := fetchFileDescriptors(ctx, grpcPool, service, seenFiles, &mu)
			if err != nil {
				errCh <- errors.WithMessagef(err, "failed to fetch file descriptors for service %s", service)
			}
		}()
	}

	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		return nil, <-errCh
	}

	mu.Lock()
	for _, fd := range seenFiles {
		result = append(result, fd)
	}
	mu.Unlock()

	return result, nil
}

// listServices lists all services provided by the server via reflection.
func listServices(ctx context.Context, grpcPool *client.GRPCClientPool) ([]string, error) {
	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_ListServices{
			ListServices: "*",
		},
	}

	resp, err := sendReflectionRequest(ctx, grpcPool, req)
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
func fetchFileDescriptors(ctx context.Context, grpcPool *client.GRPCClientPool, symbol string, seen map[string]*descriptorpb.FileDescriptorProto, mu *sync.Mutex) error {
	mu.Lock()
	if _, exists := seen[symbol]; exists {
		mu.Unlock()
		return nil
	}
	mu.Unlock()

	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: symbol,
		},
	}

	fdProtos, err := fetchFileDescriptorsFromRequest(ctx, grpcPool, req)
	if err != nil {
		return errors.WithMessagef(err, "failed to fetch file descriptors containing symbol %s", symbol)
	}

	return processFileDescriptors(ctx, grpcPool, fdProtos, seen, mu)
}

// fetchFileByName fetches the file descriptor by filename and its dependencies.
func fetchFileByName(ctx context.Context, grpcPool *client.GRPCClientPool, name string, seen map[string]*descriptorpb.FileDescriptorProto, mu *sync.Mutex) error {
	mu.Lock()
	if _, exists := seen[name]; exists {
		mu.Unlock()
		return nil
	}
	mu.Unlock()

	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileByFilename{
			FileByFilename: name,
		},
	}

	fdProtos, err := fetchFileDescriptorsFromRequest(ctx, grpcPool, req)
	if err != nil {
		return errors.WithMessagef(err, "failed to fetch file descriptors for file %s", name)
	}

	return processFileDescriptors(ctx, grpcPool, fdProtos, seen, mu)
}

// fetchFileDescriptorsFromRequest sends a reflection request and returns the file descriptors.
func fetchFileDescriptorsFromRequest(ctx context.Context, grpcPool *client.GRPCClientPool, req *reflection.ServerReflectionRequest) ([]*descriptorpb.FileDescriptorProto, error) {
	resp, err := sendReflectionRequest(ctx, grpcPool, req)
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
func processFileDescriptors(ctx context.Context, grpcPool *client.GRPCClientPool, fdProtos []*descriptorpb.FileDescriptorProto, seen map[string]*descriptorpb.FileDescriptorProto, mu *sync.Mutex) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(fdProtos))

	for _, fdProto := range fdProtos {
		fdProto := fdProto
		wg.Add(1)

		go func() {
			defer wg.Done()

			name := fdProto.GetName()

			mu.Lock()
			if _, exists := seen[name]; exists {
				mu.Unlock()
				return
			}
			seen[name] = fdProto
			mu.Unlock()

			for _, dep := range fdProto.Dependency {
				mu.Lock()
				if _, exists := seen[dep]; !exists {
					mu.Unlock()
					if err := fetchFileByName(ctx, grpcPool, dep, seen, mu); err != nil {
						errCh <- errors.WithMessagef(err, "failed to fetch dependency %s", dep)
						return
					}
				} else {
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		return <-errCh
	}

	return nil
}

// sendReflectionRequest sends a reflection request and returns the response.
func sendReflectionRequest(ctx context.Context, grpcPool *client.GRPCClientPool, req *reflection.ServerReflectionRequest) (*reflection.ServerReflectionResponse, error) {
	_, refClient := grpcPool.GetConn()
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
