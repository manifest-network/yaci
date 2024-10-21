package cosmos_dump

import (
	"context"
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
	start     uint64
	stop      uint64
	insecure  bool
	live      bool
	blockTime uint64
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

	// Initialize gRPC client and reflection client
	grpcConn, refClient := client.NewGRPCClients(ctx, address, insecure)
	defer grpcConn.Close()

	// Fetch descriptors and build resolver
	descriptors, err := reflection.FetchAllDescriptors(ctx, refClient)
	if err != nil {
		return errors.WithMessage(err, "failed to fetch descriptors")
	}

	files, err := reflection.BuildFileDescriptorSet(descriptors)
	if err != nil {
		return errors.WithMessage(err, "failed to build descriptor set")
	}

	resolver := reflection.NewCustomResolver(files, refClient, ctx)

	if live {
		// Live mode
		err = extractor.ExtractLiveBlocksAndTransactions(ctx, grpcConn, resolver, start, outputHandler, blockTime)
		if err != nil {
			return errors.WithMessage(err, "failed to process live blocks and transactions")
		}
	} else {
		// Batch mode
		err = extractor.ExtractBlocksAndTransactions(ctx, grpcConn, resolver, start, stop, outputHandler)
		if err != nil {
			return errors.WithMessage(err, "failed to process blocks and transactions")
		}
	}

	return nil
}
