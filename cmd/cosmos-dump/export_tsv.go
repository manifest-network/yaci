package cosmos_dump

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/liftedinit/cosmos-dump/internal/exporter"
)

var exportTSVCmd = &cobra.Command{
	Use:   "export-tsv [input] [output]",
	Short: "Export extracted block and transaction data to TSV files",
	Long:  "Reads extracted block and transaction data from the specified input directory and writes it to TSV files in the specified output directory.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputDir := args[0]
		outputDir := args[1]

		// Check if input directory exists
		if _, err := os.Stat(inputDir); os.IsNotExist(err) {
			return fmt.Errorf("input directory '%s' does not exist", inputDir)
		}

		// Ensure output directory exists
		err := os.MkdirAll(outputDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory '%s': %v", outputDir, err)
		}

		// Process blocks
		blocksTSVPath := filepath.Join(outputDir, "blocks.tsv")
		err = exporter.ExportBlocksTSV(inputDir, blocksTSVPath)
		if err != nil {
			return fmt.Errorf("failed to process blocks: %v", err)
		}

		// Process transactions
		txsTSVPath := filepath.Join(outputDir, "transactions.tsv")
		err = exporter.ExportTransactionsTSV(inputDir, txsTSVPath)
		if err != nil {
			return fmt.Errorf("failed to process transactions: %v", err)
		}

		fmt.Println("Export completed successfully.")
		return nil
	},
}
