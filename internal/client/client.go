package client

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"
	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

// NewGRPCClients initializes the gRPC connection and reflection client.
func NewGRPCClients(ctx context.Context, address string) (*grpc.ClientConn, reflection.ServerReflectionClient) {
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}

	refClient := reflection.NewServerReflectionClient(conn)
	return conn, refClient
}
