package yaci_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/gruntwork-io/terratest/modules/docker"
	"github.com/liftedinit/yaci/cmd/yaci"
	"github.com/liftedinit/yaci/internal/testutil"
	"github.com/stretchr/testify/require"
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

	t.Cleanup(func() {
		// Stop the infrastructure using Docker Compose.
		_, err := docker.RunDockerComposeE(t, opts, "-f", "infra.yml", "down", "-v")
		require.NoError(t, err)
	})
}

func testExtractBlocksAndTxs(t *testing.T) {
	t.Run("TestExtractBlocksAndTxs", func(t *testing.T) {
		// Execute the command. This will extract the chain data to a PostgreSQL database up to the latest block.
		out, err := testutil.Execute(t, yaci.RootCmd, "extract", "postgres", GRPCEndpoint, "-p", PsqlConnectionString, "-k")
		require.NoError(t, err)
		require.Contains(t, out, "Starting extraction")
		require.Contains(t, out, "Closing PostgreSQL connection pool")

		// Verify that the chain data has been extracted to the PostgreSQL database using the REST API
		client := resty.New()
		resp, err := client.
			R().
			SetHeader("Accept", "application/json").
			Get(RestTxEndpoint)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotEmpty(t, resp.Body())

		// Parse the response JSON body
		var transactions []map[string]interface{}
		err = json.Unmarshal(resp.Body(), &transactions)
		require.NoError(t, err)
		require.NotEmpty(t, transactions)

		// The number of transactions is 6 as defined in the `infra.yml` file under the `manifest-ledger-tx` service
		require.Len(t, transactions, 6)
	})
}

func testResume(t *testing.T) {
	t.Run("TestResume", func(t *testing.T) {
		beforeBlockId := getLastBlockId(t)

		// Execute the command. This will resume the extraction from the last block that was extracted.
		out, err := testutil.Execute(t, yaci.RootCmd, "extract", "postgres", GRPCEndpoint, "-p", PsqlConnectionString, "-k")
		require.NoError(t, err)
		require.Contains(t, out, "Starting extraction")
		require.Contains(t, out, "Closing PostgreSQL connection pool")

		afterBlockId := getLastBlockId(t)
		require.Greater(t, afterBlockId, beforeBlockId)
	})
}

func getLastBlockId(t *testing.T) int {
	t.Helper()

	client := resty.New()
	resp, err := client.
		R().
		SetHeader("Accept", "application/json").
		SetQueryParam("order", "id.desc").
		SetQueryParam("limit", "1").
		Get(RestBlockEndpoint)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())

	var blocks []map[string]interface{}
	err = json.Unmarshal(resp.Body(), &blocks)
	require.NoError(t, err)
	require.NotEmpty(t, blocks)
	require.Len(t, blocks, 1)

	lastBlockId := blocks[0]["id"]
	require.NotNil(t, lastBlockId)

	lastBlockIdFloat, ok := lastBlockId.(float64)
	require.True(t, ok)

	return int(lastBlockIdFloat)
}
