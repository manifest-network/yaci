package utils

import (
	"github.com/liftedinit/yaci/internal/client"
)

func GetBech32PrefixWithRetry(gRPCClient *client.GRPCClient, maxRetries uint) (string, error) {
	return ExtractGRPCField(
		gRPCClient,
		"cosmos.auth.v1beta1.Query.Bech32Prefix",
		maxRetries,
		"bech32_prefix",
		func(s string) (string, error) {
			return s, nil
		},
	)
}
