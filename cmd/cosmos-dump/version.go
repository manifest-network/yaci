package cosmos_dump

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of cosmos-dump",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cosmos-dump", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
