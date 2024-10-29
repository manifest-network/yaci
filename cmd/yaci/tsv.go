package yaci

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/liftedinit/yaci/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/liftedinit/yaci/internal/output"
)

var tsvCmd = &cobra.Command{
	Use:   "tsv [address] [flags]",
	Short: "Extract chain data to TSV files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		extractConfig := config.LoadExtractConfigFromCLI()
		if err := extractConfig.Validate(); err != nil {
			return fmt.Errorf("invalid Extract configuration: %w", err)
		}

		tsvConfig := config.LoadTSVConfigFromCLI()
		if err := tsvConfig.Validate(); err != nil {
			return fmt.Errorf("invalid TSV configuration: %w", err)
		}

		err := os.MkdirAll(tsvConfig.Output, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		outputHandler, err := output.NewTSVOutputHandler(tsvConfig.Output)
		if err != nil {
			return fmt.Errorf("failed to create TSV output handler: %w", err)
		}
		defer outputHandler.Close()

		// TODO: Resume from the latest block

		//return extractor.Extract(args[0], outputHandler, extractConfig)
		return nil
	},
}

func init() {
	tsvCmd.Flags().StringP("tsv-out", "o", "tsv", "Output directory")
	if err := viper.BindPFlags(tsvCmd.Flags()); err != nil {
		slog.Error("Failed to bind tsvCmd flags", "error", err)
	}
}
