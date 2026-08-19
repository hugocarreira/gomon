package config

import (
	"encoding/json"
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
		if err := os.Chdir(originalWD); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
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

func TestSetupOverrides(t *testing.T) {
	cfg := &Config{
		BinaryPath:   "/tmp/main",
		DebounceTime: 2 * time.Second,
		LogLevel:     "warn",
	}

	cfg.SetupOverrides("/tmp/app", " DEBUG ", 3000)

	if cfg.BinaryPath != "/tmp/app" {
		t.Fatalf("expected binary path /tmp/app, got %s", cfg.BinaryPath)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected log level debug, got %s", cfg.LogLevel)
	}
	if cfg.DebounceTime != 3*time.Second {
		t.Fatalf("expected debounce time 3s, got %v", cfg.DebounceTime)
	}

	cfg.SetupOverrides("", "", 0)
	if cfg.BinaryPath != "/tmp/app" || cfg.LogLevel != "debug" || cfg.DebounceTime != 3*time.Second {
		t.Fatalf("empty overrides changed config: %+v", cfg)
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

func TestParseDebounceSupportsLegacyAndDurationValues(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  time.Duration
	}{
		{name: "duration", value: 750 * time.Millisecond, want: 750 * time.Millisecond},
		{name: "duration string", value: "750ms", want: 750 * time.Millisecond},
		{name: "legacy string", value: "750", want: 750 * time.Millisecond},
		{name: "json number", value: json.Number("750"), want: 750 * time.Millisecond},
		{name: "int", value: int(750), want: 750 * time.Millisecond},
		{name: "int8", value: int8(7), want: 7 * time.Millisecond},
		{name: "int16", value: int16(7), want: 7 * time.Millisecond},
		{name: "int32", value: int32(7), want: 7 * time.Millisecond},
		{name: "int64", value: int64(7), want: 7 * time.Millisecond},
		{name: "uint", value: uint(7), want: 7 * time.Millisecond},
		{name: "uint8", value: uint8(7), want: 7 * time.Millisecond},
		{name: "uint16", value: uint16(7), want: 7 * time.Millisecond},
		{name: "uint32", value: uint32(7), want: 7 * time.Millisecond},
		{name: "uint64", value: uint64(7), want: 7 * time.Millisecond},
		{name: "float32", value: float32(7), want: 7 * time.Millisecond},
		{name: "float64", value: float64(7), want: 7 * time.Millisecond},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseDebounce(test.value)
			if err != nil {
				t.Fatalf("parseDebounce returned error: %v", err)
			}
			if got != test.want {
				t.Fatalf("parseDebounce(%v) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}

func TestParseDebounceRejectsInvalidValues(t *testing.T) {
	for _, value := range []any{"", "not-a-duration", float64(1.5), struct{}{}} {
		if _, err := parseDebounce(value); err == nil {
			t.Fatalf("parseDebounce(%v) unexpectedly succeeded", value)
		}
	}
	if _, err := parseDebounce(uint64(^uint64(0))); err == nil {
		t.Fatal("expected oversized debounce to fail")
	}
}

func TestConfigValidationHelpers(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error", " INFO "} {
		if !ValidLogLevel(level) {
			t.Fatalf("expected log level %q to be valid", level)
		}
	}
	if ValidLogLevel("trace") {
		t.Fatal("did not expect trace to be valid")
	}
	if got := DurationFromMilliseconds(25); got != 25*time.Millisecond {
		t.Fatalf("unexpected duration: %v", got)
	}
}

func TestLoadConfigForProjectRejectsInvalidSettings(t *testing.T) {
	project := t.TempDir()
	path := filepath.Join(project, ".gomon.yaml")
	for _, contents := range []string{"debounce_time: 0\n", "log_level: trace\n"} {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		if _, err := LoadConfigForProject(project, ""); err == nil {
			t.Fatalf("expected invalid config %q to fail", contents)
		}
	}
}
