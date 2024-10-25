package yaci_test

import (
	"bytes"
	"testing"

	"github.com/liftedinit/yaci/cmd/yaci"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const testEndpoint = "manifest-testnet-rpc.liftedinit.tech:443"

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	_, err = root.ExecuteC()
	return buf.String(), err
}

func TestRootCmd(t *testing.T) {
	// Show help
	output, err := executeCommand(yaci.RootCmd)
	assert.NoError(t, err)
	assert.Contains(t, output, "yaci connects to a gRPC server and extracts blockchain data.")

	// Test invalid logLevel
	output, err = executeCommand(yaci.RootCmd, "version", "--logLevel", "invalid")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid log level: invalid. Valid log levels are: debug|error|info|warn")
}
