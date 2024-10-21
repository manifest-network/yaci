package extractor

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/liftedinit/cosmos-dump/internal/output"
	"github.com/liftedinit/cosmos-dump/internal/reflection"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ExtractLiveBlocksAndTransactions monitors the chain and processes new blocks as they are produced.
func ExtractLiveBlocksAndTransactions(ctx context.Context, conn *grpc.ClientConn, resolver *reflection.CustomResolver, start uint64, outputHandler output.OutputHandler) error {
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

	uo := protojson.UnmarshalOptions{
		Resolver: resolver,
	}

	currentHeight := start

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Get the latest block height
			latestHeight, err := getLatestBlockHeight(ctx, conn, statusFullMethodName, statusMethodDescriptor, uo)
			if err != nil {
				return fmt.Errorf("failed to get latest block height: %v", err)
			}

			if latestHeight > currentHeight {
				fmt.Printf("New block detected: %d\n", latestHeight)
				// Process new blocks
				err = ExtractBlocksAndTransactions(ctx, conn, resolver, currentHeight+1, latestHeight, outputHandler)
				if err != nil {
					return fmt.Errorf("failed to process blocks and transactions: %v", err)
				}
				currentHeight = latestHeight
			}

			// Sleep before checking again
			time.Sleep(5 * time.Second)
		}
	}
}

func getLatestBlockHeight(ctx context.Context, conn *grpc.ClientConn, fullMethodName string, methodDescriptor protoreflect.MethodDescriptor, uo protojson.UnmarshalOptions) (uint64, error) {
	// Create the request message (empty)
	inputMsg := dynamicpb.NewMessage(methodDescriptor.Input())

	// Create the response message
	outputMsg := dynamicpb.NewMessage(methodDescriptor.Output())

	err := conn.Invoke(ctx, fullMethodName, inputMsg, outputMsg)
	if err != nil {
		return 0, fmt.Errorf("error invoking status method: %v", err)
	}

	// Extract the latest block height from the response
	latestHeightStr := outputMsg.ProtoReflect().Get(outputMsg.Descriptor().Fields().ByName("height"))
	if !latestHeightStr.IsValid() {
		return 0, fmt.Errorf("height field not found in status response")
	}

	latestHeight, err := strconv.ParseUint(latestHeightStr.String(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse latest block height: %v", err)
	}

	return latestHeight, nil
}
