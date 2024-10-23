package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/liftedinit/cosmos-dump/internal/models"
	"github.com/liftedinit/cosmos-dump/internal/output"
	"github.com/liftedinit/cosmos-dump/internal/reflection"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
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
					slog.Error("Failed to process block after 3 retries", "height", blockHeight, "error", err)
					errCh <- errors.WithMessagef(err, "Failed to process block %d", blockHeight)
				}
			}()
		}
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		return errors.WithMessage(err, "Error while fetching blocks")
	}

	return nil
}

func processSingleBlockWithRetry(ctx context.Context, grpcConn *grpc.ClientConn, resolver *reflection.CustomResolver, blockHeight uint64, outputHandler output.OutputHandler, maxRetries uint) error {
	var err error
	for attempt := uint(1); attempt <= maxRetries; attempt++ {
		err = processSingleBlock(ctx, grpcConn, resolver, blockHeight, outputHandler)
		if err == nil {
			return nil
		}
		slog.Warn("Retrying processing block", "height", blockHeight, "attempt", attempt, "error", err)
		time.Sleep(time.Duration(2*attempt) * time.Second)
	}
	return errors.WithMessagef(err, "failed to process block %d after %d attempts", blockHeight, maxRetries)
}

func processSingleBlock(ctx context.Context, grpcConn *grpc.ClientConn, resolver *reflection.CustomResolver, blockHeight uint64, outputHandler output.OutputHandler) error {
	files := resolver.Files()

	blockServiceName, blockMethodNameOnly, err := parseMethodFullName(blockMethodFullName)
	if err != nil {
		return errors.WithMessage(err, "failed to parse block method full name")
	}

	blockMethodDescriptor, err := reflection.FindMethodDescriptor(files, blockServiceName, blockMethodNameOnly)
	if err != nil {
		return errors.WithMessage(err, "failed to find block method descriptor")
	}

	blockFullMethodName := buildFullMethodName(blockMethodDescriptor)

	txServiceName, txMethodNameOnly, err := parseMethodFullName(txMethodFullName)
	if err != nil {
		return errors.WithMessage(err, "failed to parse tx method full name")
	}

	txMethodDescriptor, err := reflection.FindMethodDescriptor(files, txServiceName, txMethodNameOnly)
	if err != nil {
		return errors.WithMessage(err, "failed to find tx method descriptor")
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
		return errors.WithMessage(err, "failed to parse block input parameters")
	}

	// Create the response message
	blockOutputMsg := dynamicpb.NewMessage(blockMethodDescriptor.Output())

	err = grpcConn.Invoke(ctx, blockFullMethodName, blockInputMsg, blockOutputMsg)
	if err != nil {
		return errors.WithMessage(err, "error invoking block method")
	}

	blockJsonBytes, err := mo.Marshal(blockOutputMsg)
	if err != nil {
		return errors.WithMessage(err, "failed to marshal block response")
	}

	block := &models.Block{
		ID:   blockHeight,
		Data: blockJsonBytes,
	}

	err = outputHandler.WriteBlock(ctx, block)
	if err != nil {
		return fmt.Errorf("failed to write block: %v", err)
	}

	// Process transactions
	var data map[string]interface{}
	if err := json.Unmarshal(blockJsonBytes, &data); err != nil {
		return errors.WithMessage(err, "failed to unmarshal block JSON")
	}

	// Get txs from block, if any
	err = extractTransactions(ctx, grpcConn, data, txMethodDescriptor, txFullMethodName, blockHeight, outputHandler, uo, mo)
	if err != nil {
		return err
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
