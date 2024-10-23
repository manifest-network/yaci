package output

import (
	"context"

	"github.com/liftedinit/yaci/internal/models"
)

type OutputHandler interface {
	WriteBlock(ctx context.Context, block *models.Block) error
	WriteTransaction(ctx context.Context, tx *models.Transaction) error
	Close() error
}
