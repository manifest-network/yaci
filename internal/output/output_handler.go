package output

import (
	"context"

	"github.com/liftedinit/yaci/internal/models"
)

type OutputHandler interface {
	WriteBlockWithTransactions(ctx context.Context, block *models.Block, transactions []*models.Transaction) error
	GetLatestBlock(ctx context.Context) (*models.Block, error)
	Close() error
}
