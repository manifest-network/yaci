package output

import (
	"context"

	"github.com/liftedinit/cosmos-dump/internal/models"
)

// OutputHandler defines methods for writing blocks and transactions.
type OutputHandler interface {
	WriteBlock(ctx context.Context, block *models.Block) error
	WriteTransaction(ctx context.Context, tx *models.Transaction) error
	Close() error
}
