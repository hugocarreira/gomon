package events

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hugocarreira/gomon/config"
	"github.com/hugocarreira/gomon/files"
	"go.uber.org/zap"
)

type fakeBuilder struct {
	mu           sync.Mutex
	restartCalls int
}

func (f *fakeBuilder) BuildProject() error           { return nil }
func (f *fakeBuilder) RunBinary() (*exec.Cmd, error) { return nil, nil }
func (f *fakeBuilder) RestartBinary() error {
	f.mu.Lock()
	f.restartCalls++
	f.mu.Unlock()
	return nil
}
func (f *fakeBuilder) KillProcess(cmd *exec.Cmd) error { return nil }
func (f *fakeBuilder) restarts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.restartCalls
}

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
	cfg := &config.Config{DebounceTime: 10 * time.Millisecond}

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

	if builder.restarts() != 1 {
		t.Fatalf("expected restart call, got %d", builder.restarts())
	}
	if len(logger.successes) != 1 {
		t.Fatalf("expected build success log, got %d", len(logger.successes))
	}
}

func TestEventsHandlerOnRemoveRestarts(t *testing.T) {
	handler, builder, logger := newTestHandler(t)

	handler.OnRemove(fsnotify.Event{Name: "missing.go", Op: fsnotify.Remove})

	if builder.restarts() != 1 {
		t.Fatalf("expected restart call on remove, got %d", builder.restarts())
	}
	if len(logger.successes) != 1 {
		t.Fatalf("expected build success log for remove, got %d", len(logger.successes))
	}
}

func TestEventsHandlerOnRenameRestarts(t *testing.T) {
	handler, builder, logger := newTestHandler(t)

	handler.OnRename(fsnotify.Event{Name: "renamed.go", Op: fsnotify.Rename})

	if builder.restarts() != 1 {
		t.Fatalf("expected restart call on rename, got %d", builder.restarts())
	}
	if len(logger.successes) != 1 {
		t.Fatalf("expected build success log for rename, got %d", len(logger.successes))
	}
}

func TestEventsHandlerProcessEventUsesBitmask(t *testing.T) {
	handler, builder, _ := newTestHandler(t)
	handler.ProcessEvent(fsnotify.Event{Name: "created.go", Op: fsnotify.Create | fsnotify.Write})
	if builder.restarts() != 1 {
		t.Fatalf("expected one restart for combined create/write event, got %d", builder.restarts())
	}
}

func TestEventsHandlerProcessEventDispatchesRemainingOperations(t *testing.T) {
	handler, builder, logger := newTestHandler(t)
	handler.ProcessEvent(fsnotify.Event{Name: "removed.go", Op: fsnotify.Remove})
	handler.ProcessEvent(fsnotify.Event{Name: "renamed.go", Op: fsnotify.Rename})
	handler.ProcessEvent(fsnotify.Event{Name: "changed.go", Op: fsnotify.Chmod})

	if builder.restarts() != 2 {
		t.Fatalf("expected remove and rename to restart, got %d", builder.restarts())
	}
	if len(logger.infos) != 3 {
		t.Fatalf("expected three operation logs, got %d", len(logger.infos))
	}
}

func TestEventsHandlerOnCreateAndErrors(t *testing.T) {
	handler, builder, logger := newTestHandler(t)
	handler.OnCreate(fsnotify.Event{Name: "created.go", Op: fsnotify.Create})
	handler.OnChmod(fsnotify.Event{Name: "created.go", Op: fsnotify.Chmod})
	handler.OnError(errors.New("overflow"))
	handler.OnError(nil)

	if builder.restarts() != 1 {
		t.Fatalf("expected create to restart once, got %d", builder.restarts())
	}
	if len(logger.infos) != 2 || len(logger.errors) != 1 {
		t.Fatalf("unexpected logs: infos=%d errors=%d", len(logger.infos), len(logger.errors))
	}
}

func TestEventsHandlerHandleEventIgnoresNonGo(t *testing.T) {
	handler, _, _ := newTestHandler(t)

	handler.HandleEvent(fsnotify.Event{Name: "README.md", Op: fsnotify.Write})

	if !handler.lastEventAt.IsZero() {
		t.Fatalf("did not expect non-Go file to update debounce state")
	}
}

func TestEventsHandlerHandleEventWatchesGoBuildMetadata(t *testing.T) {
	handler, _, _ := newTestHandler(t)

	path := filepath.Join(t.TempDir(), "go.mod")
	if err := os.WriteFile(path, []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	handler.HandleEvent(fsnotify.Event{Name: path, Op: fsnotify.Write})
	if handler.lastEventAt.IsZero() {
		t.Fatal("expected go.mod to schedule a rebuild")
	}
}

func TestEventsHandlerDebouncesBurst(t *testing.T) {
	handler, builder, _ := newTestHandler(t)
	handler.debounceTime = 30 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	handler.StartDebounce(ctx)

	dir := t.TempDir()
	first := filepath.Join(dir, "first.go")
	second := filepath.Join(dir, "second.go")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}
	handler.HandleEvent(fsnotify.Event{Name: first, Op: fsnotify.Write})
	time.Sleep(10 * time.Millisecond)
	handler.HandleEvent(fsnotify.Event{Name: second, Op: fsnotify.Write})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && builder.restarts() < 2 {
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	handler.Stop()

	if got := builder.restarts(); got != 2 {
		t.Fatalf("expected one initial build and one debounced rebuild, got %d", got)
	}
}

func TestEventsHandlerStopIsIdempotent(t *testing.T) {
	handler, _, _ := newTestHandler(t)
	ctx, cancel := context.WithCancel(context.Background())
	handler.StartDebounce(ctx)
	cancel()
	handler.Stop()
	handler.Stop()
}

func TestEventsHandlerAddsCreatedDirectory(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}
	t.Cleanup(func() {
		if err := watcher.Close(); err != nil {
			t.Errorf("close watcher: %v", err)
		}
	})

	handler, _, _ := newTestHandler(t)
	handler.filesHandler = files.NewFilesHandler(watcher)
	dir := filepath.Join(t.TempDir(), "new-dir")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	handler.HandleEvent(fsnotify.Event{Name: dir, Op: fsnotify.Create})

	for _, watched := range watcher.WatchList() {
		if watched == dir {
			return
		}
	}
	t.Fatalf("expected created directory %q to be watched; got %v", dir, watcher.WatchList())
}

func TestEventsHandlerStartDebounceInitialRun(t *testing.T) {
	handler, builder, _ := newTestHandler(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler.StartDebounce(ctx)
	if builder.restarts() != 1 {
		t.Fatalf("expected builder restart during initial run, got %d", builder.restarts())
	}

	cancel()
	handler.Stop()
}
