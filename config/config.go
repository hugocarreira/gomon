package config

import (
	"time"

	"github.com/hugocarreira/gomon/logger"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Config holds the application configuration.
type Config struct {
	BinaryPath   string        `mapstructure:"binary_path"`
	DebounceTime time.Duration `mapstructure:"debounce_time"`
	LogLevel     string        `mapstructure:"log_level"`
}

// Global is the global configuration instance.
var Global *Config
var AppLogger logger.ILogger

// LoadConfig loads configuration from environment variables and sets defaults.
func LoadConfig(log logger.ILogger) error {
	viper.SetDefault("log_level", "debug")
	viper.SetDefault("binary_path", "/tmp/main")
	viper.SetDefault("debounce_time", 2000)

	if err := viper.Unmarshal(&Global); err != nil {
		AppLogger.Fatal("Failed to process config", zap.Error(err))
		return err
	}

	return nil
}

func (c *Config) SetupOverrides(binaryPath, logLevel string, debounce int) {
	if binaryPath != "" {
		c.BinaryPath = binaryPath
	}
	if logLevel != "" {
		c.LogLevel = logLevel
	}
	if debounce != 0 {
		c.DebounceTime = time.Duration(debounce) * time.Millisecond
	}
}
