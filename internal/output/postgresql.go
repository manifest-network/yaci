package output

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/liftedinit/yaci/internal/models"
)

//go:embed sql/init.sql
var initSQL string

type PostgresOutputHandler struct {
	pool *pgxpool.Pool
}

func NewPostgresOutputHandler(connString string) (*PostgresOutputHandler, error) {
	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	handler := &PostgresOutputHandler{
		pool: pool,
	}

	// Initialize tables. This is idempotent.
	if err := handler.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return handler, nil
}

func (h *PostgresOutputHandler) GetLatestBlock(ctx context.Context) (*models.Block, error) {
	var block models.Block
	err := h.pool.QueryRow(ctx, `
		SELECT id
		FROM api.blocks
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&block.ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No rows found
		}
		return nil, fmt.Errorf("failed to get the latest block: %w", err)
	}
	return &block, nil
}

func (h *PostgresOutputHandler) WriteBlockWithTransactions(ctx context.Context, block *models.Block, transactions []*models.Transaction) error {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Ensure rollback if commit is not reached

	// Write block
	_, err = tx.Exec(ctx, `
		INSERT INTO api.blocks (id, data) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data;
	`, block.ID, block.Data)
	if err != nil {
		return fmt.Errorf("failed to write blockchain block: %w", err)
	}

	// Write transactions
	for _, txData := range transactions {
		_, err = tx.Exec(ctx, `
			INSERT INTO api.transactions (id, data) VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data;
		`, txData.Hash, txData.Data)
		if err != nil {
			return fmt.Errorf("failed to write blockchain transaction: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (h *PostgresOutputHandler) initTables() error {
	// Create tables if they don't exist
	slog.Info("Initializing PostgreSQL tables")
	ctx := context.Background()
	_, err := h.pool.Exec(ctx, initSQL)
	return err
}

func (h *PostgresOutputHandler) Close() error {
	slog.Info("Closing PostgreSQL connection pool")
	h.pool.Close()
	return nil
}
