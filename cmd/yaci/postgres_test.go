package yaci_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/gruntwork-io/terratest/modules/docker"
	"github.com/liftedinit/yaci/cmd/yaci"
	"github.com/stretchr/testify/require"
)

const (
	DockerWorkingDirectory = "../../docker"
	GRPCEndpoint           = "localhost:9090"
	RestEndpoint           = "localhost:3000"
	PsqlConnectionString   = "postgres://postgres:foobar@localhost/postgres"
)

var (
	RestTxEndpoint = fmt.Sprintf("http://%s/transactions", RestEndpoint)
)

func TestPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// Start the infrastructure using Docker Compose.
	// The infrastructure is defined in the `infra.yml` file.
	opts := &docker.Options{WorkingDir: DockerWorkingDirectory}
	defer docker.RunDockerCompose(t, opts, "-f", "infra.yml", "down", "-v")
	_, err := docker.RunDockerComposeE(t, opts, "-f", "infra.yml", "up", "-d", "--wait")
	require.NoError(t, err)

	// Run the YACI command to Extract the chain data to a PostgreSQL database
	cmd := yaci.RootCmd
	cmd.SetArgs([]string{"extract", "postgres", GRPCEndpoint, "-p", PsqlConnectionString, "-k"})

	// Execute the command. This will Extract the chain data to a PostgreSQL database up to the latest block.
	err = cmd.Execute()
	require.NoError(t, err)

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
}
