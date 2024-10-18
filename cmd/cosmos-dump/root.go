package cosmos_dump

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/liftedinit/cosmos-dump/internal/client"
	"github.com/liftedinit/cosmos-dump/internal/processing"
	"github.com/liftedinit/cosmos-dump/internal/reflection"
	"github.com/liftedinit/cosmos-dump/internal/utils"

	"github.com/spf13/cobra"
)

var (
	start uint64
	stop  uint64
	out   string
)

var rootCmd = &cobra.Command{
	Use:   "cosmos-dump [address] [flags]",
	Short: "Extract block and transaction chain data to JSON files.",
	Long:  `cosmos-dump connects to a gRPC server and extracts blockchain data to JSON files.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		address := args[0]
		_, err := url.ParseRequestURI(address)
		if err != nil {
			return fmt.Errorf("invalid address: %v", err)
		}

		// Setup output directories
		err = utils.SetupOutputDirectories(out)
		if err != nil {
			return fmt.Errorf("failed to setup output directories: %v", err)
		}

		ctx := context.Background()

		// Initialize gRPC client and reflection client
		grpcConn, refClient := client.NewGRPCClients(ctx, address)
		defer grpcConn.Close()

		// Fetch all file descriptors
		descriptors, err := reflection.FetchAllDescriptors(ctx, refClient)
		if err != nil {
			return fmt.Errorf("failed to fetch descriptors: %v", err)
		}

		// Build the file descriptor set
		files, err := reflection.BuildFileDescriptorSet(descriptors)
		if err != nil {
			return fmt.Errorf("failed to build descriptor set: %v", err)
		}

		// Create a custom resolver
		resolver := reflection.NewCustomResolver(files, refClient, ctx)

		// Process blocks and transactions
		err = processing.ProcessBlocksAndTransactions(ctx, grpcConn, resolver, start, stop, out)
		if err != nil {
			return fmt.Errorf("failed to process blocks and transactions: %v", err)
		}

		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().Uint64VarP(&start, "start", "s", 1, "Start block height")
	rootCmd.Flags().Uint64VarP(&stop, "stop", "e", 1, "Stop block height")
	rootCmd.Flags().StringVarP(&out, "out", "o", "out", "Output directory")
}
