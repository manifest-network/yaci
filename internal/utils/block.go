package utils

import (
	"strconv"

	"github.com/liftedinit/yaci/internal/client"
	"github.com/pkg/errors"
)

const statusMethod = "cosmos.base.node.v1beta1.Service.Status"

// GetLatestBlockHeightWithRetry retrieves the latest block height from the gRPC server with retry logic.
func GetLatestBlockHeightWithRetry(gRPCClient *client.GRPCClient, maxRetries uint) (uint64, error) {
	return ExtractGRPCField(
		gRPCClient,
		statusMethod,
		maxRetries,
		"height",
		func(s string) (uint64, error) {
			height, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return 0, errors.WithMessage(err, "error parsing height")
			}
			return height, nil
		},
	)
}
