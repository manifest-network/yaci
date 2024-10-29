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

var jsonCmd = &cobra.Command{
	Use:   "json [address] [flags]",
	Short: "Extract chain data to JSON files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		extractConfig := config.LoadExtractConfigFromCLI()
		if err := extractConfig.Validate(); err != nil {
			return fmt.Errorf("invalid Extract configuration: %w", err)
		}
		jsonConfig := config.LoadJSONConfigFromCLI()
		if err := jsonConfig.Validate(); err != nil {
			return fmt.Errorf("invalid JSON configuration: %w", err)
		}

		err := os.MkdirAll(jsonConfig.Output, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		outputHandler, err := output.NewJSONOutputHandler(jsonConfig.Output)
		if err != nil {
			return fmt.Errorf("failed to create JSON output handler: %w", err)
		}
		defer outputHandler.Close()

		// TODO: Resume from the latest block

		//return extractor.Extract(args[0], outputHandler, extractConfig)
		return nil
	},
}

func init() {
	jsonCmd.Flags().StringP("json-out", "o", "out", "JSON output directory")
	if err := viper.BindPFlags(jsonCmd.Flags()); err != nil {
		slog.Error("Failed to bind jsonCmd flags", "error", err)
	}
}
