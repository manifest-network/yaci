package cosmos_dump

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/liftedinit/cosmos-dump/internal/output"

	"github.com/liftedinit/cosmos-dump/internal/client"
	"github.com/liftedinit/cosmos-dump/internal/extractor"
	"github.com/liftedinit/cosmos-dump/internal/reflection"
	"github.com/liftedinit/cosmos-dump/internal/utils"
)

var (
	start      uint64
	stop       uint64
	out        string
	insecure   bool
	live       bool
	outputType string
	connString string
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

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle interrupt signals for graceful shutdown
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			fmt.Println("\nReceived interrupt signal, shutting down...")
			cancel()
		}()

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

		// Initialize the appropriate output handler
		var outputHandler output.OutputHandler

		switch outputType {
		case "json":
			outputHandler, err = output.NewJSONOutputHandler(out)
			if err != nil {
				return err
			}
		case "tsv":
			outputHandler, err = output.NewTSVOutputHandler(out)
			if err != nil {
				return err
			}
		case "postgres":
			if connString == "" {
				return fmt.Errorf("connection string is required for PostgreSQL output")
			}
			outputHandler, err = output.NewPostgresOutputHandler(connString)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid output type: %s", outputType)
		}

		defer outputHandler.Close()

		// Process blocks and transactions
		if live {
			// Live mode: Monitor the chain for new blocks
			err = extractor.ExtractLiveBlocksAndTransactions(ctx, grpcConn, resolver, start, outputHandler)
			if err != nil {
				return fmt.Errorf("failed to process live blocks and transactions: %v", err)
			}
		} else {
			// Batch mode: Process blocks in the given range
			err = extractor.ExtractBlocksAndTransactions(ctx, grpcConn, resolver, start, stop, outputHandler)
			if err != nil {
				return fmt.Errorf("failed to process blocks and transactions: %v", err)
			}
		}

		return nil
	},
}

func init() {
	extractCmd.Flags().Uint64VarP(&start, "start", "s", 1, "Start block height")
	extractCmd.Flags().Uint64VarP(&stop, "stop", "e", 1, "Stop block height")
	extractCmd.Flags().StringVarP(&out, "out", "o", "out", "Output directory")
	extractCmd.Flags().BoolVarP(&insecure, "insecure", "k", false, "Skip TLS certificate verification")
	extractCmd.Flags().BoolVar(&live, "live", false, "Enable live monitoring mode")
	extractCmd.Flags().StringVarP(&outputType, "output", "t", "json", "Output type: json, tsv, postgres")
	extractCmd.Flags().StringVar(&connString, "conn", "", "Connection string for PostgreSQL output")
}
