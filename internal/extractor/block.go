package extractor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/manifest-network/yaci/internal/client"
	"github.com/manifest-network/yaci/internal/config"
	"github.com/manifest-network/yaci/internal/models"
	"github.com/manifest-network/yaci/internal/output"
	"github.com/manifest-network/yaci/internal/utils"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
)

// extractBlocksAndTransactions extracts blocks and transactions from the gRPC server.
func extractBlocksAndTransactions(gRPCClient *client.GRPCClient, start, stop uint64, outputHandler output.OutputHandler, maxConcurrency, maxRetries uint) error {
	displayProgress := start != stop
	if displayProgress {
		slog.Info("Extracting blocks and transactions", "range", fmt.Sprintf("[%d, %d]", start, stop))
	} else {
		slog.Info("Extracting blocks and transactions", "height", start)
	}
	var bar *progressbar.ProgressBar
	if displayProgress {
		bar = progressbar.NewOptions64(
			int64(stop-start+1),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionSetDescription("Processing blocks..."),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "=",
				SaucerHead:    ">",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)
		if err := bar.RenderBlank(); err != nil {
			return fmt.Errorf("failed to render progress bar: %w", err)
		}
	}

	if err := processBlocks(gRPCClient, start, stop, outputHandler, maxConcurrency, maxRetries, bar); err != nil {
		return fmt.Errorf("failed to process blocks and transactions: %w", err)
	}

	if bar != nil {
		if err := bar.Finish(); err != nil {
			return fmt.Errorf("failed to finish progress bar: %w", err)
		}
	}

	return nil
}

// processMissingBlocks processes missing blocks by fetching them from the gRPC server.
func processMissingBlocks(gRPCClient *client.GRPCClient, outputHandler output.OutputHandler, cfg config.ExtractConfig) error {
	missingBlockIds, err := outputHandler.GetMissingBlockIds(gRPCClient.Ctx)
	if err != nil {
		return fmt.Errorf("failed to get missing block IDs: %w", err)
	}

	if len(missingBlockIds) > 0 {
		slog.Warn("Missing blocks detected", "count", len(missingBlockIds))
		for _, blockID := range missingBlockIds {
			if err := processSingleBlockWithRetry(gRPCClient, blockID, outputHandler, cfg.MaxRetries); err != nil {
				return fmt.Errorf("failed to process missing block %d: %w", blockID, err)
			}
		}
	}
	return nil
}

// processBlocks processes blocks in parallel using goroutines.
func processBlocks(gRPCClient *client.GRPCClient, start, stop uint64, outputHandler output.OutputHandler, maxConcurrency, maxRetries uint, bar *progressbar.ProgressBar) error {
	eg, ctx := errgroup.WithContext(gRPCClient.Ctx)
	sem := make(chan struct{}, maxConcurrency)

	for height := start; height <= stop; height++ {
		if ctx.Err() != nil {
			slog.Info("Processing cancelled by user")
			return ctx.Err()
		}

		blockHeight := height
		sem <- struct{}{}

		clientWithCtx := &client.GRPCClient{
			Conn:     gRPCClient.Conn,
			Ctx:      ctx,
			Resolver: gRPCClient.Resolver,
		}

		eg.Go(func() error {
			defer func() { <-sem }()

			err := processSingleBlockWithRetry(clientWithCtx, blockHeight, outputHandler, maxRetries)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					slog.Error("Block processing error",
						"height", blockHeight,
						"error", err,
						"errorType", fmt.Sprintf("%T", err))
					return err
				}
				slog.Error("Failed to process block", "height", blockHeight, "error", err, "retries", maxRetries)
				return fmt.Errorf("failed to process block %d: %w", blockHeight, err)
			}

			if bar != nil {
				if err := bar.Add(1); err != nil {
					slog.Warn("Failed to update progress bar", "error", err)
				}
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("error while fetching blocks: %w", err)
	}
	return nil
}

// processSingleBlockWithRetry fetches a block and its transactions from the gRPC server with retries.
// It unmarshals the block data and writes it to the output handler.
func processSingleBlockWithRetry(gRPCClient *client.GRPCClient, blockHeight uint64, outputHandler output.OutputHandler, maxRetries uint) error {
	blockJsonParams := []byte(fmt.Sprintf(`{"height": %d}`, blockHeight))

	// Get block data with retries
	blockJsonBytes, err := utils.GetGRPCResponse(
		gRPCClient,
		blockMethodFullName,
		maxRetries,
		blockJsonParams,
	)
	if err != nil {
		return fmt.Errorf("failed to get block data: %w", err)
	}

	// Create block model
	block := &models.Block{
		ID:   blockHeight,
		Data: blockJsonBytes,
	}

	var data map[string]interface{}
	if err := json.Unmarshal(blockJsonBytes, &data); err != nil {
		return fmt.Errorf("failed to unmarshal block JSON: %w", err)
	}

	transactions, err := extractTransactions(gRPCClient, data, maxRetries)
	if err != nil {
		return fmt.Errorf("failed to extract transactions from block: %w", err)
	}

	// Write block with transactions to the output handler
	err = outputHandler.WriteBlockWithTransactions(gRPCClient.Ctx, block, transactions)
	if err != nil {
		return fmt.Errorf("failed to write block with transactions: %w", err)
	}

	return nil
}
