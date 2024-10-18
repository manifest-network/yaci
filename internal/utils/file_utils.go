package utils

import (
	"fmt"
	"os"
)

// SetupOutputDirectories checks and creates the necessary output directories.
func SetupOutputDirectories(out string) error {
	outBlocks := fmt.Sprintf("%s/block/", out)
	outTxs := fmt.Sprintf("%s/txs/", out)

	// Check if the directory exists
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' already exists", out)
	}

	err := os.MkdirAll(outBlocks, 0755)
	if err != nil {
		return fmt.Errorf("failed to create blocks directory: %v", err)
	}

	err = os.MkdirAll(outTxs, 0755)
	if err != nil {
		return fmt.Errorf("failed to create txs directory: %v", err)
	}

	return nil
}
