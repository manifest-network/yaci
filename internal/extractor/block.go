package extractor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/liftedinit/yaci/internal/client"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/liftedinit/yaci/internal/utils"
	"github.com/schollz/progressbar/v3"

	"github.com/liftedinit/yaci/internal/models"
	"github.com/liftedinit/yaci/internal/output"
)

const (
	blockMethodFullName = "cosmos.tx.v1beta1.Service.GetBlockWithTxs"
	txMethodFullName    = "cosmos.tx.v1beta1.Service.GetTx"
)

func ExtractBlocksAndTransactions(gRPCClient *client.GRPCClient, start, stop uint64, outputHandler output.OutputHandler, maxConcurrency, maxRetries uint) error {
	displayProgress := start != stop
	if displayProgress {
		slog.Info("Extracting blocks and transactions", "range", fmt.Sprintf("[%d, %d]", start, stop))
	} else {
		slog.Info("Extracting blocks and transactions", "height", start)
	}
	var bar *progressbar.ProgressBar
	if displayProgress {
		bar = progressbar.NewOptions64(
			int64(stop-start+1),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionSetDescription("Processing blocks..."),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "=",
				SaucerHead:    ">",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)
		if err := bar.RenderBlank(); err != nil {
			return fmt.Errorf("failed to render progress bar: %w", err)
		}
	}

	if err := processBlocks(gRPCClient, start, stop, outputHandler, maxConcurrency, maxRetries, bar); err != nil {
		return fmt.Errorf("failed to process blocks and transactions: %w", err)
	}

	if bar != nil {
		if err := bar.Finish(); err != nil {
			return fmt.Errorf("failed to finish progress bar: %w", err)
		}
	}

	return nil
}

func processBlocks(gRPCClient *client.GRPCClient, start, stop uint64, outputHandler output.OutputHandler, maxConcurrency, maxRetries uint, bar *progressbar.ProgressBar) error {
	eg, _ := errgroup.WithContext(gRPCClient.Ctx)
	sem := make(chan struct{}, maxConcurrency)

	for height := start; height <= stop; height++ {
		blockHeight := height
		sem <- struct{}{}

		eg.Go(func() error {
			defer func() { <-sem }()

			err := ProcessSingleBlockWithRetry(gRPCClient, blockHeight, outputHandler, maxRetries)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					slog.Error("Failed to process block", "height", blockHeight, "error", err, "retries", maxRetries)
				}
				return fmt.Errorf("failed to process block %d: %w", blockHeight, err)
			}

			if bar != nil {
				if err := bar.Add(1); err != nil {
					slog.Warn("Failed to update progress bar", "error", err)
				}
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("error while fetching blocks: %w", err)
	}
	return nil
}

func ProcessSingleBlockWithRetry(gRPCClient *client.GRPCClient, blockHeight uint64, outputHandler output.OutputHandler, maxRetries uint) error {
	var err error
	for attempt := uint(1); attempt <= maxRetries; attempt++ {
		// Check if the context has been cancelled before starting
		if gRPCClient.Ctx.Err() != nil {
			return gRPCClient.Ctx.Err()
		}
		err = processSingleBlock(gRPCClient, blockHeight, outputHandler)
		if err == nil {
			return nil
		}
		// Check if the context has been cancelled before retrying
		if gRPCClient.Ctx.Err() != nil {
			return gRPCClient.Ctx.Err()
		}
		// Wait before retrying
		slog.Debug("Retrying processing block", "height", blockHeight, "attempt", attempt, "error", err)
		select {
		// Check if the context has been cancelled during the sleep
		case <-gRPCClient.Ctx.Done():
			return gRPCClient.Ctx.Err()
		case <-time.After(time.Duration(2*attempt) * time.Second):
		}
	}
	return err
}

func processSingleBlock(gRPCClient *client.GRPCClient, blockHeight uint64, outputHandler output.OutputHandler) error {
	blockServiceName, blockMethodNameOnly, err := utils.ParseMethodFullName(blockMethodFullName)
	if err != nil {
		return fmt.Errorf("failed to parse block method full name: %w", err)
	}

	blockMethodDescriptor, err := gRPCClient.Resolver.FindMethodDescriptor(blockServiceName, blockMethodNameOnly)
	if err != nil {
		return fmt.Errorf("failed to find block method descriptor: %w", err)
	}

	blockFullMethodName := utils.BuildFullMethodName(blockMethodDescriptor)

	txServiceName, txMethodNameOnly, err := utils.ParseMethodFullName(txMethodFullName)
	if err != nil {
		return fmt.Errorf("failed to parse tx method full name: %w", err)
	}

	txMethodDescriptor, err := gRPCClient.Resolver.FindMethodDescriptor(txServiceName, txMethodNameOnly)
	if err != nil {
		return fmt.Errorf("failed to find tx method descriptor: %w", err)
	}

	txFullMethodName := utils.BuildFullMethodName(txMethodDescriptor)

	uo := protojson.UnmarshalOptions{
		Resolver: gRPCClient.Resolver,
	}

	mo := protojson.MarshalOptions{
		Resolver: gRPCClient.Resolver,
	}

	blockJsonParams := fmt.Sprintf(`{"height": %d}`, blockHeight)

	// Create the request message
	blockInputMsg := dynamicpb.NewMessage(blockMethodDescriptor.Input())

	if err := uo.Unmarshal([]byte(blockJsonParams), blockInputMsg); err != nil {
		return fmt.Errorf("failed to parse block input parameters: %w", err)
	}

	// Create the response message
	blockOutputMsg := dynamicpb.NewMessage(blockMethodDescriptor.Output())

	err = gRPCClient.Conn.Invoke(gRPCClient.Ctx, blockFullMethodName, blockInputMsg, blockOutputMsg)
	if err != nil {
		return fmt.Errorf("error invoking block method: %w", err)
	}

	blockJsonBytes, err := mo.Marshal(blockOutputMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal block response: %w", err)
	}

	block := &models.Block{
		ID:   blockHeight,
		Data: blockJsonBytes,
	}

	// Process transactions
	var data map[string]interface{}
	if err := json.Unmarshal(blockJsonBytes, &data); err != nil {
		return fmt.Errorf("failed to unmarshal block JSON: %w", err)
	}

	// Get txs from block, if any
	transactions, err := extractTransactions(gRPCClient.Ctx, gRPCClient.Conn, data, txMethodDescriptor, txFullMethodName, uo, mo)
	if err != nil {
		return fmt.Errorf("failed to extract transactions from block: %w", err)
	}

	err = outputHandler.WriteBlockWithTransactions(gRPCClient.Ctx, block, transactions)
	if err != nil {
		return fmt.Errorf("failed to write block with transactions: %w", err)
	}

	return nil
}
