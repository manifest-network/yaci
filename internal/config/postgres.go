package config

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

type PostgresConfig struct {
	ConnString string
}

func (c PostgresConfig) Validate() error {
	if c.ConnString == "" {
		return fmt.Errorf("missing PostgreSQL connection string")
	}

	_, err := pgxpool.ParseConfig(c.ConnString)
	if err != nil {
		return fmt.Errorf("failed to parse PostgreSQL connection string: %w", err)
	}

	return nil
}

func LoadPostgresConfigFromCLI() PostgresConfig {
	return PostgresConfig{
		ConnString: viper.GetString("postgres-conn"),
	}
}
