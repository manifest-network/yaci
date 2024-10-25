package yaci

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var (
	validLogLevels = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	validLogLevelsStr = strings.Join(slices.Sorted(maps.Keys(validLogLevels)), "|")
	logLevel          string
)

var RootCmd = &cobra.Command{
	Use:   "yaci",
	Short: "Extract chain data",
	Long:  `yaci connects to a gRPC server and extracts blockchain data.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

		if err := setLogLevel(logLevel); err != nil {
			return err
		}
		slog.Info("Application started", "version", Version)
		return nil
	},
}

// setLogLevel sets the log level
func setLogLevel(logLevel string) error {
	level, exists := validLogLevels[logLevel]
	if !exists {
		return fmt.Errorf("invalid log level: %s. Valid log levels are: %s", logLevel, validLogLevelsStr)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	return nil
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&logLevel, "logLevel", "l", "info", fmt.Sprintf("set log level (%s)", validLogLevelsStr))

	RootCmd.SilenceUsage = true
	RootCmd.SilenceErrors = true // Handled in Execute()

	RootCmd.AddCommand(ExtractCmd)
	RootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		slog.Error("An error occurred", "error", err)
		os.Exit(1)
	}
}
