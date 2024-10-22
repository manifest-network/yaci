package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

type GRPCClientPool struct {
	conns      []*grpc.ClientConn
	refClients []reflection.ServerReflectionClient
	size       int
	mu         sync.Mutex
	index      int
}

var keepaliveParams = keepalive.ClientParameters{
	Time:                60 * time.Second,
	Timeout:             30 * time.Second,
	PermitWithoutStream: true,
}

func NewGRPCClientPool(ctx context.Context, address string, insecure bool, poolSize int) (*GRPCClientPool, error) {
	pool := &GRPCClientPool{size: poolSize}

	for i := 0; i < poolSize; i++ {
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithKeepaliveParams(keepaliveParams))
		if insecure {
			opts = append(opts, grpc.WithInsecure())
		} else {
			creds := credentials.NewClientTLSFromCert(nil, "")
			opts = append(opts, grpc.WithTransportCredentials(creds))
		}

		// Add a custom channel ID to prevent re-use
		uniqueDialOption := grpc.WithUserAgent(fmt.Sprintf("grpc-client-%d", i))
		opts = append(opts, uniqueDialOption)

		conn, err := grpc.DialContext(ctx, address, opts...)
		if err != nil {
			// Close any previously opened connections
			for _, c := range pool.conns {
				c.Close()
			}
			return nil, fmt.Errorf("failed to connect: %v", err)
		}

		refClient := reflection.NewServerReflectionClient(conn)
		pool.conns = append(pool.conns, conn)
		pool.refClients = append(pool.refClients, refClient)
	}

	return pool, nil
}

func (p *GRPCClientPool) GetConn() (*grpc.ClientConn, reflection.ServerReflectionClient) {
	p.mu.Lock()
	defer p.mu.Unlock()
	conn := p.conns[p.index]
	refClient := p.refClients[p.index]
	p.index = (p.index + 1) % p.size
	return conn, refClient
}

func (p *GRPCClientPool) Close() {
	for _, conn := range p.conns {
		conn.Close()
	}
}
