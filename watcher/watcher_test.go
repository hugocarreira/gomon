package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hugocarreira/gomon/config"
	"github.com/hugocarreira/gomon/files"
	"github.com/hugocarreira/gomon/logger"
)

type stubEventsHandler struct {
	startCalled bool
	stopCalled  bool
	handled     []fsnotify.Event
}

func (s *stubEventsHandler) StartDebounce(ctx context.Context) {
	s.startCalled = true
}

func (s *stubEventsHandler) HandleEvent(event fsnotify.Event) {
	s.handled = append(s.handled, event)
}

func (s *stubEventsHandler) ProcessEvent(event fsnotify.Event) {}
func (s *stubEventsHandler) OnCreate(event fsnotify.Event)     {}
func (s *stubEventsHandler) OnWrite(event fsnotify.Event)      {}
func (s *stubEventsHandler) OnRemove(event fsnotify.Event)     {}
func (s *stubEventsHandler) OnRename(event fsnotify.Event)     {}
func (s *stubEventsHandler) OnChmod(event fsnotify.Event)      {}
func (s *stubEventsHandler) OnError(err error)                 {}

func (s *stubEventsHandler) Stop() {
	s.stopCalled = true
}

func TestWatcherStartStopsOnContextCancel(t *testing.T) {
	dir := t.TempDir()

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("failed to create fsnotify watcher: %v", err)
	}

	filesHandler := files.NewFilesHandler(fsWatcher)
	eventHandler := &stubEventsHandler{}
	w := &Watcher{
		projectPath:  dir,
		log:          logger.NewLogger(),
		watcher:      fsWatcher,
		eventHandler: eventHandler,
		filesHandler: filesHandler,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- w.Start(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("watcher.Start returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("watcher.Start did not exit after context cancellation")
	}

	if !eventHandler.startCalled {
		t.Fatal("expected StartDebounce to be called")
	}
	if !eventHandler.stopCalled {
		t.Fatal("expected Stop to be called during cleanup")
	}
}

func TestNewWatcherRejectsNilConfig(t *testing.T) {
	if _, err := NewWatcher(t.TempDir(), nil, logger.NewLogger()); err == nil {
		t.Fatal("expected nil config to return an error")
	}
}

func TestNewWatcherCreatesOwnedResources(t *testing.T) {
	w, err := NewWatcher(t.TempDir(), &config.Config{DebounceTime: time.Second}, logger.NewLogger())
	if err != nil {
		t.Fatalf("NewWatcher returned error: %v", err)
	}
	w.cleanup()
	w.cleanup()
}

func TestWatcherStartForwardsFilesystemEvents(t *testing.T) {
	dir := t.TempDir()
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("failed to create fsnotify watcher: %v", err)
	}
	filesHandler := files.NewFilesHandler(fsWatcher)
	eventHandler := &stubEventsHandler{}
	w := &Watcher{
		projectPath:  dir,
		log:          logger.NewLogger(),
		watcher:      fsWatcher,
		eventHandler: eventHandler,
		filesHandler: filesHandler,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Start(ctx) }()
	time.Sleep(30 * time.Millisecond)
	path := filepath.Join(dir, "changed.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write event fixture: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("watcher.Start returned error: %v", err)
	}
	if len(eventHandler.handled) == 0 {
		t.Fatal("expected watcher event to reach the event handler")
	}
}
