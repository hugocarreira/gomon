package watcher

import (
	"context"

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
	binaryPath   string
	log          logger.ILogger
	watcher      *fsnotify.Watcher
	eventHandler events.IEventsHandler
	filesHandler *files.Files
}

// NewWatcher creates a new Watcher instance.
func NewWatcher(projectPath string, config *config.Config, log logger.ILogger) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Failed to create watcher", zap.Error(err))
		return nil, err
	}

	b := builder.NewBuilder(projectPath, config.BinaryPath)
	filesHandler := files.NewFilesHandler(w)
	eventHandler := events.NewEventsHandler(log, b, filesHandler, config)

	return &Watcher{
		projectPath:  projectPath,
		binaryPath:   config.BinaryPath,
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
		w.log.Fatal("Failed to add directory to watcher", zap.Error(err))
		return err
	}

	w.log.Info("Watcher started for directory", zap.String("path", w.projectPath))

	w.eventHandler.StartDebounce(ctx)

	for {
		select {
		case <-ctx.Done():
			w.log.Info("Stopping watcher...")
			w.cleanup()
			return nil
		case event := <-w.watcher.Events:
			w.eventHandler.HandleEvent(event)
		case err := <-w.watcher.Errors:
			w.eventHandler.OnError(err)
		}
	}
}

// cleanup releases all resources used by the watcher.
func (w *Watcher) cleanup() {
	w.log.Info("Cleaning up resources...")

	if closer, ok := w.eventHandler.(interface{ Stop() }); ok {
		closer.Stop()
	}

	if w.watcher != nil {
		w.watcher.Close()
	}
}
