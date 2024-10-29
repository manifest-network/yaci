package yaci

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	validLogLevels = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	validLogLevelsStr = strings.Join(slices.Sorted(maps.Keys(validLogLevels)), "|")
)

var RootCmd = &cobra.Command{
	Use:   "yaci",
	Short: "Extract chain data",
	Long:  `yaci connects to a gRPC server and extracts blockchain data.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		logLevel := viper.GetString("logLevel")
		if err := setLogLevel(logLevel); err != nil {
			return err
		}
		slog.Debug("Application started", "version", Version)
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
	RootCmd.PersistentFlags().StringP("logLevel", "l", "info", fmt.Sprintf("set log level (%s)", validLogLevelsStr))
	if err := viper.BindPFlags(RootCmd.PersistentFlags()); err != nil {
		slog.Error("Failed to bind rootCmd flags", "error", err)
	}

	RootCmd.SilenceUsage = true
	RootCmd.SilenceErrors = true

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.yaci")
	viper.AddConfigPath("/etc/yaci")

	viper.SetEnvPrefix("yaci")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	RootCmd.AddCommand(ExtractCmd)
	RootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() {
	if err := viper.ReadInConfig(); err == nil {
		slog.Info("Using config file", "file", viper.ConfigFileUsed())
	} else {
		slog.Info("No config file found")
	}

	if err := RootCmd.Execute(); err != nil {
		slog.Error("An error occurred", "error", err)
		os.Exit(1)
	}
}
