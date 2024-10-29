package yaci

import (
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/liftedinit/yaci/internal/config"
	"github.com/liftedinit/yaci/internal/extractor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/liftedinit/yaci/internal/output"
)

var PostgresRunE = func(cmd *cobra.Command, args []string) error {
	extractConfig := config.LoadExtractConfigFromCLI()
	if err := extractConfig.Validate(); err != nil {
		return fmt.Errorf("invalid Extract configuration: %w", err)
	}

	postgresConfig := config.LoadPostgresConfigFromCLI()
	if err := postgresConfig.Validate(); err != nil {
		return fmt.Errorf("invalid PostgreSQL configuration: %w", err)
	}

	slog.Debug("Command-line arguments", "extractConfig", extractConfig, "postgresConfig", postgresConfig)

	_, err := pgxpool.ParseConfig(postgresConfig.ConnString)
	if err != nil {
		return fmt.Errorf("failed to parse PostgreSQL connection string: %w", err)
	}

	outputHandler, err := output.NewPostgresOutputHandler(postgresConfig.ConnString)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL output handler: %w", err)
	}
	defer outputHandler.Close()

	return extractor.Extract(args[0], outputHandler, extractConfig)
}

var PostgresCmd = &cobra.Command{
	Use:   "postgres [address] [flags]",
	Short: "Extract chain data to a PostgreSQL database",
	Args:  cobra.ExactArgs(1),
	RunE:  PostgresRunE,
}

func init() {
	PostgresCmd.Flags().StringP("postgres-conn", "p", "", "PosftgreSQL connection string")
	if err := viper.BindPFlags(PostgresCmd.Flags()); err != nil {
		slog.Error("Failed to bind postgresCmd flags", "error", err)
	}
}
