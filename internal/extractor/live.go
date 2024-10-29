package extractor

import (
	"context"
	"time"

	"github.com/liftedinit/yaci/internal/utils"
	"github.com/pkg/errors"

	"github.com/liftedinit/yaci/internal/output"
	"github.com/liftedinit/yaci/internal/reflection"

	"google.golang.org/grpc"
)

// ExtractLiveBlocksAndTransactions monitors the chain and processes new blocks as they are produced.
func ExtractLiveBlocksAndTransactions(ctx context.Context, grpcConn *grpc.ClientConn, resolver *reflection.CustomResolver, start uint64, outputHandler output.OutputHandler, blockTime, maxConcurrency, maxRetries uint) error {
	currentHeight := start
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Get the latest block height
			latestHeight, err := utils.GetLatestBlockHeightWithRetry(ctx, grpcConn, resolver, maxRetries)
			if err != nil {
				return errors.WithMessage(err, "Failed to get latest block height")
			}

			if latestHeight > currentHeight {
				err = ExtractBlocksAndTransactions(ctx, grpcConn, resolver, currentHeight+1, latestHeight, outputHandler, maxConcurrency, maxRetries)
				if err != nil {
					return errors.WithMessage(err, "Failed to process blocks and transactions")
				}
				currentHeight = latestHeight
			}

			// Sleep before checking again
			time.Sleep(time.Duration(blockTime) * time.Second)
		}
	}
}
