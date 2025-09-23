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

	"github.com/manifest-network/yaci/cmd/yaci"
	"github.com/manifest-network/yaci/internal/testutil"
)

const (
	DockerWorkingDirectory = "../../docker"
	GRPCEndpoint           = "localhost:9090"
	RestEndpoint           = "localhost:3000"
	PsqlConnectionString   = "postgres://postgres:foobar@localhost/postgres"
)

var (
	RestTxEndpoint    = fmt.Sprintf("http://%s/transactions_raw", RestEndpoint)
	RestBlockEndpoint = fmt.Sprintf("http://%s/blocks_raw", RestEndpoint)
)

func TestPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// Start the infrastructure using Docker Compose.
	// The infrastructure is defined in the `compose.yaml` file.
	opts := &docker.Options{WorkingDir: DockerWorkingDirectory + "/infra"}
	_, err := docker.RunDockerComposeE(t, opts, "up", "--build", "-d", "--wait")
	require.NoError(t, err)

	testExtractBlocksAndTxs(t)
	testResume(t)
	testMissingBlocks(t)
	testReindex(t)

	t.Cleanup(func() {
		// Stop the infrastructure using Docker Compose.
		_, err := docker.RunDockerComposeE(t, opts, "down", "-v")
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
		// The number of transactions is 42 as defined in the `compose.yaml` file under the `manifest-ledger-tx` service
		require.Len(t, transactions, 42)
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
		// Get the data field of block 3 using the REST API
		queryParams := map[string]string{
			"id": fmt.Sprintf("eq.%d", 3),
		}
		block3 := getJSONResponse(t, RestBlockEndpoint, queryParams)

		// Update the data field of block 3 to an empty JSON object
		_, err := docker.RunE(t, "postgres", &docker.RunOptions{
			Command:              []string{"psql", "-h", "localhost", "-U", "postgres", "-c", "UPDATE api.blocks_raw SET data = '{}' WHERE id = 3"},
			EnvironmentVariables: []string{"PGPASSWORD=foobar"},
			Detach:               false,
			Remove:               true,
			OtherOptions:         []string{"--network", "host"},
		})
		require.NoError(t, err)

		// Verify that the data field of block 3 is empty using the REST API
		emptyBlock3 := getJSONResponse(t, RestBlockEndpoint, queryParams)
		require.NotEmpty(t, emptyBlock3)
		require.Empty(t, emptyBlock3[0]["data"])

		// Execute the command. This will reindex the database from block 1 to the latest block.
		out, err := executeExtractCommand(t, "--reindex")
		require.NoError(t, err)
		require.Contains(t, out, "Starting extraction")
		require.Contains(t, out, "Reindexing entire database...")
		require.Contains(t, out, "\"start\":1")
		require.Contains(t, out, "Closing PostgreSQL connection pool")

		// Verify that the data field of block 3 is not empty using the REST API
		// The data field of block 3 should be the same as before setting the empty JSON object
		newBlock3 := getJSONResponse(t, RestBlockEndpoint, queryParams)
		require.NotEmpty(t, newBlock3)
		require.NotEmpty(t, newBlock3[0]["data"])
		require.Equal(t, block3, newBlock3)
	})
}

func TestPrometheusMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	opts := &docker.Options{WorkingDir: DockerWorkingDirectory + "/yaci"}
	_, err := docker.RunDockerComposeE(t, opts, "up", "--build", "-d", "--wait")
	require.NoError(t, err)

	testPrometheusMetrics(t)

	t.Cleanup(func() {
		_, err := docker.RunDockerComposeE(t, opts, "down", "-v")
		require.NoError(t, err)
	})
}

func testPrometheusMetrics(t *testing.T) {
	t.Run("TestPrometheusMetrics", func(t *testing.T) {
		resp, err := resty.New().R().Get("http://localhost:2112/metrics")
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		body := string(resp.Body())
		require.Contains(t, body, "yaci_addresses_total_unique_user{source=\"postgres\"} 3")
		require.Contains(t, body, "yaci_addresses_total_unique_group{source=\"postgres\"} 3")
		require.Contains(t, body, "yaci_transactions_total_count{source=\"postgres\"} 42")
		// 3000000umfx were burned by the MFX to PWR conversion
		// 123umfx were burned by a POA proposal
		require.Contains(t, body, "yaci_tokenomics_total_burn_amount{source=\"postgres\"} 3.000123e+06")
		require.Contains(t, body, "yaci_tokenomics_total_payout_amount{source=\"postgres\"} 7.54321e+06")
		// 6000000factory/.../upwr were minted by the MFX to PWR conversion
		require.Contains(t, body, "yaci_tokenomics_total_pwr_minted_amount{source=\"postgres\"} 6e+06")
		require.Contains(t, body, "yaci_locked_tokens_count{amount=\"2000000000\",denom=\"umfx\",source=\"postgres\"} 1")
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
