package output

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/liftedinit/cosmos-dump/internal/models"
	"github.com/pkg/errors"
)

type TSVOutputHandler struct {
	blockFile   *os.File
	txFile      *os.File
	blockWriter *bufio.Writer
	txWriter    *bufio.Writer
}

const (
	blocksTSV = "blocks.tsv"
	txsTSV    = "transactions.tsv"
)

func NewTSVOutputHandler(outDir string) (*TSVOutputHandler, error) {
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create output directory")
	}

	blockFilePath := filepath.Join(outDir, blocksTSV)
	txFilePath := filepath.Join(outDir, txsTSV)

	blockFile, err := os.Create(blockFilePath)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create blocks TSV file")
	}

	txFile, err := os.Create(txFilePath)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create transactions TSV file: %v")
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
