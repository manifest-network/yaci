package yaci

import (
	"log/slog"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/liftedinit/yaci/internal/output"
)

var PostgresRunE = func(cmd *cobra.Command, args []string) error {
	postgresConn := viper.GetString("postgres-conn")
	slog.Debug("Command-line argument", "postgres-conn", postgresConn)

	_, err := pgxpool.ParseConfig(postgresConn)
	if err != nil {
		return errors.WithMessage(err, "failed to parse PostgreSQL connection string")
	}

	outputHandler, err := output.NewPostgresOutputHandler(postgresConn)
	if err != nil {
		return errors.WithMessage(err, "failed to create PostgreSQL output handler")
	}
	defer outputHandler.Close()

	return extract(args[0], outputHandler)
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
