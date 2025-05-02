package grpc

import (
	"errors"

	"github.com/liftedinit/yaci/internal/client"
	"github.com/prometheus/client_golang/prometheus"
)

type GrpcCollectorFactory func(client *client.GRPCClient, extraParams ...interface{}) (prometheus.Collector, error)

// GrpcRegistry holds factories for gRPC-based collectors.
type GrpcRegistry struct {
	factories []GrpcCollectorFactory
}

// NewGrpcRegistry creates a new registry for gRPC collectors.
func NewGrpcRegistry() *GrpcRegistry {
	return &GrpcRegistry{
		factories: make([]GrpcCollectorFactory, 0),
	}
}

// Register adds a new gRPC collector factory.
func (r *GrpcRegistry) Register(factory GrpcCollectorFactory) {
	r.factories = append(r.factories, factory)
}

// CreateGrpcCollectors instantiates all registered gRPC collectors.
func (r *GrpcRegistry) CreateGrpcCollectors(client *client.GRPCClient, extraParams ...interface{}) ([]prometheus.Collector, error) {
	if client == nil {
		return nil, errors.New("gRPC client is nil")
	}
	if client.Conn == nil {
		return nil, errors.New("gRPC client connection is nil for gRPC collectors")
	}

	collectors := make([]prometheus.Collector, 0, len(r.factories))
	for _, factory := range r.factories {
		collector, err := factory(client, extraParams...)
		if err != nil {
			// Consider logging the specific factory that failed
			return nil, err
		}
		collectors = append(collectors, collector)
	}
	return collectors, nil
}

// DefaultGrpcRegistry is the default registry instance for gRPC collectors.
var DefaultGrpcRegistry = NewGrpcRegistry()

// RegisterGrpcCollectorFactory registers a factory with the default gRPC registry.
func RegisterGrpcCollectorFactory(factory GrpcCollectorFactory) {
	DefaultGrpcRegistry.Register(factory)
}
