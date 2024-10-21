package output

import (
	"context"
	_ "embed"
	"log/slog"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"

	"github.com/liftedinit/cosmos-dump/internal/models"
)

//go:embed sql/init.sql
var initSQL string

type PostgresOutputHandler struct {
	pool *pgxpool.Pool
}

func NewPostgresOutputHandler(connString string) (*PostgresOutputHandler, error) {
	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to connect to PostgreSQL")
	}

	handler := &PostgresOutputHandler{
		pool: pool,
	}

	// Initialize tables
	if err := handler.initTables(); err != nil {
		return nil, errors.WithMessage(err, "failed to initialize tables")
	}

	return handler, nil
}

func (h *PostgresOutputHandler) initTables() error {
	// Create tables if they don't exist
	slog.Info("Initializing PostgreSQL tables")
	ctx := context.Background()
	_, err := h.pool.Exec(ctx, initSQL)
	return err
}

func (h *PostgresOutputHandler) WriteBlock(ctx context.Context, block *models.Block) error {
	slog.Debug("Writing block", "id", block.ID)
	_, err := h.pool.Exec(ctx, `
        INSERT INTO api.blocks (id, data) VALUES ($1, $2)
        ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data;
    `, block.ID, block.Data)
	return err
}

func (h *PostgresOutputHandler) WriteTransaction(ctx context.Context, tx *models.Transaction) error {
	slog.Debug("Writing transaction", "hash", tx.Hash)
	_, err := h.pool.Exec(ctx, `
        INSERT INTO api.transactions (id, data) VALUES ($1, $2)
        ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data;
    `, tx.Hash, tx.Data)
	return err
}

func (h *PostgresOutputHandler) Close() error {
	slog.Info("Closing PostgreSQL connection pool")
	h.pool.Close()
	return nil
}
