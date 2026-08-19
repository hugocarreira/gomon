package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hugocarreira/gomon/logger"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	defaultConfigName   = ".gomon.yaml"
	legacyConfigName    = "config/config.yaml"
	DefaultLogLevel     = "debug"
	DefaultDebounceTime = 2 * time.Second
)

// Config holds the application configuration.
type Config struct {
	BinaryPath   string        `mapstructure:"binary_path"`
	DebounceTime time.Duration `mapstructure:"debounce_time"`
	LogLevel     string        `mapstructure:"log_level"`
}

// Global is retained for callers of the original package API. New code should
// use LoadConfigForProject and pass the returned configuration explicitly.
var Global *Config

// AppLogger is used only for compatibility warnings from LoadConfig.
var AppLogger logger.ILogger

// LoadConfig loads the configuration for the current working directory.
// It preserves the original API while allowing the application to use the
// project-aware LoadConfigForProject function.
func LoadConfig() error {
	projectPath, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := LoadConfigForProject(projectPath, "")
	if err != nil {
		return err
	}
	Global = cfg
	return nil
}

// LoadConfigForProject loads an optional project configuration. The preferred
// file is <project>/.gomon.yaml; the legacy <project>/config/config.yaml file
// remains supported for migration. An explicit path bypasses auto-discovery.
func LoadConfigForProject(projectPath, explicitPath string) (*Config, error) {
	projectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("resolve project path: %w", err)
	}

	v := viper.New()
	v.SetDefault("binary_path", "")
	v.SetDefault("debounce_time", int64(DefaultDebounceTime/time.Millisecond))
	v.SetDefault("log_level", DefaultLogLevel)

	configPath, legacy, err := selectConfigPath(projectPath, explicitPath)
	if err != nil {
		return nil, err
	}
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config %q: %w", configPath, err)
		}
		if legacy && AppLogger != nil {
			AppLogger.Warn("Legacy config path is deprecated; use .gomon.yaml", zap.String("path", configPath))
		}
	}

	cfg, err := configFromViper(v)
	if err != nil {
		return nil, fmt.Errorf("process config: %w", err)
	}
	return cfg, nil
}

func selectConfigPath(projectPath, explicitPath string) (string, bool, error) {
	if explicitPath != "" {
		path, err := filepath.Abs(explicitPath)
		if err != nil {
			return "", false, fmt.Errorf("resolve config path: %w", err)
		}
		if _, err := os.Stat(path); err != nil {
			return "", false, fmt.Errorf("stat config %q: %w", path, err)
		}
		return path, false, nil
	}

	preferred := filepath.Join(projectPath, defaultConfigName)
	if _, err := os.Stat(preferred); err == nil {
		return preferred, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, fmt.Errorf("stat config %q: %w", preferred, err)
	}

	legacy := filepath.Join(projectPath, legacyConfigName)
	if _, err := os.Stat(legacy); err == nil {
		return legacy, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, fmt.Errorf("stat config %q: %w", legacy, err)
	}

	return "", false, nil
}

func configFromViper(v *viper.Viper) (*Config, error) {
	debounce, err := parseDebounce(v.Get("debounce_time"))
	if err != nil {
		return nil, err
	}
	if debounce <= 0 {
		return nil, fmt.Errorf("debounce_time must be greater than zero")
	}

	level := strings.ToLower(strings.TrimSpace(v.GetString("log_level")))
	if !validLogLevel(level) {
		return nil, fmt.Errorf("unsupported log_level %q", level)
	}

	return &Config{
		BinaryPath:   strings.TrimSpace(v.GetString("binary_path")),
		DebounceTime: debounce,
		LogLevel:     level,
	}, nil
}

func parseDebounce(value any) (time.Duration, error) {
	switch typed := value.(type) {
	case time.Duration:
		return typed, nil
	case string:
		value := strings.TrimSpace(typed)
		if value == "" {
			return 0, fmt.Errorf("debounce_time cannot be empty")
		}
		if duration, err := time.ParseDuration(value); err == nil {
			return duration, nil
		}
		milliseconds, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid debounce_time %q", value)
		}
		return time.Duration(milliseconds) * time.Millisecond, nil
	case json.Number:
		milliseconds, err := typed.Int64()
		if err != nil {
			return 0, fmt.Errorf("invalid debounce_time %q", typed)
		}
		return time.Duration(milliseconds) * time.Millisecond, nil
	case int:
		return time.Duration(typed) * time.Millisecond, nil
	case int8:
		return time.Duration(typed) * time.Millisecond, nil
	case int16:
		return time.Duration(typed) * time.Millisecond, nil
	case int32:
		return time.Duration(typed) * time.Millisecond, nil
	case int64:
		return time.Duration(typed) * time.Millisecond, nil
	case uint:
		return time.Duration(typed) * time.Millisecond, nil
	case uint8:
		return time.Duration(typed) * time.Millisecond, nil
	case uint16:
		return time.Duration(typed) * time.Millisecond, nil
	case uint32:
		return time.Duration(typed) * time.Millisecond, nil
	case uint64:
		if typed > uint64(^uint64(0)>>1)/uint64(time.Millisecond) {
			return 0, fmt.Errorf("debounce_time is too large")
		}
		return time.Duration(typed) * time.Millisecond, nil
	case float32:
		return parseDebounce(strconv.FormatFloat(float64(typed), 'f', -1, 32))
	case float64:
		if typed != float64(int64(typed)) {
			return 0, fmt.Errorf("debounce_time must be an integer number of milliseconds")
		}
		return parseDebounce(int64(typed))
	default:
		return 0, fmt.Errorf("invalid debounce_time type %T", value)
	}
}

func validLogLevel(level string) bool {
	switch level {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}

// ValidLogLevel reports whether level is supported by the logger.
func ValidLogLevel(level string) bool {
	return validLogLevel(strings.ToLower(strings.TrimSpace(level)))
}

// DurationFromMilliseconds converts the CLI's millisecond representation to
// the duration used internally by the watcher.
func DurationFromMilliseconds(milliseconds int) time.Duration {
	return time.Duration(milliseconds) * time.Millisecond
}
