package yaci

import (
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/liftedinit/yaci/internal/metrics"
	"github.com/liftedinit/yaci/internal/output/postgresql"
	"github.com/liftedinit/yaci/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/liftedinit/yaci/internal/config"
	"github.com/liftedinit/yaci/internal/extractor"
)

var PostgresRunE = func(cmd *cobra.Command, args []string) error {
	postgresConfig := config.LoadPostgresConfigFromCLI()
	if err := postgresConfig.Validate(); err != nil {
		return fmt.Errorf("invalid PostgreSQL configuration: %w", err)
	}

	slog.Debug("Command-line arguments", "postgresConfig", postgresConfig)

	_, err := pgxpool.ParseConfig(postgresConfig.ConnString)
	if err != nil {
		return fmt.Errorf("failed to parse PostgreSQL connection string: %w", err)
	}

	outputHandler, err := postgresql.NewPostgresOutputHandler(postgresConfig.ConnString)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL output handler: %w", err)
	}
	defer outputHandler.Close()

	if extractConfig.EnablePrometheus {
		slog.Info("Starting Prometheus metrics server...")

		// The total unique addresses metric requires to know the Bech32 prefix of the chain.
		// Query the gRPC server for the Bech32 prefix.
		bech32Prefix, err := utils.GetBech32PrefixWithRetry(gRPCClient, extractConfig.MaxRetries)
		if err != nil {
			return fmt.Errorf("failed to get Bech32 prefix: %w", err)
		}
		slog.Debug("Bech32 prefix retrieved", "bech32_prefix", bech32Prefix)

		db := stdlib.OpenDBFromPool(outputHandler.GetPool())
		if err := metrics.CreateMetricsServer(db, bech32Prefix, extractConfig.PrometheusListenAddr); err != nil {
			return fmt.Errorf("failed to start metrics server: %w", err)
		}
	}

	return extractor.Extract(gRPCClient, outputHandler, extractConfig)
}

var PostgresCmd = &cobra.Command{
	Use:   "postgres [flags]",
	Short: "Extract chain data to a PostgreSQL database",
	RunE:  PostgresRunE,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if parent := cmd.Parent(); parent != nil && parent.PreRunE != nil {
			if err := parent.PreRunE(parent, args); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	PostgresCmd.Flags().StringP("postgres-conn", "p", "", "PosftgreSQL connection string")
	if err := viper.BindPFlags(PostgresCmd.Flags()); err != nil {
		slog.Error("Failed to bind postgresCmd flags", "error", err)
	}
}
