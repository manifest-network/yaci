package yaci

import (
	"log/slog"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/liftedinit/yaci/internal/output"
)

var tsvCmd = &cobra.Command{
	Use:   "tsv [address] [flags]",
	Short: "Extract chain data to TSV files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tsvOut := viper.GetString("tsv-out")
		slog.Debug("Command-line argument", "tsv-out", tsvOut)

		err := os.MkdirAll(tsvOut, 0755)
		if err != nil {
			return errors.WithMessage(err, "failed to create output directory")
		}

		outputHandler, err := output.NewTSVOutputHandler(tsvOut)
		if err != nil {
			return errors.WithMessage(err, "failed to create TSV output handler")
		}
		defer outputHandler.Close()

		return extract(args[0], outputHandler)
	},
}

func init() {
	tsvCmd.Flags().StringP("tsv-out", "o", "tsv", "Output directory")
	if err := viper.BindPFlags(tsvCmd.Flags()); err != nil {
		slog.Error("Failed to bind tsvCmd flags", "error", err)
	}
}
