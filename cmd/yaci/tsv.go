package yaci

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/liftedinit/yaci/internal/output"
)

var tsvOut string

var tsvCmd = &cobra.Command{
	Use:   "tsv [address] [flags]",
	Short: "Extract chain data to TSV files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := os.MkdirAll(tsvOut, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		outputHandler, err := output.NewTSVOutputHandler(tsvOut)
		if err != nil {
			return fmt.Errorf("failed to create TSV output handler: %w", err)
		}
		defer outputHandler.Close()

		return extract(args[0], outputHandler)
	},
}

func init() {
	tsvCmd.Flags().StringVarP(&tsvOut, "out", "o", "tsv", "Output directory")
}
