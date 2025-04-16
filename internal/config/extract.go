package config

import (
	"fmt"
	"net"
	"strconv"

	"github.com/spf13/viper"
)

type ExtractConfig struct {
	MaxConcurrency       uint
	MaxRetries           uint
	BlockTime            uint
	BlockStart           uint64
	BlockStop            uint64
	LiveMonitoring       bool
	Insecure             bool
	ReIndex              bool
	MaxRecvMsgSize       int
	EnablePrometheus     bool
	PrometheusListenAddr string
}

func (c ExtractConfig) Validate() error {
	if c.LiveMonitoring && c.BlockStop != 0 {
		return fmt.Errorf("cannot set --live and --stop flags together")
	}

	if c.EnablePrometheus {
		host, port, err := net.SplitHostPort(c.PrometheusListenAddr)
		if err != nil {
			return fmt.Errorf("invalid prometheus-addr format, expected host:port: %w", err)
		}

		if _, err := strconv.Atoi(port); err != nil {
			return fmt.Errorf("invalid port in prometheus-addr: %w", err)
		}

		if host != "" && host != "0.0.0.0" && host != "localhost" && net.ParseIP(host) == nil {
			return fmt.Errorf("invalid host in prometheus-addr: %s", host)
		}
	}
	return nil
}

func LoadExtractConfigFromCLI() ExtractConfig {
	return ExtractConfig{
		MaxConcurrency:       viper.GetUint("max-concurrency"),
		MaxRetries:           viper.GetUint("max-retries"),
		BlockTime:            viper.GetUint("block-time"),
		BlockStart:           viper.GetUint64("start"),
		BlockStop:            viper.GetUint64("stop"),
		LiveMonitoring:       viper.GetBool("live"),
		Insecure:             viper.GetBool("insecure"),
		ReIndex:              viper.GetBool("reindex"),
		MaxRecvMsgSize:       viper.GetInt("max-recv-msg-size"),
		EnablePrometheus:     viper.GetBool("enable-prometheus"),
		PrometheusListenAddr: viper.GetString("prometheus-addr"),
	}
}
