package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type TSVConfig struct {
	Output string
}

func (c TSVConfig) Validate() error {
	if c.Output == "" {
		return fmt.Errorf("missing output directory")
	}
	return nil
}

func LoadTSVConfigFromCLI() TSVConfig {
	return TSVConfig{
		Output: viper.GetString("tsv-out"),
	}
}
