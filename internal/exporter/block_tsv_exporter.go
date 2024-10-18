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

func ExportBlocksTSV(inputDir, outputPath string) error {
	blocksDir := filepath.Join(inputDir, "block")
	files, err := ioutil.ReadDir(blocksDir)
	if err != nil {
		return fmt.Errorf("failed to read blocks directory: %v", err)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create blocks TSV file: %v", err)
	}
	defer outputFile.Close()
	writer := bufio.NewWriter(outputFile)

	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), "block_") || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		filePath := filepath.Join(blocksDir, file.Name())
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read block file '%s': %v", file.Name(), err)
		}

		// Extract block height from file name
		parts := strings.Split(file.Name(), "_")
		if len(parts) != 2 {
			continue
		}
		idPart := strings.TrimSuffix(parts[1], ".json")
		idStr := strings.TrimLeft(idPart, "0")
		if idStr == "" {
			idStr = "0"
		}

		// Remove whitespace from JSON data
		compactData := new(bytes.Buffer)
		err = json.Compact(compactData, data)
		if err != nil {
			return fmt.Errorf("failed to compact JSON data for block '%s': %v", file.Name(), err)
		}

		// Write to TSV
		line := fmt.Sprintf("%s\t%s\n", idStr, compactData.String())
		_, err = writer.WriteString(line)
		if err != nil {
			return fmt.Errorf("failed to write to blocks TSV file: %v", err)
		}
	}

	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush blocks TSV file: %v", err)
	}

	return nil
}
