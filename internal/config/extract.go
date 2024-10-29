package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type ExtractConfig struct {
	MaxConcurrency uint
	MaxRetries     uint
	BlockTime      uint
	BlockStart     uint64
	BlockStop      uint64
	LiveMonitoring bool
	Insecure       bool
}

func (c ExtractConfig) Validate() error {
	if c.LiveMonitoring && c.BlockStop != 0 {
		return fmt.Errorf("cannot set --live and --stop flags together")
	}
	return nil
}

func LoadExtractConfigFromCLI() ExtractConfig {
	return ExtractConfig{
		MaxConcurrency: viper.GetUint("max-concurrency"),
		MaxRetries:     viper.GetUint("max-retries"),
		BlockTime:      viper.GetUint("block-time"),
		BlockStart:     viper.GetUint64("start"),
		BlockStop:      viper.GetUint64("stop"),
		LiveMonitoring: viper.GetBool("live"),
		Insecure:       viper.GetBool("insecure"),
	}
}
