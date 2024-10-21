package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/liftedinit/cosmos-dump/internal/models"
	"github.com/liftedinit/cosmos-dump/internal/output"
	"github.com/liftedinit/cosmos-dump/internal/reflection"
)

const (
	blockMethodFullName = "cosmos.tx.v1beta1.Service.GetBlockWithTxs"
	txMethodFullName    = "cosmos.tx.v1beta1.Service.GetTx"
)

func ExtractBlocksAndTransactions(ctx context.Context, conn *grpc.ClientConn, resolver *reflection.CustomResolver, start, stop uint64, outputHandler output.OutputHandler) error {
	files := resolver.Files()

	blockServiceName, blockMethodNameOnly, err := parseMethodFullName(blockMethodFullName)
	if err != nil {
		return err
	}

	blockMethodDescriptor, err := reflection.FindMethodDescriptor(files, blockServiceName, blockMethodNameOnly)
	if err != nil {
		return fmt.Errorf("failed to find block method descriptor: %v", err)
	}

	blockFullMethodName := buildFullMethodName(blockMethodDescriptor)

	txServiceName, txMethodNameOnly, err := parseMethodFullName(txMethodFullName)
	if err != nil {
		return err
	}

	txMethodDescriptor, err := reflection.FindMethodDescriptor(files, txServiceName, txMethodNameOnly)
	if err != nil {
		return fmt.Errorf("failed to find tx method descriptor: %v", err)
	}

	txFullMethodName := buildFullMethodName(txMethodDescriptor)

	uo := protojson.UnmarshalOptions{
		Resolver: resolver,
	}

	mo := protojson.MarshalOptions{
		Resolver: resolver,
	}

	for i := start; i <= stop; i++ {
		blockJsonParams := fmt.Sprintf(`{"height": %d}`, i)

		// Create the request message
		blockInputMsg := dynamicpb.NewMessage(blockMethodDescriptor.Input())

		if err := uo.Unmarshal([]byte(blockJsonParams), blockInputMsg); err != nil {
			return fmt.Errorf("failed to parse block input parameters: %v", err)
		}

		// Create the response message
		blockOutputMsg := dynamicpb.NewMessage(blockMethodDescriptor.Output())

		err = conn.Invoke(ctx, blockFullMethodName, blockInputMsg, blockOutputMsg)
		if err != nil {
			return fmt.Errorf("error invoking block method: %v", err)
		}

		blockJsonBytes, err := mo.Marshal(blockOutputMsg)
		if err != nil {
			return fmt.Errorf("failed to marshal block response: %v", err)
		}

		block := &models.Block{
			ID:   i,
			Data: blockJsonBytes,
		}

		err = outputHandler.WriteBlock(ctx, block)
		if err != nil {
			return fmt.Errorf("failed to write block: %v", err)
		}

		// Process transactions
		var data map[string]interface{}
		if err := json.Unmarshal(blockJsonBytes, &data); err != nil {
			return fmt.Errorf("failed to unmarshal block JSON: %v", err)
		}

		// Get txs from block, if any
		err = extractTransactions(ctx, conn, data, txMethodDescriptor, txFullMethodName, i, outputHandler, uo, mo)
		if err != nil {
			return err
		}

	}

	return nil
}

func parseMethodFullName(methodFullName string) (string, string, error) {
	lastDot := strings.LastIndex(methodFullName, ".")
	if lastDot == -1 {
		return "", "", fmt.Errorf("invalid method name: %s", methodFullName)
	}
	serviceName := methodFullName[:lastDot]
	methodNameOnly := methodFullName[lastDot+1:]
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
