package utils

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/liftedinit/yaci/internal/client"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func getLatestBlockHeight(gRPCClient *client.GRPCClient, fullMethodName string, methodDescriptor protoreflect.MethodDescriptor) (uint64, error) {
	// Create the request message (empty)
	inputMsg := dynamicpb.NewMessage(methodDescriptor.Input())

	// Create the response message
	outputMsg := dynamicpb.NewMessage(methodDescriptor.Output())

	err := gRPCClient.Conn.Invoke(gRPCClient.Ctx, fullMethodName, inputMsg, outputMsg)
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

func GetLatestBlockHeightWithRetry(gRPCClient *client.GRPCClient, maxRetries uint) (uint64, error) {
	statusMethodFullName := "cosmos.base.node.v1beta1.Service.Status"
	statusServiceName, statusMethodNameOnly, err := ParseMethodFullName(statusMethodFullName)
	if err != nil {
		return 0, err
	}

	statusMethodDescriptor, err := gRPCClient.Resolver.FindMethodDescriptor(statusServiceName, statusMethodNameOnly)
	if err != nil {
		return 0, fmt.Errorf("failed to find status method descriptor: %v", err)
	}
	fullMethodName := BuildFullMethodName(statusMethodDescriptor)

	var latestHeight uint64
	for attempt := uint(1); attempt <= maxRetries; attempt++ {
		latestHeight, err = getLatestBlockHeight(gRPCClient, fullMethodName, statusMethodDescriptor)
		if err == nil {
			return latestHeight, nil
		}
		slog.Debug("Retrying getting latest block height", "attempt", attempt, "error", err)
		time.Sleep(time.Duration(2*attempt) * time.Second)
	}

	return 0, errors.WithMessage(err, fmt.Sprintf("Failed to get latest block height after %d retries", maxRetries))
}
