package extractor

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/liftedinit/yaci/internal/client"
	"github.com/liftedinit/yaci/internal/models"
	"github.com/liftedinit/yaci/internal/utils"
)

func extractTransactions(gRPCClient *client.GRPCClient, data map[string]interface{}, maxRetries uint) ([]*models.Transaction, error) {
	blockData, exists := data["block"].(map[string]interface{})
	if !exists || blockData == nil {
		return nil, nil
	}

	dataField, exists := blockData["data"].(map[string]interface{})
	if !exists || dataField == nil {
		return nil, nil
	}

	txs, exists := dataField["txs"].([]interface{})
	if !exists {
		return nil, nil
	}

	var transactions []*models.Transaction
	for _, tx := range txs {
		txStr, ok := tx.(string)
		if !ok {
			continue
		}
		decodedBytes, err := base64.StdEncoding.DecodeString(txStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode tx: %w", err)
		}
		hash := sha256.Sum256(decodedBytes)
		hashStr := hex.EncodeToString(hash[:])

		txJsonParams := []byte(fmt.Sprintf(`{"hash": "%s"}`, hashStr))
		txJsonBytes, err := utils.GetGRPCResponse(
			gRPCClient,
			txMethodFullName,
			maxRetries,
			txJsonParams,
		)

		transaction := &models.Transaction{
			Hash: hashStr,
			Data: txJsonBytes,
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}
