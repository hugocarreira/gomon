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

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if Global == nil {
		t.Fatalf("expected global config to be initialized")
	}

	if Global.BinaryPath != "/tmp/main" {
		t.Fatalf("expected binary path /tmp/main, got %s", Global.BinaryPath)
	}

	if Global.DebounceTime != time.Duration(2000) {
		t.Fatalf("expected debounce time 2000, got %v", Global.DebounceTime)
	}
}
