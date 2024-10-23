package yaci

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of yaci",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("yaci", Version)
	},
}
