package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type JSONConfig struct {
	Output string
}

func (c JSONConfig) Validate() error {
	if c.Output == "" {
		return fmt.Errorf("missing output directory")
	}
	return nil
}

func LoadJSONConfigFromCLI() JSONConfig {
	return JSONConfig{
		Output: viper.GetString("json-out"),
	}
}
