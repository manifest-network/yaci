package postgresql

import (
	"context"
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"math"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/liftedinit/yaci/internal/models"
)

//go:embed migrations/*
var migrationsFS embed.FS

type PostgresOutputHandler struct {
	pool *pgxpool.Pool
}

func (h *PostgresOutputHandler) GetPool() *pgxpool.Pool {
	return h.pool
}

func NewPostgresOutputHandler(connString string, maxConcurrency uint) (*PostgresOutputHandler, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL connection string: %w", err)
	}

	if maxConcurrency > math.MaxInt32 {
		return nil, fmt.Errorf("max concurrency exceeds maximum int32 value")
	}
	config.MaxConns = int32(maxConcurrency)

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	handler := &PostgresOutputHandler{
		pool: pool,
	}

	// Run migrations. This is idempotent.
	if err = handler.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return handler, nil
}

func (h *PostgresOutputHandler) GetLatestBlock(ctx context.Context) (*models.Block, error) {
	var block models.Block
	err := h.pool.QueryRow(ctx, `
		SELECT id
		FROM api.blocks_raw
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&block.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No rows found
		}
		return nil, fmt.Errorf("failed to get the latest block: %w", err)
	}
	return &block, nil
}

func (h *PostgresOutputHandler) GetEarliestBlock(ctx context.Context) (*models.Block, error) {
	var block models.Block
	err := h.pool.QueryRow(ctx, `
		SELECT id
		FROM api.blocks_raw
		ORDER BY id ASC
		LIMIT 1
	`).Scan(&block.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No rows found
		}
		return nil, fmt.Errorf("failed to get the earliest block: %w", err)
	}
	return &block, nil
}

func (h *PostgresOutputHandler) GetMissingBlockIds(ctx context.Context) ([]uint64, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT s.id
		FROM generate_series(
				 (SELECT MIN(id) FROM api.blocks_raw),
				 (SELECT MAX(id) FROM api.blocks_raw)
			 ) AS s(id)
		LEFT JOIN api.blocks_raw t ON t.id = s.id
		WHERE t.id IS NULL;
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get missing block IDs: %w", err)
	}
	defer rows.Close()

	var missing []uint64
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan missing block ID: %w", err)
		}
		missing = append(missing, id)
	}

	return missing, nil
}

func (h *PostgresOutputHandler) WriteBlockWithTransactions(ctx context.Context, block *models.Block, transactions []*models.Transaction) error {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Ensure rollback if commit is not reached

	// Write block
	_, err = tx.Exec(ctx, `
		INSERT INTO api.blocks_raw (id, data) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data;
	`, block.ID, block.Data)
	if err != nil {
		return fmt.Errorf("failed to write blockchain block: %w", err)
	}

	// Write transactions
	for _, txData := range transactions {
		_, err = tx.Exec(ctx, `
			INSERT INTO api.transactions_raw (id, data) VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data;
		`, txData.Hash, txData.Data)
		if err != nil {
			return fmt.Errorf("failed to write blockchain transaction: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (h *PostgresOutputHandler) runMigrations() error {
	// Create tables if they don't exist
	slog.Info("Running PostgreSQL migrations...")

	d, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	driver, err := migratepgx.WithInstance(stdlib.OpenDBFromPool(h.pool), &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", d, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (h *PostgresOutputHandler) Close() error {
	slog.Info("Closing PostgreSQL connection pool")
	h.pool.Close()
	slog.Info("PostgreSQL connection pool closed")
	return nil
}
