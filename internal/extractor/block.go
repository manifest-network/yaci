package extractor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/liftedinit/yaci/internal/models"
	"github.com/liftedinit/yaci/internal/output"
	"github.com/liftedinit/yaci/internal/reflection"
)

const (
	blockMethodFullName = "cosmos.tx.v1beta1.Service.GetBlockWithTxs"
	txMethodFullName    = "cosmos.tx.v1beta1.Service.GetTx"
)

func ExtractBlocksAndTransactions(ctx context.Context, grpcConn *grpc.ClientConn, resolver *reflection.CustomResolver, start, stop uint64, outputHandler output.OutputHandler, maxConcurrency, maxRetries uint) error {
	if start == stop {
		slog.Info("Extracting blocks and transactions", "height", start)

	} else {
		slog.Info("Extracting blocks and transactions", "range", fmt.Sprintf("[%d, %d]", start, stop))
	}

	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	errCh := make(chan error, stop-start+1)

	for i := start; i <= stop; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			sem <- struct{}{}
			wg.Add(1)

			blockHeight := i
			if blockHeight%5000 == 0 {
				slog.Info("Still processing blocks...", "height", blockHeight)
			}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()

				err := processSingleBlockWithRetry(ctx, grpcConn, resolver, blockHeight, outputHandler, maxRetries)
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						slog.Error("Failed to process block after 3 retries", "height", blockHeight, "error", err)
					}
					errCh <- fmt.Errorf("failed to process block %d: %w", blockHeight, err)
				}
			}()
		}
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		return fmt.Errorf("error while fetching blocks: %w", err)
	}

	return nil
}

func processSingleBlockWithRetry(ctx context.Context, grpcConn *grpc.ClientConn, resolver *reflection.CustomResolver, blockHeight uint64, outputHandler output.OutputHandler, maxRetries uint) error {
	var err error
	for attempt := uint(1); attempt <= maxRetries; attempt++ {
		// Check if the context has been cancelled before starting
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err = processSingleBlock(ctx, grpcConn, resolver, blockHeight, outputHandler)
		if err == nil {
			return nil
		}
		// Check if the context has been cancelled before retrying
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Wait before retrying
		slog.Warn("Retrying processing block", "height", blockHeight, "attempt", attempt, "error", err)
		select {
		// Check if the context has been cancelled during the sleep
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(2*attempt) * time.Second):
		}
	}
	return err
}

func processSingleBlock(ctx context.Context, grpcConn *grpc.ClientConn, resolver *reflection.CustomResolver, blockHeight uint64, outputHandler output.OutputHandler) error {
	files := resolver.Files()

	blockServiceName, blockMethodNameOnly, err := parseMethodFullName(blockMethodFullName)
	if err != nil {
		return fmt.Errorf("failed to parse block method full name: %w", err)
	}

	blockMethodDescriptor, err := reflection.FindMethodDescriptor(files, blockServiceName, blockMethodNameOnly)
	if err != nil {
		return fmt.Errorf("failed to find block method descriptor: %w", err)
	}

	blockFullMethodName := buildFullMethodName(blockMethodDescriptor)

	txServiceName, txMethodNameOnly, err := parseMethodFullName(txMethodFullName)
	if err != nil {
		return fmt.Errorf("failed to parse tx method full name: %w", err)
	}

	txMethodDescriptor, err := reflection.FindMethodDescriptor(files, txServiceName, txMethodNameOnly)
	if err != nil {
		return fmt.Errorf("failed to find tx method descriptor: %w", err)
	}

	txFullMethodName := buildFullMethodName(txMethodDescriptor)

	uo := protojson.UnmarshalOptions{
		Resolver: resolver,
	}

	mo := protojson.MarshalOptions{
		Resolver: resolver,
	}

	blockJsonParams := fmt.Sprintf(`{"height": %d}`, blockHeight)

	// Create the request message
	blockInputMsg := dynamicpb.NewMessage(blockMethodDescriptor.Input())

	if err := uo.Unmarshal([]byte(blockJsonParams), blockInputMsg); err != nil {
		return fmt.Errorf("failed to parse block input parameters: %w", err)
	}

	// Create the response message
	blockOutputMsg := dynamicpb.NewMessage(blockMethodDescriptor.Output())

	err = grpcConn.Invoke(ctx, blockFullMethodName, blockInputMsg, blockOutputMsg)
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
	transactions, err := extractTransactions(ctx, grpcConn, data, txMethodDescriptor, txFullMethodName, blockHeight, outputHandler, uo, mo)
	if err != nil {
		return fmt.Errorf("failed to extract transactions from block: %w", err)
	}

	err = outputHandler.WriteBlockWithTransactions(ctx, block, transactions)
	if err != nil {
		return fmt.Errorf("failed to write block with transactions: %w", err)
	}

	return nil
}

func parseMethodFullName(methodFullName string) (string, string, error) {
	if methodFullName == "" {
		return "", "", fmt.Errorf("method full name is empty")
	}

	lastDot := strings.LastIndex(methodFullName, ".")
	if lastDot == -1 {
		return "", "", fmt.Errorf("no dot found in method full name")
	}
	serviceName := methodFullName[:lastDot]
	methodNameOnly := methodFullName[lastDot+1:]

	if serviceName == "" || methodNameOnly == "" {
		return "", "", fmt.Errorf("invalid method full name format")
	}

	return serviceName, methodNameOnly, nil
}

func buildFullMethodName(methodDescriptor protoreflect.MethodDescriptor) string {
	fullMethodName := "/" + string(methodDescriptor.FullName())
	lastDot := strings.LastIndex(fullMethodName, ".")
	if lastDot != -1 {
		fullMethodName = fullMethodName[:lastDot] + "/" + fullMethodName[lastDot+1:]
	}
	return fullMethodName
}
