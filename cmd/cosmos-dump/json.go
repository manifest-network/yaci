package cosmos_dump

import (
	"os"

	"github.com/liftedinit/cosmos-dump/internal/output"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var jsonOut string

var jsonCmd = &cobra.Command{
	Use:   "json [address] [flags]",
	Short: "Extract chain data to JSON files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := os.MkdirAll(jsonOut, 0755)
		if err != nil {
			return errors.WithMessage(err, "failed to create output directory")
		}

		outputHandler, err := output.NewJSONOutputHandler(jsonOut)
		if err != nil {
			return errors.WithMessage(err, "failed to create JSON output handler")
		}
		defer outputHandler.Close()

		return extract(args[0], outputHandler)
	},
}

func init() {
	jsonCmd.Flags().StringVarP(&jsonOut, "out", "o", "out", "Output directory")
}
