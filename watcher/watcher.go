package watcher

import (
	"context"
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/hugocarreira/gomon/builder"
	"github.com/hugocarreira/gomon/config"
	"github.com/hugocarreira/gomon/events"
	"github.com/hugocarreira/gomon/files"
	"github.com/hugocarreira/gomon/logger"
	"go.uber.org/zap"
)

// Watcher monitors file changes in the project path and triggers rebuilds.
type Watcher struct {
	projectPath  string
	builder      builder.IBuilder
	log          logger.ILogger
	watcher      *fsnotify.Watcher
	eventHandler events.IEventsHandler
	filesHandler *files.Files
	cleanupOnce  sync.Once
}

// NewWatcher creates a new Watcher instance.
func NewWatcher(projectPath string, config *config.Config, log logger.ILogger) (*Watcher, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	b := builder.NewBuilder(projectPath, config.BinaryPath)
	filesHandler := files.NewFilesHandler(w)
	eventHandler := events.NewEventsHandler(log, b, filesHandler, config)

	return &Watcher{
		projectPath:  projectPath,
		builder:      b,
		log:          log,
		watcher:      w,
		eventHandler: eventHandler,
		filesHandler: filesHandler,
	}, nil
}

// Start begins the file watching loop.
func (w *Watcher) Start(ctx context.Context) error {
	err := w.filesHandler.AddDir(w.projectPath)
	if err != nil {
		w.log.Error("Failed to add directory to watcher", zap.Error(err))
		w.cleanup()
		return err
	}

	w.log.Info("Watcher started for directory", zap.String("path", w.projectPath))

	w.eventHandler.StartDebounce(ctx)

	events := w.watcher.Events
	errorsCh := w.watcher.Errors
	for events != nil || errorsCh != nil {
		select {
		case <-ctx.Done():
			w.log.Info("Stopping watcher...")
			w.cleanup()
			return nil
		case event, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			w.eventHandler.HandleEvent(event)
		case err, ok := <-errorsCh:
			if !ok {
				errorsCh = nil
				continue
			}
			w.eventHandler.OnError(err)
		}
	}
	w.cleanup()
	return nil
}

// cleanup releases all resources used by the watcher.
func (w *Watcher) cleanup() {
	w.cleanupOnce.Do(func() {
		w.log.Info("Cleaning up resources...")

		w.eventHandler.Stop()
		if closer, ok := w.builder.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				w.log.Error("Failed to stop application", zap.Error(err))
			}
		}
		if w.watcher != nil {
			if err := w.watcher.Close(); err != nil {
				w.log.Error("Failed to close watcher", zap.Error(err))
			}
		}
	})
}
