package yaci_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/manifest-network/yaci/cmd/yaci"
)

func TestExtractCmd(t *testing.T) {
	// --stop and --live are mutually exclusive
	_, err := executeCommand(yaci.RootCmd, "extract", "postgres", "foobar", "--live", "--stop", "10")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "cannot set --live and --stop flags together")

	// Show help
	output, err := executeCommand(yaci.RootCmd, "extract")
	assert.NoError(t, err)
	assert.Contains(t, output, "Extract chain data to")
}
