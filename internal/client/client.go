package client

import (
	"context"
	"log/slog"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

// NewGRPCClients initializes the gRPC connection and reflection client.
func NewGRPCClients(ctx context.Context, address string, insecure bool) (*grpc.ClientConn, reflection.ServerReflectionClient) {
	var opts []grpc.DialOption
	if insecure {
		opts = append(opts, grpc.WithInsecure())
	} else {
		creds := credentials.NewClientTLSFromCert(nil, "")
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	conn, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		slog.Error("Failed to connect", "error", err)
		os.Exit(1)
	}

	refClient := reflection.NewServerReflectionClient(conn)
	return conn, refClient
}
