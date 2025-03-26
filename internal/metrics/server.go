package metrics

import (
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/liftedinit/yaci/internal/metrics/collectors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func CreateMetricsServer(db *sql.DB, bech32Prefix, addr string) (*http.Server, error) {
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

	allCollectors := []prometheus.Collector{
		collectors.NewTotalTransactionCountCollector(db),
		collectors.NewTotalUniqueAddressesCollector(db, bech32Prefix)}
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
			slog.Error("Failed to start metrics server", "error", err)
			errChan <- err
		}
	}()

	return server, errChan
}
