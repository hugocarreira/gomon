package events

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/hugocarreira/gomon/config"
	"github.com/hugocarreira/gomon/files"
	"go.uber.org/zap"
)

type fakeBuilder struct {
	restartCalls int
}

func (f *fakeBuilder) BuildProject() error             { return nil }
func (f *fakeBuilder) RunBinary() (*exec.Cmd, error)   { return nil, nil }
func (f *fakeBuilder) RestartBinary() error            { f.restartCalls++; return nil }
func (f *fakeBuilder) KillProcess(cmd *exec.Cmd) error { return nil }
func (f *fakeBuilder) Close() error                    { return nil }

type fakeLogger struct {
	infos        []string
	warnings     []string
	errors       []string
	debugs       []string
	buildings    []string
	runnings     []string
	successes    []string
	failures     []string
	fatalMessage string
}

func (f *fakeLogger) Info(m string, fields ...zap.Field)  { f.infos = append(f.infos, m) }
func (f *fakeLogger) Warn(m string, fields ...zap.Field)  { f.warnings = append(f.warnings, m) }
func (f *fakeLogger) Error(m string, fields ...zap.Field) { f.errors = append(f.errors, m) }
func (f *fakeLogger) Debug(m string, fields ...zap.Field) { f.debugs = append(f.debugs, m) }
func (f *fakeLogger) Fatal(m string, fields ...zap.Field) { f.fatalMessage = m }
func (f *fakeLogger) Building(m string)                   { f.buildings = append(f.buildings, m) }
func (f *fakeLogger) Running(m string)                    { f.runnings = append(f.runnings, m) }
func (f *fakeLogger) BuildSuccess(m string)               { f.successes = append(f.successes, m) }
func (f *fakeLogger) BuildError(m string)                 { f.failures = append(f.failures, m) }

func newTestHandler(t *testing.T) (*EventsHandler, *fakeBuilder, *fakeLogger) {
	t.Helper()

	builder := &fakeBuilder{}
	logger := &fakeLogger{}
	filesHandler := &files.Files{}
	cfg := &config.Config{DebounceTime: 10}

	handler := NewEventsHandler(logger, builder, filesHandler, cfg)
	return handler, builder, logger
}

func TestEventsHandlerOnWriteRestarts(t *testing.T) {
	handler, builder, logger := newTestHandler(t)

	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	if err := os.WriteFile(file, []byte("package main"), 0o644); err != nil {
		t.Fatalf("write helper file: %v", err)
	}

	handler.OnWrite(fsnotify.Event{Name: file, Op: fsnotify.Write})

	if builder.restartCalls != 1 {
		t.Fatalf("expected restart call, got %d", builder.restartCalls)
	}
	if len(logger.successes) != 1 {
		t.Fatalf("expected build success log, got %d", len(logger.successes))
	}
}

func TestEventsHandlerOnRemoveRestarts(t *testing.T) {
	handler, builder, logger := newTestHandler(t)

	handler.OnRemove(fsnotify.Event{Name: "missing.go", Op: fsnotify.Remove})

	if builder.restartCalls != 1 {
		t.Fatalf("expected restart call on remove, got %d", builder.restartCalls)
	}
	if len(logger.successes) != 1 {
		t.Fatalf("expected build success log for remove, got %d", len(logger.successes))
	}
}

func TestEventsHandlerHandleEventIgnoresNonGo(t *testing.T) {
	handler, _, _ := newTestHandler(t)

	handler.HandleEvent(fsnotify.Event{Name: "README.md", Op: fsnotify.Write})

	if _, exists := handler.lastEventTime["README.md"]; exists {
		t.Fatalf("did not expect non-Go file to update lastEventTime")
	}
}

func TestEventsHandlerStartDebounceInitialRun(t *testing.T) {
	handler, builder, _ := newTestHandler(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler.StartDebounce(ctx)
	if builder.restartCalls != 1 {
		t.Fatalf("expected builder restart during initial run, got %d", builder.restartCalls)
	}

	cancel()
	handler.Stop()
}
