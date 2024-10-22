package cosmos_dump

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/liftedinit/cosmos-dump/internal/client"
	"github.com/liftedinit/cosmos-dump/internal/extractor"
	"github.com/liftedinit/cosmos-dump/internal/output"
	"github.com/liftedinit/cosmos-dump/internal/reflection"
)

var (
	start          uint64
	stop           uint64
	insecure       bool
	live           bool
	blockTime      uint64
	maxConcurrency uint64
	grpcPoolSize   uint64
)

var ExtractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract chain data to various output formats",
	Long:  `Extract blockchain data and output it in the specified format.`,
}

func init() {
	ExtractCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "Skip TLS certificate verification (INSECURE)")
	ExtractCmd.PersistentFlags().BoolVar(&live, "live", false, "Enable live monitoring")
	ExtractCmd.PersistentFlags().Uint64VarP(&start, "start", "s", 1, "Start block height")
	ExtractCmd.PersistentFlags().Uint64VarP(&stop, "stop", "e", 1, "Stop block height")
	ExtractCmd.PersistentFlags().Uint64VarP(&blockTime, "block-time", "t", 2, "Block time in seconds")
	ExtractCmd.PersistentFlags().Uint64VarP(&maxConcurrency, "max-concurrency", "c", 10, "Maximum block retrieval concurrency (advanced)")
	ExtractCmd.PersistentFlags().Uint64VarP(&grpcPoolSize, "grpc-pool-size", "p", 100, "gRPC client pool size (advanced)")

	ExtractCmd.AddCommand(jsonCmd)
	ExtractCmd.AddCommand(tsvCmd)
	ExtractCmd.AddCommand(postgresCmd)
}

func extract(address string, outputHandler output.OutputHandler) error {
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

	slog.Info("Initializing gRPC client pool", "address", address, "insecure", insecure, "grpc-pool-size", grpcPoolSize, "max-concurrency", maxConcurrency)
	grpcPool, err := client.NewGRPCClientPool(ctx, address, insecure, int(grpcPoolSize))
	if err != nil {
		return fmt.Errorf("failed to initialize gRPC client pool: %v", err)
	}
	defer grpcPool.Close()

	slog.Info("Fetching protocol buffer descriptors from gRPC server...")
	descriptors, err := reflection.FetchAllDescriptors(ctx, grpcPool, maxConcurrency)
	if err != nil {
		return errors.WithMessage(err, "failed to fetch descriptors")
	}

	slog.Info("Building protocol buffer descriptor set...")
	files, err := reflection.BuildFileDescriptorSet(descriptors)
	if err != nil {
		return errors.WithMessage(err, "failed to build descriptor set")
	}

	resolver := reflection.NewCustomResolver(files, grpcPool, ctx)

	if live {
		slog.Info("Starting live extraction", "block_time", blockTime)
		err = extractor.ExtractLiveBlocksAndTransactions(ctx, grpcPool, resolver, start, outputHandler, blockTime, maxConcurrency)
		if err != nil {
			return errors.WithMessage(err, "failed to process live blocks and transactions")
		}
	} else {
		slog.Info("Starting extraction", "start", start, "stop", stop)
		err = extractor.ExtractBlocksAndTransactions(ctx, grpcPool, resolver, start, stop, outputHandler, maxConcurrency)
		if err != nil {
			return errors.WithMessage(err, "failed to process blocks and transactions")
		}
	}

	return nil
}
