package extractor

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/liftedinit/cosmos-dump/internal/models"
	"github.com/liftedinit/cosmos-dump/internal/output"
)

func extractTransactions(ctx context.Context, conn *grpc.ClientConn, data map[string]interface{}, txMethodDescriptor protoreflect.MethodDescriptor, txFullMethodName string, blockHeight uint64, outputHandler output.OutputHandler, uo protojson.UnmarshalOptions, mo protojson.MarshalOptions) error {
	blockData, exists := data["block"].(map[string]interface{})
	if !exists || blockData == nil {
		return nil
	}

	dataField, exists := blockData["data"].(map[string]interface{})
	if !exists || dataField == nil {
		return nil
	}

	txs, exists := dataField["txs"].([]interface{})
	if !exists {
		return nil
	}

	for _, tx := range txs {
		txStr, ok := tx.(string)
		if !ok {
			continue
		}
		decodedBytes, err := base64.StdEncoding.DecodeString(txStr)
		if err != nil {
			return errors.WithMessage(err, "failed to decode tx")
		}
		hash := sha256.Sum256(decodedBytes)
		hashStr := hex.EncodeToString(hash[:])

		txInputMsg := dynamicpb.NewMessage(txMethodDescriptor.Input())
		txJsonParams := fmt.Sprintf(`{"hash": "%s"}`, hashStr)
		if err := uo.Unmarshal([]byte(txJsonParams), txInputMsg); err != nil {
			return errors.WithMessage(err, "failed to parse tx input parameters")
		}
		txOutputMsg := dynamicpb.NewMessage(txMethodDescriptor.Output())

		err = conn.Invoke(ctx, txFullMethodName, txInputMsg, txOutputMsg)
		if err != nil {
			return errors.WithMessage(err, "error invoking tx method")
		}
		txJsonBytes, err := mo.Marshal(txOutputMsg)
		if err != nil {
			return errors.WithMessage(err, "failed to marshal tx response")
		}

		transaction := &models.Transaction{
			Hash: hashStr,
			Data: txJsonBytes,
		}

		err = outputHandler.WriteTransaction(ctx, transaction)
		if err != nil {
			return errors.WithMessage(err, "failed to write transaction")
		}
	}

	return nil
}
