package config

import (
	"os"
	"testing"
	"time"

	"github.com/hugocarreira/gomon/logger"
)

func TestLoadConfigPopulatesGlobal(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to determine working directory: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(originalWD)
	})

	if err := os.Chdir(".."); err != nil {
		t.Fatalf("failed to move to module root: %v", err)
	}

	AppLogger = logger.NewLogger()
	t.Cleanup(func() {
		AppLogger = nil
		Global = nil
	})

	if err := LoadConfig(AppLogger); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if Global == nil {
		t.Fatalf("expected global config to be initialized")
	}

	if Global.BinaryPath != "/tmp/main" {
		t.Fatalf("expected binary path /tmp/main, got %s", Global.BinaryPath)
	}

	if Global.LogLevel != "debug" {
		t.Fatalf("expected log level debug, got %s", Global.LogLevel)
	}

	if Global.DebounceTime != time.Duration(2000) {
		t.Fatalf("expected debounce time 2000, got %v", Global.DebounceTime)
	}
}

func TestSetupOverrides(t *testing.T) {
	cfg := &Config{
		BinaryPath:   "/tmp/main",
		DebounceTime: time.Duration(2000),
		LogLevel:     "warn",
	}

	cfg.SetupOverrides("/my/binary", "debug", 3000)

	if cfg.BinaryPath != "/my/binary" {
		t.Fatalf("expected binary path /my/binary, got %s", cfg.BinaryPath)
	}

	if cfg.DebounceTime != time.Duration(3000)*time.Millisecond {
		t.Fatalf("expected debounce time 3000ms, got %v", cfg.DebounceTime)
	}

	if cfg.LogLevel != "debug" {
		t.Fatalf("expected log level debug, got %s", cfg.LogLevel)
	}

	// Test that empty overrides do not change values
	cfg.SetupOverrides("", "", 0)

	if cfg.BinaryPath != "/my/binary" {
		t.Fatalf("expected binary path to remain /my/binary, got %s", cfg.BinaryPath)
	}

	if cfg.LogLevel != "debug" {
		t.Fatalf("expected log level to remain debug, got %s", cfg.LogLevel)
	}

	if cfg.DebounceTime != time.Duration(3000)*time.Millisecond {
		t.Fatalf("expected debounce time to remain 3000ms, got %v", cfg.DebounceTime)
	}

}
