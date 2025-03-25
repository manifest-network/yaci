package utils

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/liftedinit/yaci/internal/client"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func ParseMethodFullName(methodFullName string) (string, string, error) {
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

func BuildFullMethodName(methodDescriptor protoreflect.MethodDescriptor) string {
	fullMethodName := "/" + string(methodDescriptor.FullName())
	lastDot := strings.LastIndex(fullMethodName, ".")
	if lastDot != -1 {
		fullMethodName = fullMethodName[:lastDot] + "/" + fullMethodName[lastDot+1:]
	}
	return fullMethodName
}

// RetryGRPCCall retries a gRPC call with exponential backoff
func RetryGRPCCall[T any](
	gRPCClient *client.GRPCClient,
	methodFullName string,
	maxRetries uint,
	callFunc func(string, protoreflect.MethodDescriptor) (T, error),
) (T, error) {
	serviceName, methodNameOnly, err := ParseMethodFullName(methodFullName)
	if err != nil {
		var zero T
		return zero, err
	}

	methodDescriptor, err := gRPCClient.Resolver.FindMethodDescriptor(serviceName, methodNameOnly)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("failed to find method descriptor: %v", err)
	}
	fullMethodName := BuildFullMethodName(methodDescriptor)

	var result T
	for attempt := uint(1); attempt <= maxRetries; attempt++ {
		result, err = callFunc(fullMethodName, methodDescriptor)
		if err == nil {
			return result, nil
		}
		slog.Debug("Retrying gRPC call", "method", methodFullName, "attempt", attempt, "error", err)
		time.Sleep(time.Duration(2*attempt) * time.Second)
	}

	var zero T
	return zero, errors.WithMessage(err, fmt.Sprintf("Failed after %d retries", maxRetries))
}

// ExtractGRPCField calls a gRPC method and extracts a specific field from the response
func ExtractGRPCField[T any](
	gRPCClient *client.GRPCClient,
	methodFullName string,
	maxRetries uint,
	fieldName string,
	converter func(string) (T, error),
) (T, error) {
	return RetryGRPCCall(
		gRPCClient,
		methodFullName,
		maxRetries,
		func(fullMethodName string, methodDescriptor protoreflect.MethodDescriptor) (T, error) {
			inputMsg := dynamicpb.NewMessage(methodDescriptor.Input())
			outputMsg := dynamicpb.NewMessage(methodDescriptor.Output())

			err := gRPCClient.Conn.Invoke(gRPCClient.Ctx, fullMethodName, inputMsg, outputMsg)
			if err != nil {
				var zero T
				return zero, errors.WithMessage(err, "error invoking method")
			}

			fieldValue := outputMsg.ProtoReflect().Get(outputMsg.Descriptor().Fields().ByName(protoreflect.Name(fieldName)))
			if !fieldValue.IsValid() {
				var zero T
				return zero, errors.New(fieldName + " field not found in response")
			}

			return converter(fieldValue.String())
		},
	)
}
