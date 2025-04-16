package utils

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/liftedinit/yaci/internal/client"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ParseMethodFullName parses a gRPC method full name into service name and method name
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

// BuildFullMethodName constructs the full method name from the method descriptor
func BuildFullMethodName(methodDescriptor protoreflect.MethodDescriptor) string {
	fullMethodName := "/" + string(methodDescriptor.FullName())
	lastDot := strings.LastIndex(fullMethodName, ".")
	if lastDot != -1 {
		fullMethodName = fullMethodName[:lastDot] + "/" + fullMethodName[lastDot+1:]
	}
	return fullMethodName
}

// InvokeGRPC invokes a gRPC method and returns the response message
func invokeGRPC(
	gRPCClient *client.GRPCClient,
	fullMethodName string,
	methodDescriptor protoreflect.MethodDescriptor,
	inputParams []byte,
) (*dynamicpb.Message, error) {
	// Create request and response messages
	inputMsg := dynamicpb.NewMessage(methodDescriptor.Input())
	outputMsg := dynamicpb.NewMessage(methodDescriptor.Output())

	// Unmarshal input parameters if provided
	if len(inputParams) > 0 {
		uo := protojson.UnmarshalOptions{Resolver: gRPCClient.Resolver}
		if err := uo.Unmarshal(inputParams, inputMsg); err != nil {
			return nil, fmt.Errorf("failed to parse input parameters: %w", err)
		}
	}

	// Make the gRPC call
	err := gRPCClient.Conn.Invoke(gRPCClient.Ctx, fullMethodName, inputMsg, outputMsg)
	if err != nil {
		return nil, err
	}

	return outputMsg, nil
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
			outputMsg, err := invokeGRPC(gRPCClient, fullMethodName, methodDescriptor, nil)
			if err != nil {
				var zero T
				return zero, fmt.Errorf("error invoking method: $%w", err)
			}

			fieldValue := outputMsg.ProtoReflect().Get(outputMsg.Descriptor().Fields().ByName(protoreflect.Name(fieldName)))
			if !fieldValue.IsValid() {
				var zero T
				return zero, fmt.Errorf("field `%s` not found in response: %w", fieldName, err)
			}

			return converter(fieldValue.String())
		},
	)
}

// GetGRPCResponse calls a gRPC method and returns the response as JSON
func GetGRPCResponse(
	gRPCClient *client.GRPCClient,
	methodFullName string,
	maxRetries uint,
	inputParams []byte,
) ([]byte, error) {
	return RetryGRPCCall(
		gRPCClient,
		methodFullName,
		maxRetries,
		func(fullMethodName string, methodDescriptor protoreflect.MethodDescriptor) ([]byte, error) {
			outputMsg, err := invokeGRPC(gRPCClient, fullMethodName, methodDescriptor, inputParams)
			if err != nil {
				return nil, fmt.Errorf("error invoking method: %w", err)
			}

			// Marshal the response to JSON
			mo := protojson.MarshalOptions{Resolver: gRPCClient.Resolver}
			responseBytes, err := mo.Marshal(outputMsg)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return responseBytes, nil
		},
	)
}
