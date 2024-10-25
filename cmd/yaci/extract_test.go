package yaci_test

import (
	"testing"

	"github.com/liftedinit/yaci/cmd/yaci"
	"github.com/stretchr/testify/assert"
)

func TestExtractCmd(t *testing.T) {
	// --stop and --live are mutually exclusive
	_, err := executeCommand(yaci.RootCmd, "extract", "json", "foobar", "--live", "--stop", "10")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "if any flags in the group [live stop] are set none of the others can be; [live stop] were all set")

	// Show help
	output, err := executeCommand(yaci.RootCmd, "extract")
	assert.NoError(t, err)
	assert.Contains(t, output, "Extract chain data to")
}
