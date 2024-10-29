package extractor

import (
	"context"
	"fmt"
	"time"

	"github.com/liftedinit/yaci/internal/output"
	"github.com/liftedinit/yaci/internal/reflection"
	"github.com/liftedinit/yaci/internal/utils"
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
				return fmt.Errorf("failed to get latest block height: %w", err)
			}

			if latestHeight > currentHeight {
				err = ExtractBlocksAndTransactions(ctx, grpcConn, resolver, currentHeight+1, latestHeight, outputHandler, maxConcurrency, maxRetries)
				if err != nil {
					return fmt.Errorf("failed to process blocks and transactions: %w", err)
				}
				currentHeight = latestHeight
			}

			// Sleep before checking again
			time.Sleep(time.Duration(blockTime) * time.Second)
		}
	}
}
