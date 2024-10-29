package yaci

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/liftedinit/yaci/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/liftedinit/yaci/internal/client"
	"github.com/liftedinit/yaci/internal/extractor"
	"github.com/liftedinit/yaci/internal/output"
	"github.com/liftedinit/yaci/internal/reflection"
)

var ExtractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract chain data to various output formats",
	Long:  `Extract blockchain data and output it in the specified format.`,
}

func init() {
	ExtractCmd.PersistentFlags().BoolP("insecure", "k", false, "Skip TLS certificate verification (INSECURE)")
	ExtractCmd.PersistentFlags().Bool("live", false, "Enable live monitoring")
	ExtractCmd.PersistentFlags().Uint64P("start", "s", 1, "Start block height")
	ExtractCmd.PersistentFlags().Uint64P("stop", "e", 0, "Stop block height")
	ExtractCmd.PersistentFlags().UintP("block-time", "t", 2, "Block time in seconds")
	ExtractCmd.PersistentFlags().UintP("max-retries", "r", 3, "Maximum number of retries for failed block processing")
	ExtractCmd.PersistentFlags().UintP("max-concurrency", "c", 100, "Maximum block retrieval concurrency (advanced)")

	if err := viper.BindPFlags(ExtractCmd.PersistentFlags()); err != nil {
		slog.Error("Failed to bind ExtractCmd flags", "error", err)
	}

	// TODO: Clashes with the Docker test. Why?
	ExtractCmd.MarkFlagsMutuallyExclusive("live", "stop")

	ExtractCmd.AddCommand(jsonCmd)
	ExtractCmd.AddCommand(tsvCmd)
	ExtractCmd.AddCommand(PostgresCmd)
}

func extract(address string, outputHandler output.OutputHandler) error {
	insecure := viper.GetBool("insecure")
	live := viper.GetBool("live")
	start := viper.GetUint64("start")
	stop := viper.GetUint64("stop")
	blockTime := viper.GetUint("block-time")
	maxConcurrency := viper.GetUint("max-concurrency")
	maxRetries := viper.GetUint("max-retries")

	slog.Debug("Command-line arguments", "address", address, "insecure", insecure, "live", live, "start", start, "stop", stop, "block_time", blockTime, "max_concurrency", maxConcurrency, "max_retries", maxRetries)

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
	grpcConn := client.NewGRPCClients(ctx, address, insecure)
	defer grpcConn.Close()

	slog.Info("Fetching protocol buffer descriptors from gRPC server... This may take a while.")
	descriptors, err := reflection.FetchAllDescriptors(ctx, grpcConn, maxRetries)
	if err != nil {
		return fmt.Errorf("failed to fetch descriptors: %w", err)
	}

	slog.Info("Building protocol buffer descriptor set...")
	files, err := reflection.BuildFileDescriptorSet(descriptors)
	if err != nil {
		return fmt.Errorf("failed to build descriptor set: %w", err)
	}

	resolver := reflection.NewCustomResolver(files, grpcConn, ctx, maxRetries)

	if stop == 0 {
		stop, err = utils.GetLatestBlockHeightWithRetry(ctx, grpcConn, resolver, maxRetries)
		if err != nil {
			return fmt.Errorf("failed to get latest block height: %w", err)
		}
	}

	if live {
		slog.Info("Starting live extraction", "block_time", blockTime)
		err = extractor.ExtractLiveBlocksAndTransactions(ctx, grpcConn, resolver, start, outputHandler, blockTime, maxConcurrency, maxRetries)
		if err != nil {
			return fmt.Errorf("failed to process live blocks and transactions: %w", err)
		}
	} else {
		slog.Info("Starting extraction", "start", start, "stop", stop)
		err = extractor.ExtractBlocksAndTransactions(ctx, grpcConn, resolver, start, stop, outputHandler, maxConcurrency, maxRetries)
		if err != nil {
			return fmt.Errorf("failed to process blocks and transactions: %w", err)
		}
	}

	return nil
}
