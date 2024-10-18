package cosmos_dump

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/liftedinit/cosmos-dump/internal/client"
	"github.com/liftedinit/cosmos-dump/internal/extractor"
	"github.com/liftedinit/cosmos-dump/internal/reflection"
	"github.com/liftedinit/cosmos-dump/internal/utils"
)

var (
	start    uint64
	stop     uint64
	out      string
	insecure bool
)

var extractCmd = &cobra.Command{
	Use:   "extract [address] [flags]",
	Short: "Extract block and transaction chain data to JSON files.",
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
		grpcConn, refClient := client.NewGRPCClients(ctx, address, insecure)
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
		err = extractor.ExtractBlocksAndTransactions(ctx, grpcConn, resolver, start, stop, out)
		if err != nil {
			return fmt.Errorf("failed to process blocks and transactions: %v", err)
		}

		return nil
	},
}

func init() {
	extractCmd.Flags().Uint64VarP(&start, "start", "s", 1, "Start block height")
	extractCmd.Flags().Uint64VarP(&stop, "stop", "e", 1, "Stop block height")
	extractCmd.Flags().StringVarP(&out, "out", "o", "out", "Output directory")
	extractCmd.Flags().BoolVarP(&insecure, "insecure", "k", false, "Skip TLS certificate verification")
}
