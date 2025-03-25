package utils

import (
	"github.com/liftedinit/yaci/internal/client"
)

const bech32PrefixMethod = "cosmos.auth.v1beta1.Query.Bech32Prefix"

// GetBech32PrefixWithRetry retrieves the Bech32 prefix from the gRPC server with retry logic.
func GetBech32PrefixWithRetry(gRPCClient *client.GRPCClient, maxRetries uint) (string, error) {
	return ExtractGRPCField(
		gRPCClient,
		bech32PrefixMethod,
		maxRetries,
		"bech32_prefix",
		func(s string) (string, error) {
			return s, nil
		},
	)
}
