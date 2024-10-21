package output

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/liftedinit/cosmos-dump/internal/models"
)

type TSVOutputHandler struct {
	blockFile   *os.File
	txFile      *os.File
	blockWriter *bufio.Writer
	txWriter    *bufio.Writer
}

func NewTSVOutputHandler(outDir string) (*TSVOutputHandler, error) {
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	blockFilePath := filepath.Join(outDir, "blocks.tsv")
	txFilePath := filepath.Join(outDir, "transactions.tsv")

	blockFile, err := os.Create(blockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create blocks TSV file: %v", err)
	}

	txFile, err := os.Create(txFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactions TSV file: %v", err)
	}

	return &TSVOutputHandler{
		blockFile:   blockFile,
		txFile:      txFile,
		blockWriter: bufio.NewWriter(blockFile),
		txWriter:    bufio.NewWriter(txFile),
	}, nil
}

func (h *TSVOutputHandler) WriteBlock(ctx context.Context, block *models.Block) error {
	line := fmt.Sprintf("%d\t%s\n", block.ID, string(block.Data))
	_, err := h.blockWriter.WriteString(line)
	return err
}

func (h *TSVOutputHandler) WriteTransaction(ctx context.Context, tx *models.Transaction) error {
	line := fmt.Sprintf("%s\t%s\n", tx.Hash, string(tx.Data))
	_, err := h.txWriter.WriteString(line)
	return err
}

func (h *TSVOutputHandler) Close() error {
	if err := h.blockWriter.Flush(); err != nil {
		return err
	}
	if err := h.txWriter.Flush(); err != nil {
		return err
	}
	if err := h.blockFile.Close(); err != nil {
		return err
	}
	if err := h.txFile.Close(); err != nil {
		return err
	}
	return nil
}
