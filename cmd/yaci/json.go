package yaci

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/liftedinit/yaci/internal/output"
)

var jsonOut string

var jsonCmd = &cobra.Command{
	Use:   "json [address] [flags]",
	Short: "Extract chain data to JSON files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
	jsonCmd.Flags().StringVarP(&jsonOut, "out", "o", "out", "Output directory")
}
