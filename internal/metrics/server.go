package metrics

import (
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/liftedinit/yaci/internal/client"
	grpcPromCollectors "github.com/liftedinit/yaci/internal/metrics/collectors/grpc"
	sqlPromCollectors "github.com/liftedinit/yaci/internal/metrics/collectors/sql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func CreateMetricsServer(db *sql.DB, grpcClient *client.GRPCClient, bech32Prefix, addr string) (*http.Server, error) {
	if db == nil {
		return nil, errors.New("database connection is nil")
	}

	if bech32Prefix == "" {
		return nil, errors.New("bech32 prefix is empty")
	}

	if addr == "" {
		return nil, errors.New("address is empty")
	}

	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.New("invalid address format")
	}

	port, err := net.LookupPort("tcp", portStr)
	if err != nil || port < 1 || port > 65535 {
		return nil, errors.New("invalid port number")
	}

	sqlCollectors, err := sqlPromCollectors.DefaultSqlRegistry.CreateSqlCollectors(db, bech32Prefix)
	if err != nil {
		slog.Error("Failed to create SQL collectors", "error", err)
		sqlCollectors = []prometheus.Collector{}
	}

	grpcCollectors, err := grpcPromCollectors.DefaultGrpcRegistry.CreateGrpcCollectors(grpcClient)
	if err != nil {
		// Allow server to start even if gRPC collectors fail, but log the error.
		slog.Error("Failed to create gRPC collectors", "error", err)
		grpcCollectors = []prometheus.Collector{} // Ensure slice is not nil
	}

	allCollectors := append(sqlCollectors, grpcCollectors...)

	for _, c := range allCollectors {
		if err := prometheus.Register(c); err != nil {
			var are prometheus.AlreadyRegisteredError
			if errors.As(err, &are) {
				slog.Info("Collector already registered", "collector", are.ExistingCollector)
			}
		}
	}

	server, errChan := listen(addr)

	select {
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
	default:
	}

	return server, nil
}

func listen(addr string) (*http.Server, chan error) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{Addr: addr, Handler: mux}
	errChan := make(chan error)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				slog.Error("Failed to start metrics server", "error", err)
				errChan <- err
			} else {
				slog.Info("Metrics server closed")
			}
		}
	}()

	return server, errChan
}
