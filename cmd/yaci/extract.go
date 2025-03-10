package yaci

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ExtractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract chain data to various output formats",
	Long:  `Extract blockchain data and output it in the specified format.`,
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

	if err := viper.BindPFlags(ExtractCmd.PersistentFlags()); err != nil {
		slog.Error("Failed to bind ExtractCmd flags", "error", err)
	}

	ExtractCmd.AddCommand(PostgresCmd)
}
