package output

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/liftedinit/cosmos-dump/internal/models"
)

type PostgresOutputHandler struct {
	pool *pgxpool.Pool
}

func NewPostgresOutputHandler(connString string) (*PostgresOutputHandler, error) {
	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	handler := &PostgresOutputHandler{
		pool: pool,
	}

	// Initialize tables
	if err := handler.initTables(); err != nil {
		return nil, err
	}

	return handler, nil
}

func (h *PostgresOutputHandler) initTables() error {
	// Create tables if they don't exist
	ctx := context.Background()
	_, err := h.pool.Exec(ctx, `
		-- Create the schema if it doesn't exist
		CREATE SCHEMA IF NOT EXISTS api;

		-- Create the tables if they don't exist
        CREATE TABLE IF NOT EXISTS api.blocks (
            id SERIAL PRIMARY KEY,
            data JSONB NOT NULL
        );
        CREATE TABLE IF NOT EXISTS api.transactions (
            id VARCHAR(64) PRIMARY KEY,
            data JSONB NOT NULL
        );

		-- Create a role for anonymous web access if it doesn't exist
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'web_anon') THEN
				CREATE ROLE web_anon NOLOGIN;
			END IF;
		END
		$$;

		-- Grant access to the web_anon role. Will succeed even if the role already has access.
		GRANT USAGE ON SCHEMA api TO web_anon;
		GRANT SELECT ON api.blocks TO web_anon;
		GRANT SELECT ON api.transactions TO web_anon;
    `)
	return err
}

func (h *PostgresOutputHandler) WriteBlock(ctx context.Context, block *models.Block) error {
	_, err := h.pool.Exec(ctx, `
        INSERT INTO api.blocks (id, data) VALUES ($1, $2)
        ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data;
    `, block.ID, block.Data)
	return err
}

func (h *PostgresOutputHandler) WriteTransaction(ctx context.Context, tx *models.Transaction) error {
	_, err := h.pool.Exec(ctx, `
        INSERT INTO api.transactions (id, data) VALUES ($1, $2)
        ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data;
    `, tx.Hash, tx.Data)
	return err
}

func (h *PostgresOutputHandler) Close() error {
	h.pool.Close()
	return nil
}
