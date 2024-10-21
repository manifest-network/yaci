package cosmos_dump

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cosmos-dump",
	Short: "Extract chain data",
	Long:  `cosmos-dump connects to a gRPC server and extracts blockchain data.`,
}

func init() {
	rootCmd.AddCommand(extractCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
