package yaci_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gruntwork-io/terratest/modules/docker"
	"github.com/stretchr/testify/require"

	"github.com/liftedinit/yaci/cmd/yaci"
	"github.com/liftedinit/yaci/internal/testutil"
)

const (
	DockerWorkingDirectory = "../../docker"
	GRPCEndpoint           = "localhost:9090"
	RestEndpoint           = "localhost:3000"
	PsqlConnectionString   = "postgres://postgres:foobar@localhost/postgres"
)

var (
	RestTxEndpoint    = fmt.Sprintf("http://%s/transactions", RestEndpoint)
	RestBlockEndpoint = fmt.Sprintf("http://%s/blocks", RestEndpoint)
)

func TestPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// Start the infrastructure using Docker Compose.
	// The infrastructure is defined in the `infra.yml` file.
	opts := &docker.Options{WorkingDir: DockerWorkingDirectory}
	_, err := docker.RunDockerComposeE(t, opts, "-f", "infra.yml", "up", "-d", "--wait")
	require.NoError(t, err)

	testExtractBlocksAndTxs(t)
	testResume(t)
	testMissingBlocks(t)
	testReindex(t)

	t.Cleanup(func() {
		// Stop the infrastructure using Docker Compose.
		_, err := docker.RunDockerComposeE(t, opts, "-f", "infra.yml", "down", "-v")
		require.NoError(t, err)
	})
}

func testExtractBlocksAndTxs(t *testing.T) {
	t.Run("TestExtractBlocksAndTxs", func(t *testing.T) {
		// Execute the command. This will extract the chain data to a PostgreSQL database up to the latest block.
		out, err := executeExtractCommand(t)
		require.NoError(t, err)
		require.Contains(t, out, "Starting extraction")
		require.Contains(t, out, "Closing PostgreSQL connection pool")

		transactions := getJSONResponse(t, RestTxEndpoint, nil)
		require.NotEmpty(t, transactions)
		// The number of transactions is 6 as defined in the `infra.yml` file under the `manifest-ledger-tx` service
		require.Len(t, transactions, 6)
	})
}

func testResume(t *testing.T) {
	t.Run("TestResume", func(t *testing.T) {
		beforeBlockId := getLastBlockId(t)

		// Execute the command. This will resume the extraction from the last block that was extracted.
		out, err := executeExtractCommand(t)
		require.NoError(t, err)
		require.Contains(t, out, "Starting extraction")
		require.Contains(t, out, "Closing PostgreSQL connection pool")

		afterBlockId := getLastBlockId(t)
		require.Greater(t, afterBlockId, beforeBlockId)
	})
}

func testMissingBlocks(t *testing.T) {
	t.Run("TestMissingBlocks", func(t *testing.T) {
		beforeBlockId := getLastBlockId(t)
		missingBlockId := beforeBlockId + 1
		blockAfterNext := beforeBlockId + 2

		// Extract the block after the next block, creating a gap in the database
		out, err := executeExtractCommand(t, "-s", strconv.Itoa(blockAfterNext), "-e", strconv.Itoa(blockAfterNext))
		require.NoError(t, err)
		require.Contains(t, out, "Starting extraction")
		require.Contains(t, out, "Closing PostgreSQL connection pool")

		afterBlockId := getLastBlockId(t)
		require.Equal(t, blockAfterNext, afterBlockId)

		// Make sure there is a gap in the database
		_, ok := maybeGetBlockId(t, missingBlockId)
		require.False(t, ok)

		// Create some blocks
		time.Sleep(10 * time.Second)

		// Execute the command. This will extract the missing block to the PostgreSQL database.
		out, err = executeExtractCommand(t, "-s", "0", "-e", "0")
		require.NoError(t, err)
		require.Contains(t, out, "Starting extraction")
		require.Contains(t, out, "Missing blocks detected")
		require.Contains(t, out, "Closing PostgreSQL connection pool")

		// Verify that the missing block has been extracted to the PostgreSQL database using the REST API
		missingBlock, ok := maybeGetBlockId(t, missingBlockId)
		require.True(t, ok)
		require.Equal(t, missingBlockId, missingBlock)
	})
}

func testReindex(t *testing.T) {
	t.Run("TestReindex", func(t *testing.T) {
		// Execute the command. This will reindex the database from block 1 to the latest block.
		out, err := executeExtractCommand(t, "--reindex")
		require.NoError(t, err)
		require.Contains(t, out, "Starting extraction")
		require.Contains(t, out, "Reindexing entire database...")
		require.Contains(t, out, "\"start\":1")
		require.Contains(t, out, "Closing PostgreSQL connection pool")
	})
}

func getBlockIdFromMap(t *testing.T, block map[string]interface{}) int {
	t.Helper()
	blockId, ok := block["id"].(float64)
	require.True(t, ok)
	return int(blockId)
}

func maybeGetBlockId(t *testing.T, blockHeight int) (int, bool) {
	t.Helper()

	queryParams := map[string]string{
		"id": fmt.Sprintf("eq.%d", blockHeight),
	}
	blocks := getJSONResponse(t, RestBlockEndpoint, queryParams)

	if len(blocks) == 0 {
		return 0, false
	}

	require.Len(t, blocks, 1)
	blockId := getBlockIdFromMap(t, blocks[0])
	return blockId, true
}

func getLastBlockId(t *testing.T) int {
	t.Helper()

	queryParams := map[string]string{
		"order": "id.desc",
		"limit": "1",
	}
	blocks := getJSONResponse(t, RestBlockEndpoint, queryParams)
	require.NotEmpty(t, blocks)
	require.Len(t, blocks, 1)

	return getBlockIdFromMap(t, blocks[0])
}

func executeExtractCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	baseArgs := []string{"extract", "postgres", GRPCEndpoint, "-p", PsqlConnectionString, "-k"}
	return testutil.Execute(t, yaci.RootCmd, append(baseArgs, args...)...)
}

func getJSONResponse(t *testing.T, endpoint string, queryParams map[string]string) []map[string]interface{} {
	t.Helper()

	client := resty.New()
	req := client.R().SetHeader("Accept", "application/json")
	if queryParams != nil {
		req.SetQueryParams(queryParams)
	}
	resp, err := req.Get(endpoint)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())

	var data []map[string]interface{}
	err = json.Unmarshal(resp.Body(), &data)
	require.NoError(t, err)

	return data
}
