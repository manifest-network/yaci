package exporter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func ExportTransactionsTSV(inputDir, outputPath string) error {
	txsDir := filepath.Join(inputDir, "txs")
	files, err := ioutil.ReadDir(txsDir)
	if err != nil {
		return fmt.Errorf("failed to read transactions directory: %v", err)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create transactions TSV file: %v", err)
	}
	defer outputFile.Close()
	writer := bufio.NewWriter(outputFile)

	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), "tx_") || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		filePath := filepath.Join(txsDir, file.Name())
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read transaction file '%s': %v", file.Name(), err)
		}

		// Extract transaction hash from file name
		parts := strings.Split(file.Name(), "_")
		if len(parts) < 3 {
			continue
		}
		hashPart := parts[2]
		hash := strings.TrimSuffix(hashPart, ".json")

		// Remove whitespace from JSON data
		compactData := new(bytes.Buffer)
		err = json.Compact(compactData, data)
		if err != nil {
			return fmt.Errorf("failed to compact JSON data for transaction '%s': %v", file.Name(), err)
		}

		// Write to TSV
		line := fmt.Sprintf("%s\t%s\n", hash, compactData.String())
		_, err = writer.WriteString(line)
		if err != nil {
			return fmt.Errorf("failed to write to transactions TSV file: %v", err)
		}
	}

	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush transactions TSV file: %v", err)
	}

	return nil
}
