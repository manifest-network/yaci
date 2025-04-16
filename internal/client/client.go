package client

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/liftedinit/yaci/internal/reflection"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

var keepaliveParams = keepalive.ClientParameters{
	Time:                60 * time.Second,
	Timeout:             30 * time.Second,
	PermitWithoutStream: true,
}

type GRPCClient struct {
	Ctx      context.Context
	Conn     *grpc.ClientConn
	Resolver *reflection.CustomResolver
}

func NewGRPCClient(ctx context.Context, address string, insecure bool, maxCallRecvMsgSize int) (*GRPCClient, error) {
	slog.Info("Initializing gRPC client pool...")
	conn := dial(ctx, address, insecure, maxCallRecvMsgSize)

	slog.Info("Fetching protocol buffer descriptors from gRPC server... This may take a while.")
	descriptors, err := reflection.FetchAllDescriptors(ctx, conn, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch descriptors: %w", err)
	}

	slog.Info("Building protocol buffer descriptor set...")
	files, err := reflection.BuildFileDescriptorSet(descriptors)
	if err != nil {
		return nil, fmt.Errorf("failed to build descriptor set: %w", err)
	}

	resolver := reflection.NewCustomResolver(ctx, files, conn, 3)

	return &GRPCClient{
		Ctx:      ctx,
		Conn:     conn,
		Resolver: resolver,
	}, nil
}

func dial(ctx context.Context, address string, insecure bool, maxCallRecvMsgSize int) *grpc.ClientConn {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithKeepaliveParams(keepaliveParams))
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxCallRecvMsgSize)))
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

	return conn
}
