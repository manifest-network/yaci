package output

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/liftedinit/yaci/internal/models"
)

type TSVOutputHandler struct {
	blockFile   *os.File
	txFile      *os.File
	blockWriter *bufio.Writer
	txWriter    *bufio.Writer
	mu          sync.Mutex
}

const (
	blocksTSV = "blocks.tsv"
	txsTSV    = "transactions.tsv"
)

func NewTSVOutputHandler(outDir string) (*TSVOutputHandler, error) {
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	blockFilePath := filepath.Join(outDir, blocksTSV)
	txFilePath := filepath.Join(outDir, txsTSV)

	blockFile, err := os.Create(blockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create blocks TSV file: %w", err)
	}

	txFile, err := os.Create(txFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactions TSV file: %w", err)
	}

	return &TSVOutputHandler{
		blockFile:   blockFile,
		txFile:      txFile,
		blockWriter: bufio.NewWriter(blockFile),
		txWriter:    bufio.NewWriter(txFile),
	}, nil
}

func (h *TSVOutputHandler) WriteBlockWithTransactions(_ context.Context, block *models.Block, transactions []*models.Transaction) error {
	h.mu.Lock()
	defer h.mu.Unlock()
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

func (h *TSVOutputHandler) writeBlock(block *models.Block) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	line := fmt.Sprintf("%d\t%s\n", block.ID, string(block.Data))
	_, err := h.blockWriter.WriteString(line)
	return err
}

func (h *TSVOutputHandler) writeTransaction(tx *models.Transaction) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	line := fmt.Sprintf("%s\t%s\n", tx.Hash, string(tx.Data))
	_, err := h.txWriter.WriteString(line)
	return err
}

func (h *TSVOutputHandler) Close() error {
	if err := h.blockWriter.Flush(); err != nil {
		slog.Error("failed to flush block writer", "errors", err)
		return err
	}
	if err := h.txWriter.Flush(); err != nil {
		slog.Error("failed to flush tx writer", "errors", err)
		return err
	}
	if err := h.blockFile.Close(); err != nil {
		slog.Error("failed to close block file", "errors", err)
		return err
	}
	if err := h.txFile.Close(); err != nil {
		slog.Error("failed to close tx file", "errors", err)
		return err
	}
	return nil
}
