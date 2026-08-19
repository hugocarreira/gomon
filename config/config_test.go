package config

import (
	"os"
	"path/filepath"
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

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if Global == nil {
		t.Fatalf("expected global config to be initialized")
	}

	if Global.BinaryPath != "" {
		t.Fatalf("expected empty binary path, got %s", Global.BinaryPath)
	}

	if Global.DebounceTime != 2*time.Second {
		t.Fatalf("expected debounce time 2s, got %v", Global.DebounceTime)
	}
	if Global.LogLevel != DefaultLogLevel {
		t.Fatalf("expected log level %q, got %q", DefaultLogLevel, Global.LogLevel)
	}
}

func TestLoadConfigForProjectDefaultsWithoutFile(t *testing.T) {
	cfg, err := LoadConfigForProject(t.TempDir(), "")
	if err != nil {
		t.Fatalf("LoadConfigForProject returned error: %v", err)
	}

	if cfg.BinaryPath != "" || cfg.DebounceTime != DefaultDebounceTime || cfg.LogLevel != DefaultLogLevel {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadConfigForProjectPreferredConfig(t *testing.T) {
	project := t.TempDir()
	contents := []byte("binary_path: build/app\ndebounce_time: 750ms\nlog_level: info\n")
	if err := os.WriteFile(filepath.Join(project, ".gomon.yaml"), contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfigForProject(project, "")
	if err != nil {
		t.Fatalf("LoadConfigForProject returned error: %v", err)
	}
	if cfg.BinaryPath != "build/app" || cfg.DebounceTime != 750*time.Millisecond || cfg.LogLevel != "info" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadConfigForProjectLegacyNumericDebounce(t *testing.T) {
	project := t.TempDir()
	legacyDir := filepath.Join(project, "config")
	if err := os.Mkdir(legacyDir, 0o755); err != nil {
		t.Fatalf("create legacy config directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "config.yaml"), []byte("debounce_time: 2000\n"), 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	cfg, err := LoadConfigForProject(project, "")
	if err != nil {
		t.Fatalf("LoadConfigForProject returned error: %v", err)
	}
	if cfg.DebounceTime != 2*time.Second {
		t.Fatalf("expected legacy numeric debounce to mean 2s, got %v", cfg.DebounceTime)
	}
}

func TestLoadConfigForProjectExplicitMissingFile(t *testing.T) {
	_, err := LoadConfigForProject(t.TempDir(), filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected missing explicit config to return an error")
	}
}
