package extractor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/liftedinit/yaci/internal/client"
	"github.com/liftedinit/yaci/internal/config"
	"github.com/liftedinit/yaci/internal/output"
	"github.com/liftedinit/yaci/internal/reflection"
	"github.com/liftedinit/yaci/internal/utils"
)

// Extract extracts blocks and transactions from a gRPC server.
func Extract(address string, outputHandler output.OutputHandler, config config.ExtractConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handleInterrupt(cancel)

	grpcConn, resolver, err := initializeGRPC(ctx, address, config)
	if err != nil {
		return fmt.Errorf("failed to initialize gRPC: %w", err)
	}
	defer grpcConn.Close()

	// Check if the missing block check should be skipped before setting the block range
	skipMissingBlockCheck := shouldSkipMissingBlockCheck(config)

	if err := setBlockRange(ctx, grpcConn, resolver, outputHandler, &config); err != nil {
		return err
	}

	if !skipMissingBlockCheck {
		if err := processMissingBlocks(ctx, grpcConn, resolver, outputHandler, config); err != nil {
			return err
		}
	}

	if config.LiveMonitoring {
		slog.Info("Starting live extraction", "block_time", config.BlockTime)
		err = ExtractLiveBlocksAndTransactions(ctx, grpcConn, resolver, config.BlockStart, outputHandler, config.BlockTime, config.MaxConcurrency, config.MaxRetries)
		if err != nil {
			return fmt.Errorf("failed to process live blocks and transactions: %w", err)
		}
	} else {
		slog.Info("Starting extraction", "start", config.BlockStart, "stop", config.BlockStop)
		err = ExtractBlocksAndTransactions(ctx, grpcConn, resolver, config.BlockStart, config.BlockStop, outputHandler, config.MaxConcurrency, config.MaxRetries)
		if err != nil {
			return fmt.Errorf("failed to process blocks and transactions: %w", err)
		}
	}

	return nil
}

// handleInterrupt handles interrupt signals for graceful shutdown.
func handleInterrupt(cancel context.CancelFunc) {
	// Handle interrupt signals for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		slog.Info("Received interrupt signal, shutting down...")
		cancel()
	}()
}

// initializeGRPC initializes the gRPC client, fetches protocol buffer descriptors & creates the PB resolver.
func initializeGRPC(ctx context.Context, address string, cfg config.ExtractConfig) (*grpc.ClientConn, *reflection.CustomResolver, error) {
	slog.Info("Initializing gRPC client pool...")
	grpcConn := client.NewGRPCClients(ctx, address, cfg.Insecure)

	slog.Info("Fetching protocol buffer descriptors from gRPC server... This may take a while.")
	descriptors, err := reflection.FetchAllDescriptors(ctx, grpcConn, cfg.MaxRetries)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch descriptors: %w", err)
	}

	slog.Info("Building protocol buffer descriptor set...")
	files, err := reflection.BuildFileDescriptorSet(descriptors)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build descriptor set: %w", err)
	}

	resolver := reflection.NewCustomResolver(files, grpcConn, ctx, cfg.MaxRetries)
	return grpcConn, resolver, nil
}

// setBlockRange sets correct the block range based on the configuration.
// If the start block is not set, it will be set to the latest block in the database.
// If the stop block is not set, it will be set to the latest block in the gRPC server.
// If the start block is greater than the stop block, an error will be returned.
func setBlockRange(ctx context.Context, grpcConn *grpc.ClientConn, resolver *reflection.CustomResolver, outputHandler output.OutputHandler, cfg *config.ExtractConfig) error {
	if cfg.ReIndex {
		slog.Info("Reindexing entire database...")
		cfg.BlockStart = 1
		cfg.BlockStop = 0
	}

	if cfg.BlockStart == 0 {
		cfg.BlockStart = 1
		latestLocalBlock, err := outputHandler.GetLatestBlock(ctx)
		if err != nil {
			return fmt.Errorf("failed to get the latest block: %w", err)
		}
		if latestLocalBlock != nil {
			cfg.BlockStart = latestLocalBlock.ID + 1
		}
	}

	if cfg.BlockStop == 0 {
		latestRemoteBlock, err := utils.GetLatestBlockHeightWithRetry(ctx, grpcConn, resolver, cfg.MaxRetries)
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

// processMissingBlocks processes missing blocks by fetching them from the gRPC server.
func processMissingBlocks(ctx context.Context, grpcConn *grpc.ClientConn, resolver *reflection.CustomResolver, outputHandler output.OutputHandler, cfg config.ExtractConfig) error {
	missingBlockIds, err := outputHandler.GetMissingBlockIds(ctx)
	if err != nil {
		return fmt.Errorf("failed to get missing block IDs: %w", err)
	}

	if len(missingBlockIds) > 0 {
		slog.Warn("Missing blocks detected", "count", len(missingBlockIds), "blocks", missingBlockIds)
		for _, blockID := range missingBlockIds {
			if err := ProcessSingleBlockWithRetry(ctx, grpcConn, resolver, blockID, outputHandler, cfg.MaxRetries); err != nil {
				return fmt.Errorf("failed to process missing block %d: %w", blockID, err)
			}
		}
	}
	return nil
}
