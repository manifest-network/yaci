package yaci

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/liftedinit/yaci/internal/output"
)

var jsonCmd = &cobra.Command{
	Use:   "json [address] [flags]",
	Short: "Extract chain data to JSON files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut := viper.GetString("json-out")
		slog.Debug("Command-line argument", "json-out", jsonOut)

		err := os.MkdirAll(jsonOut, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		outputHandler, err := output.NewJSONOutputHandler(jsonOut)
		if err != nil {
			return fmt.Errorf("failed to create JSON output handler: %w", err)
		}
		defer outputHandler.Close()

		return extract(args[0], outputHandler)
	},
}

func init() {
	jsonCmd.Flags().StringP("json-out", "o", "out", "JSON output directory")
	if err := viper.BindPFlags(jsonCmd.Flags()); err != nil {
		slog.Error("Failed to bind jsonCmd flags", "error", err)
	}
}
