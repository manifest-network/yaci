package extractor

import (
	"fmt"
	"time"

	"github.com/liftedinit/yaci/internal/client"
	"github.com/liftedinit/yaci/internal/output"
	"github.com/liftedinit/yaci/internal/utils"
)

// extractLiveBlocksAndTransactions monitors the chain and processes new blocks as they are produced.
func extractLiveBlocksAndTransactions(gRPCClient *client.GRPCClient, start uint64, outputHandler output.OutputHandler, blockTime, maxConcurrency, maxRetries uint) error {
	currentHeight := start - 1
	for {
		select {
		case <-gRPCClient.Ctx.Done():
			return nil
		default:
			// Get the latest block height
			latestHeight, err := utils.GetLatestBlockHeightWithRetry(gRPCClient, maxRetries)
			if err != nil {
				return fmt.Errorf("failed to get latest block height: %w", err)
			}

			if latestHeight > currentHeight {
				err = extractBlocksAndTransactions(gRPCClient, currentHeight+1, latestHeight, outputHandler, maxConcurrency, maxRetries)
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
