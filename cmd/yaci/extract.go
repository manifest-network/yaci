package yaci

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/liftedinit/yaci/internal/client"
	"github.com/liftedinit/yaci/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	extractConfig config.ExtractConfig
	gRPCClient    *client.GRPCClient
)

var ExtractCmd = &cobra.Command{
	Use:   "extract [address]",
	Args:  cobra.ExactArgs(1),
	Short: "Extract chain data to various output formats",
	Long:  `Extract blockchain data and output it in the specified format.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if parent := cmd.Parent(); parent != nil && parent.PreRunE != nil {
			if err := parent.PreRunE(parent, args); err != nil {
				return err
			}
		}

		extractConfig = config.LoadExtractConfigFromCLI()
		if err := extractConfig.Validate(); err != nil {
			return fmt.Errorf("invalid Extract configuration: %w", err)
		}

		slog.Debug("Command-line arguments", "extractConfig", extractConfig)
		slog.Debug("gRPC endpoint", "address", args[0])

		ctx, cancel := context.WithCancel(context.Background())
		handleInterrupt(cancel)

		var err error
		gRPCClient, err = client.NewGRPCClient(ctx, args[0], extractConfig.Insecure, extractConfig.MaxRecvMsgSize)
		if err != nil {
			return fmt.Errorf("failed to initialize gRPC: %w", err)
		}

		return nil
	},
}

func init() {
	ExtractCmd.PersistentFlags().BoolP("insecure", "k", false, "Skip TLS certificate verification (INSECURE)")
	ExtractCmd.PersistentFlags().Bool("live", false, "Enable live monitoring")
	ExtractCmd.PersistentFlags().Bool("reindex", false, "Reindex the database from block 1 to the latest block (advanced)")
	ExtractCmd.PersistentFlags().Uint64P("start", "s", 0, "Start block height")
	ExtractCmd.PersistentFlags().Uint64P("stop", "e", 0, "Stop block height")
	ExtractCmd.PersistentFlags().UintP("block-time", "t", 2, "Block time in seconds")
	ExtractCmd.PersistentFlags().UintP("max-retries", "r", 3, "Maximum number of retries for failed block processing")
	ExtractCmd.PersistentFlags().UintP("max-concurrency", "c", 100, "Maximum block retrieval concurrency (advanced)")
	ExtractCmd.PersistentFlags().IntP("max-recv-msg-size", "m", 4194304, "Maximum gRPC message size in bytes (advanced)")
	ExtractCmd.PersistentFlags().Bool("enable-prometheus", false, "Enable Prometheus metrics server")
	ExtractCmd.PersistentFlags().String("prometheus-addr", "0.0.0.0:2112", "Address and port of the Prometheus metrics server")

	if err := viper.BindPFlags(ExtractCmd.PersistentFlags()); err != nil {
		slog.Error("Failed to bind ExtractCmd flags", "error", err)
	}

	ExtractCmd.AddCommand(PostgresCmd)
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
