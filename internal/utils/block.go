package utils

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/liftedinit/yaci/internal/reflection"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

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

func GetLatestBlockHeightWithRetry(ctx context.Context, conn *grpc.ClientConn, resolver *reflection.CustomResolver, maxRetries uint) (uint64, error) {
	statusMethodFullName := "cosmos.base.node.v1beta1.Service.Status"
	statusServiceName, statusMethodNameOnly, err := ParseMethodFullName(statusMethodFullName)
	if err != nil {
		return 0, err
	}

	statusMethodDescriptor, err := resolver.FindMethodDescriptor(statusServiceName, statusMethodNameOnly)
	if err != nil {
		return 0, fmt.Errorf("failed to find status method descriptor: %v", err)
	}
	fullMethodName := BuildFullMethodName(statusMethodDescriptor)

	var latestHeight uint64
	for attempt := uint(1); attempt <= maxRetries; attempt++ {
		latestHeight, err = getLatestBlockHeight(ctx, conn, fullMethodName, statusMethodDescriptor)
		if err == nil {
			return latestHeight, nil
		}
		slog.Warn("Retrying getting latest block height", "attempt", attempt, "error", err)
		time.Sleep(time.Duration(2*attempt) * time.Second)
	}

	return 0, errors.WithMessage(err, fmt.Sprintf("Failed to get latest block height after %d retries", maxRetries))
}
