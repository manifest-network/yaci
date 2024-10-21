package extractor

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/liftedinit/cosmos-dump/internal/output"
	"github.com/liftedinit/cosmos-dump/internal/reflection"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ExtractLiveBlocksAndTransactions monitors the chain and processes new blocks as they are produced.
func ExtractLiveBlocksAndTransactions(ctx context.Context, conn *grpc.ClientConn, resolver *reflection.CustomResolver, start uint64, outputHandler output.OutputHandler, blockTime uint64) error {
	// Prepare the Status method descriptors
	statusMethodFullName := "cosmos.base.node.v1beta1.Service.Status"
	statusServiceName, statusMethodNameOnly, err := parseMethodFullName(statusMethodFullName)
	if err != nil {
		return err
	}

	files := resolver.Files()

	statusMethodDescriptor, err := reflection.FindMethodDescriptor(files, statusServiceName, statusMethodNameOnly)
	if err != nil {
		return fmt.Errorf("failed to find status method descriptor: %v", err)
	}

	statusFullMethodName := buildFullMethodName(statusMethodDescriptor)
	currentHeight := start

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Get the latest block height
			latestHeight, err := getLatestBlockHeight(ctx, conn, statusFullMethodName, statusMethodDescriptor)
			if err != nil {
				return errors.WithMessage(err, "Failed to get latest block height")
			}

			if latestHeight > currentHeight {
				err = ExtractBlocksAndTransactions(ctx, conn, resolver, currentHeight+1, latestHeight, outputHandler)
				if err != nil {
					return errors.WithMessage(err, "Failed to process blocks and transactions")
				}
				currentHeight = latestHeight
			}

			// Sleep before checking again
			time.Sleep(time.Duration(blockTime) * time.Second)
		}
	}
}

func getLatestBlockHeight(ctx context.Context, conn *grpc.ClientConn, fullMethodName string, methodDescriptor protoreflect.MethodDescriptor) (uint64, error) {
	// Create the request message (empty)
	inputMsg := dynamicpb.NewMessage(methodDescriptor.Input())

	// Create the response message
	outputMsg := dynamicpb.NewMessage(methodDescriptor.Output())

	err := conn.Invoke(ctx, fullMethodName, inputMsg, outputMsg)
	if err != nil {
		return 0, errors.WithMessage(err, "error invoking status method")
	}

	// Extract the latest block height from the response
	latestHeightStr := outputMsg.ProtoReflect().Get(outputMsg.Descriptor().Fields().ByName("height"))
	if !latestHeightStr.IsValid() {
		return 0, errors.WithMessage(err, "height field not found in status response")
	}

	latestHeight, err := strconv.ParseUint(latestHeightStr.String(), 10, 64)
	if err != nil {
		return 0, errors.WithMessage(err, "failed to parse latest block height")
	}

	return latestHeight, nil
}
