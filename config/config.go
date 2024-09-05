package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	BinaryPath   string        `mapstructure:"binary_path"`
	DebounceTime time.Duration `mapstructure:"debounce_time"`
}

var Global *Config

func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	viper.SetDefault("log_level", "debug")
	viper.SetDefault("binary_path", "/tmp/main")
	viper.SetDefault("debounce_time", 2000)

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Erro ao ler arquivo de configuração: %v", err)
		return err
	}

	if err := viper.Unmarshal(&Global); err != nil {
		log.Fatalf("Erro ao processar a configuração: %v", err)
		return err
	}

	return nil
}
