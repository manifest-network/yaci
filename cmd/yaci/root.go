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

var rootCmd = &cobra.Command{
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
	rootCmd.PersistentFlags().StringP("logLevel", "l", "info", fmt.Sprintf("set log level (%s)", validLogLevelsStr))
	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		slog.Error("Failed to bind rootCmd flags", "error", err)
	}

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.yaci")
	viper.AddConfigPath("/etc/yaci")

	viper.SetEnvPrefix("yaci")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	rootCmd.AddCommand(ExtractCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() {
	if err := viper.ReadInConfig(); err == nil {
		slog.Info("Using config file", "file", viper.ConfigFileUsed())
	} else {
		slog.Info("No config file found")
	}

	if err := rootCmd.Execute(); err != nil {
		slog.Error("An error occurred", "error", err)
		os.Exit(1)
	}
}
