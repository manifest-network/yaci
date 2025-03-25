package utils

import (
	"strconv"

	"github.com/liftedinit/yaci/internal/client"
	"github.com/pkg/errors"
)

func GetLatestBlockHeightWithRetry(gRPCClient *client.GRPCClient, maxRetries uint) (uint64, error) {
	return ExtractGRPCField(
		gRPCClient,
		"cosmos.base.node.v1beta1.Service.Status",
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
