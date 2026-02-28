package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	TLS      bool   `mapstructure:"tls"`
}

type ScannerConfig struct {
	BatchSize int `mapstructure:"batch_size"`
	Workers   int `mapstructure:"workers"`
}

type GroupConfig struct {
	Name    string `mapstructure:"name"`
	Pattern string `mapstructure:"pattern"`
}

type OutputConfig struct {
	Format          string `mapstructure:"format"`
	SortBy          string `mapstructure:"sort_by"`
	CliffThreshold  int    `mapstructure:"cliff_threshold"`
	CliffWindowMins int    `mapstructure:"cliff_window_minutes"`
}

type Config struct {
	Redis   RedisConfig   `mapstructure:"redis"`
	Scanner ScannerConfig `mapstructure:"scanner"`
	Groups  []GroupConfig `mapstructure:"groups"`
	Output  OutputConfig  `mapstructure:"output"`
}

func Load(cfgFile string) (*Config, error) {
	if cfgFile != "" {
		// user explicitly specified config file path via --config flag
		viper.SetConfigFile(cfgFile)
	} else {
		// no flag given, look for config.yaml in the current directory
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.tls", false)
	viper.SetDefault("scanner.batch_size", 100)
	viper.SetDefault("scanner.workers", 20)
	viper.SetDefault("output.format", "table")
	viper.SetDefault("output.sort_by", "memory")
	viper.SetDefault("output.cliff_threshold", 20)
	viper.SetDefault("output.cliff_window_minutes", 5)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &cfg, nil
}