package utils

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/liftedinit/yaci/internal/client"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func GetBech32Prefix(gRPCClient *client.GRPCClient, maxRetries uint) (string, error) {
	methodFullName := "cosmos.auth.v1beta1.Query.Bech32Prefix"
	serviceName, methodNameOnly, err := ParseMethodFullName(methodFullName)
	if err != nil {
		return "", err
	}

	methodDescriptor, err := gRPCClient.Resolver.FindMethodDescriptor(serviceName, methodNameOnly)
	if err != nil {
		return "", fmt.Errorf("failed to find method descriptor: %v", err)
	}
	fullMethodName := BuildFullMethodName(methodDescriptor)

	var bech32Prefix string
	for attempt := uint(1); attempt <= maxRetries; attempt++ {
		bech32Prefix, err = getBech32Prefix(gRPCClient, fullMethodName, methodDescriptor)
		if err == nil {
			return bech32Prefix, nil
		}
		slog.Debug("Retrying getting bech32 prefix", "attempt", attempt, "error", err)
		time.Sleep(time.Duration(2*attempt) * time.Second)
	}

	return "", errors.WithMessage(err, fmt.Sprintf("Failed to get bech32 prefix after %d retries", maxRetries))
}

func getBech32Prefix(gRPCClient *client.GRPCClient, fullMethodName string, methodDescriptor protoreflect.MethodDescriptor) (string, error) {
	inputMsg := dynamicpb.NewMessage(methodDescriptor.Input())
	outputMsg := dynamicpb.NewMessage(methodDescriptor.Output())

	err := gRPCClient.Conn.Invoke(gRPCClient.Ctx, fullMethodName, inputMsg, outputMsg)
	if err != nil {
		return "", errors.WithMessage(err, "error invoking method")
	}

	// Extract the bech32 prefix from the response
	bech32Prefix := outputMsg.ProtoReflect().Get(outputMsg.Descriptor().Fields().ByName("bech32_prefix"))
	if !bech32Prefix.IsValid() {
		return "", errors.WithMessage(err, "bech32Prefix field not found in status response")
	}

	return bech32Prefix.String(), nil
}
