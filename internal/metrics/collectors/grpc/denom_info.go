package grpc

import (
	"log/slog"
	"strconv"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	"github.com/liftedinit/yaci/internal/client"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DenomInfoCollector collects denom metadata and total supply metrics from the Cosmos SDK bank module via gRPC.
// Initialize the collector with the denom you want to monitor.
type DenomInfoCollector struct {
	grpcClient      *client.GRPCClient
	denom           string
	denomInfoDesc   *prometheus.Desc // Denom metadata
	upDesc          *prometheus.Desc // gRPC query success
	totalSupplyDesc *prometheus.Desc // Token supply
	initialError    error
}

// NewDenomInfoCollector creates a new DenomInfoCollector.
// It requires a gRPC client connection to query the bank module.
func NewDenomInfoCollector(client *client.GRPCClient, denom string) *DenomInfoCollector {
	var initialError error
	if client == nil {
		initialError = status.Error(codes.Internal, "gRPC client is nil")
	}
	if client != nil && client.Conn == nil {
		initialError = status.Error(codes.Internal, "gRPC client connection is nil")
	}
	if denom == "" {
		initialError = status.Error(codes.InvalidArgument, "denom is empty")
	}

	return &DenomInfoCollector{
		grpcClient:   client,
		initialError: initialError,
		denom:        denom,
		denomInfoDesc: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "tokenomics", "denom_info"),
			"Information about a Cosmos SDK denomination.",
			[]string{"symbol", "denom", "name", "display"},
			prometheus.Labels{"source": "grpc"},
		),
		totalSupplyDesc: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "tokenomics", "total_supply"),
			"Total supply of a specific denomination.",
			[]string{"denom"},
			prometheus.Labels{"source": "grpc"},
		),
		upDesc: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "tokenomics", "denom_grpc_up"),
			"Whether the gRPC query was successful.",
			nil,
			prometheus.Labels{"source": "grpc", "queries": "DenomMetadata, SupplyOf"},
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (c *DenomInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.denomInfoDesc
	ch <- c.totalSupplyDesc
	ch <- c.upDesc
}

// Collect implements the prometheus.Collector interface.
func (c *DenomInfoCollector) Collect(ch chan<- prometheus.Metric) {
	// Check for initialization or connection errors first.
	if err := validateClient(c.grpcClient, c.initialError); err != nil {
		reportUpMetric(ch, c.upDesc, 0) // Report gRPC down
		reportInvalidMetric(ch, c.totalSupplyDesc, err)
		reportInvalidMetric(ch, c.denomInfoDesc, err)
		return
	}

	bankQueryClient := bankv1beta1.NewQueryClient(c.grpcClient.Conn)
	denomMetaResp, denomMetaErr := bankQueryClient.DenomMetadata(c.grpcClient.Ctx, &bankv1beta1.QueryDenomMetadataRequest{Denom: c.denom})
	if denomMetaErr != nil {
		slog.Error("Failed to query via gRPC", "query", "DenomMetadata", "error", denomMetaErr)
	}

	totalSupplyResp, totalSupplyErr := bankQueryClient.SupplyOf(c.grpcClient.Ctx, &bankv1beta1.QuerySupplyOfRequest{Denom: c.denom})
	if totalSupplyErr != nil {
		slog.Error("Failed to query via gRPC", "query", "SupplyOf", "error", totalSupplyErr)
	}

	// Report 'up' metric based on query success
	upValue := 0.0
	if denomMetaErr == nil && totalSupplyErr == nil {
		upValue = 1.0
	}
	reportUpMetric(ch, c.upDesc, upValue)

	c.collectDenomMetadata(ch, denomMetaResp, denomMetaErr)
	c.collectTotalSupply(ch, totalSupplyResp, totalSupplyErr)
}

func (c *DenomInfoCollector) collectDenomMetadata(ch chan<- prometheus.Metric, resp *bankv1beta1.QueryDenomMetadataResponse, queryErr error) {
	if queryErr != nil {
		reportInvalidMetric(ch, c.denomInfoDesc, queryErr)
		return
	}
	if resp == nil {
		return
	}

	metadata := resp.Metadata

	if metadata != nil {
		metric, err := prometheus.NewConstMetric(
			c.denomInfoDesc,
			prometheus.GaugeValue,
			1, // Value is 1 to indicate presence/info
			metadata.Symbol,
			metadata.Base,
			metadata.Name,
			metadata.Display,
		)
		if err != nil {
			slog.Error("Failed to create denom metadata metric", "symbol", metadata.Symbol, "base", metadata.Base, "error", err)
		} else {
			ch <- metric
		}
	}
}

func (c *DenomInfoCollector) collectTotalSupply(ch chan<- prometheus.Metric, resp *bankv1beta1.QuerySupplyOfResponse, queryErr error) {
	if queryErr != nil {
		reportInvalidMetric(ch, c.totalSupplyDesc, queryErr)
		return
	}
	if resp == nil {
		return
	}
	coin := resp.Amount
	if coin == nil {
		slog.Warn("Total supply response is nil")
		reportInvalidMetric(ch, c.totalSupplyDesc, status.Error(codes.Internal, "total supply response is nil"))
		return
	}

	amount, err := strconv.ParseFloat(coin.Amount, 64)
	if err != nil {
		parseErr := status.Errorf(codes.Internal, "failed to parse amount '%s' for denom '%s': %v", coin.Amount, coin.Denom, err)
		slog.Warn("Failed to parse total supply amount", "denom", coin.Denom, "amount", coin.Amount, "error", err)
		reportInvalidMetric(ch, c.totalSupplyDesc, parseErr)
	}

	metric, err := prometheus.NewConstMetric(
		c.totalSupplyDesc,
		prometheus.GaugeValue,
		amount,
		coin.Denom,
	)
	if err != nil {
		slog.Error("Failed to create total supply metric", "denom", coin.Denom, "error", err)
	} else {
		ch <- metric
	}
}
