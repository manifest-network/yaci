package output

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/liftedinit/yaci/internal/models"
)

type JSONOutputHandler struct {
	blockDir string
	txDir    string
}

func NewJSONOutputHandler(outDir string) (*JSONOutputHandler, error) {
	blockDir := filepath.Join(outDir, "block")
	txDir := filepath.Join(outDir, "txs")

	err := os.MkdirAll(blockDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create blocks directory: %w", err)
	}

	err = os.MkdirAll(txDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactions directory: %w", err)
	}

	return &JSONOutputHandler{
		blockDir: blockDir,
		txDir:    txDir,
	}, nil
}

func (h *JSONOutputHandler) WriteBlockWithTransactions(_ context.Context, block *models.Block, transactions []*models.Transaction) error {
	if err := h.writeBlock(block); err != nil {
		return fmt.Errorf("failed to write block: %w", err)
	}

	for _, tx := range transactions {
		if err := h.writeTransaction(tx); err != nil {
			return fmt.Errorf("failed to write transaction: %w", err)
		}
	}

	return nil
}

func (h *JSONOutputHandler) writeBlock(block *models.Block) error {
	fileName := fmt.Sprintf("block_%010d.json", block.ID)
	filePath := filepath.Join(h.blockDir, fileName)
	return os.WriteFile(filePath, block.Data, 0644)
}

func (h *JSONOutputHandler) writeTransaction(tx *models.Transaction) error {
	fileName := fmt.Sprintf("tx_%s.json", tx.Hash)
	filePath := filepath.Join(h.txDir, fileName)
	return os.WriteFile(filePath, tx.Data, 0644)
}

func (h *JSONOutputHandler) Close() error {
	return nil
}
