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
}

// Global is the global configuration instance.
var Global *Config
var AppLogger logger.ILogger

// LoadConfig loads configuration from config file.
func LoadConfig() error {
	if AppLogger == nil {
		AppLogger = logger.NewLogger()
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	viper.SetDefault("log_level", "debug")
	viper.SetDefault("binary_path", "/tmp/main")
	viper.SetDefault("debounce_time", 2000)

	if err := viper.ReadInConfig(); err != nil {
		AppLogger.Fatal("Failed to read config file", zap.Error(err))
		return err
	}
	if err := viper.Unmarshal(&Global); err != nil {
		AppLogger.Fatal("Failed to process config", zap.Error(err))
		return err
	}

	return nil
}
