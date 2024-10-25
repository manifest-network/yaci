package yaci

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/liftedinit/yaci/internal/output"
)

var postgresCmd = &cobra.Command{
	Use:   "postgres [address] [psql-connection-string]",
	Short: "Extract chain data to a PostgreSQL database",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		connString := args[1]
		if connString == "" {
			return fmt.Errorf("connection string is required for PostgreSQL output")
		}

		outputHandler, err := output.NewPostgresOutputHandler(connString)
		if err != nil {
			return fmt.Errorf("failed to create PostgreSQL output handler: %w", err)
		}
		defer outputHandler.Close()

		latestBlock, err := outputHandler.GetLatestBlock(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to get the latest block: %w", err)
		}
		if latestBlock != nil {
			slog.Info("Resuming from block", "height", latestBlock.ID)
			start = latestBlock.ID + 1
		}

		return extract(args[0], outputHandler)
	},
}
