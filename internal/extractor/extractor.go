package extractor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

	// Handle interrupt signals for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		slog.Info("Received interrupt signal, shutting down...")
		cancel()
	}()

	slog.Info("Initializing gRPC client pool...")
	grpcConn := client.NewGRPCClients(ctx, address, config.Insecure)
	defer grpcConn.Close()

	slog.Info("Fetching protocol buffer descriptors from gRPC server... This may take a while.")
	descriptors, err := reflection.FetchAllDescriptors(ctx, grpcConn, config.MaxRetries)
	if err != nil {
		return fmt.Errorf("failed to fetch descriptors: %w", err)
	}

	slog.Info("Building protocol buffer descriptor set...")
	files, err := reflection.BuildFileDescriptorSet(descriptors)
	if err != nil {
		return fmt.Errorf("failed to build descriptor set: %w", err)
	}

	resolver := reflection.NewCustomResolver(files, grpcConn, ctx, config.MaxRetries)

	if config.BlockStart == 0 {
		// Set the start block to the latest local block + 1 if not specified
		// If the latest local block is not available, start from block 1
		config.BlockStart = 1

		latestLocalBlock, err := outputHandler.GetLatestBlock(ctx)
		if err != nil {
			return fmt.Errorf("failed to get the latest block: %w", err)
		}
		if latestLocalBlock != nil {
			config.BlockStart = latestLocalBlock.ID + 1
		}
	}

	if config.BlockStop == 0 {
		// Set the stop block to the latest remote block if not specified
		// If the latest remote block is not available, stop at the latest block
		config.BlockStop = 1

		latestRemoteBlock, err := utils.GetLatestBlockHeightWithRetry(ctx, grpcConn, resolver, config.MaxRetries)
		if err != nil {
			return fmt.Errorf("failed to get the latest block: %w", err)
		}
		config.BlockStop = latestRemoteBlock
	}

	if config.BlockStart > config.BlockStop {
		return fmt.Errorf("start block is greater than stop block")
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
