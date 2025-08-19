package extractor

import (
	"fmt"
	"log/slog"

	"github.com/manifest-network/yaci/internal/client"
	"github.com/manifest-network/yaci/internal/config"
	"github.com/manifest-network/yaci/internal/output"
	"github.com/manifest-network/yaci/internal/utils"
)

const (
	blockMethodFullName = "cosmos.tx.v1beta1.Service.GetBlockWithTxs"
	txMethodFullName    = "cosmos.tx.v1beta1.Service.GetTx"
)

// Extract extracts blocks and transactions from a gRPC server.
func Extract(gRPCClient *client.GRPCClient, outputHandler output.OutputHandler, config config.ExtractConfig) error {
	// Check if the missing block check should be skipped before setting the block range
	skipMissingBlockCheck := shouldSkipMissingBlockCheck(config)

	if err := setBlockRange(gRPCClient, outputHandler, &config); err != nil {
		return err
	}

	if !skipMissingBlockCheck {
		if err := processMissingBlocks(gRPCClient, outputHandler, config); err != nil {
			return err
		}
	}

	if config.LiveMonitoring {
		slog.Info("Starting live extraction", "block_time", config.BlockTime)
		err := extractLiveBlocksAndTransactions(gRPCClient, config.BlockStart, outputHandler, config.BlockTime, config.MaxConcurrency, config.MaxRetries)
		if err != nil {
			return fmt.Errorf("failed to process live blocks and transactions: %w", err)
		}
	} else {
		slog.Info("Starting extraction", "start", config.BlockStart, "stop", config.BlockStop)
		err := extractBlocksAndTransactions(gRPCClient, config.BlockStart, config.BlockStop, outputHandler, config.MaxConcurrency, config.MaxRetries)
		if err != nil {
			return fmt.Errorf("failed to process blocks and transactions: %w", err)
		}
	}

	return nil
}

// setBlockRange sets correct the block range based on the configuration.
// If the start block is not set, it will be set to the latest block in the database.
// If the stop block is not set, it will be set to the latest block in the gRPC server.
// If the start block is greater than the stop block, an error will be returned.
func setBlockRange(gRPCClient *client.GRPCClient, outputHandler output.OutputHandler, cfg *config.ExtractConfig) error {
	if cfg.ReIndex {
		slog.Info("Reindexing entire database...")
		// TODO: Get the earliest block from the gRPC server
		// See https://github.com/manifest-network/yaci/issues/28
		cfg.BlockStart = 1
		earliestLocalBlock, err := outputHandler.GetEarliestBlock(gRPCClient.Ctx)
		if err != nil {
			return fmt.Errorf("failed to get the earliest local block: %w", err)
		}
		if earliestLocalBlock != nil {
			cfg.BlockStart = earliestLocalBlock.ID
		}
		cfg.BlockStop = 0
	}

	if cfg.BlockStart == 0 {
		// TODO: Get the earliest block from the gRPC server
		// See https://github.com/manifest-network/yaci/issues/28
		cfg.BlockStart = 1
		latestLocalBlock, err := outputHandler.GetLatestBlock(gRPCClient.Ctx)
		if err != nil {
			return fmt.Errorf("failed to get the latest block: %w", err)
		}
		if latestLocalBlock != nil {
			cfg.BlockStart = latestLocalBlock.ID + 1
		}
	}

	if cfg.BlockStop == 0 {
		latestRemoteBlock, err := utils.GetLatestBlockHeightWithRetry(gRPCClient, cfg.MaxRetries)
		if err != nil {
			return fmt.Errorf("failed to get the latest block: %w", err)
		}
		cfg.BlockStop = latestRemoteBlock
	}

	if cfg.BlockStart > cfg.BlockStop {
		return fmt.Errorf("start block is greater than stop block")
	}

	return nil
}

// shouldSkipMissingBlockCheck returns true if the missing block check should be skipped.
func shouldSkipMissingBlockCheck(cfg config.ExtractConfig) bool {
	return (cfg.BlockStart != 0 && cfg.BlockStop != 0) || cfg.ReIndex
}
