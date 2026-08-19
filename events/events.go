package events

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hugocarreira/gomon/builder"
	"github.com/hugocarreira/gomon/config"
	"github.com/hugocarreira/gomon/files"
	"github.com/hugocarreira/gomon/logger"
	"go.uber.org/zap"
)

// IEventsHandler handles file system events.
type IEventsHandler interface {
	StartDebounce(ctx context.Context)
	HandleEvent(event fsnotify.Event)
	ProcessEvent(event fsnotify.Event)
	OnCreate(event fsnotify.Event)
	OnWrite(event fsnotify.Event)
	OnRemove(event fsnotify.Event)
	OnRename(event fsnotify.Event)
	OnChmod(event fsnotify.Event)
	OnError(err error)
	Stop()
}

// EventsHandler processes file system events and triggers debounced rebuilds.
type EventsHandler struct {
	log           logger.ILogger
	builder       builder.IBuilder
	filesHandler  *files.Files
	debounceTime  time.Duration
	rebuild       chan struct{}
	lastEventTime map[string]time.Time
	initialRun    bool
	started       bool
	stopped       bool
	mu            sync.Mutex
	stopChan      chan struct{}
	stopOnce      sync.Once
	done          chan struct{}
}

// NewEventsHandler creates a new EventsHandler instance.
func NewEventsHandler(log logger.ILogger, build builder.IBuilder, filesHandler *files.Files, cfg *config.Config) *EventsHandler {
	debounceTime := config.DefaultDebounceTime
	if cfg != nil && cfg.DebounceTime > 0 {
		debounceTime = cfg.DebounceTime
	}
	return &EventsHandler{
		log:           log,
		builder:       build,
		filesHandler:  filesHandler,
		debounceTime:  debounceTime,
		rebuild:       make(chan struct{}, 1),
		lastEventTime: make(map[string]time.Time),
		stopChan:      make(chan struct{}),
		done:          make(chan struct{}),
	}
}

// HandleEvent discovers new directories and schedules one trailing-edge
// rebuild for relevant Go build input changes.
func (e *EventsHandler) HandleEvent(event fsnotify.Event) {
	if e.isStopped() {
		return
	}

	isDir, statErr := e.filesHandler.IsDir(event.Name)
	ignoredDirectory := isDir && files.ShouldIgnoreDir(filepath.Base(event.Name))
	if event.Op.Has(fsnotify.Create) && isDir && !ignoredDirectory {
		if err := e.filesHandler.HandleFiles(event.Name); err != nil {
			e.log.Error("Failed to add directory to watcher", zap.Error(err))
		}
	}
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		e.log.Error("Failed to inspect filesystem event", zap.Error(statErr))
	}

	watchable := files.ShouldWatchFile(event.Name)
	directoryChanged := e.filesHandler.WasWatchedDir(event.Name) && event.Op.Has(fsnotify.Remove|fsnotify.Rename)
	inputChanged := watchable && event.Op.Has(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename)
	newDirectory := isDir && event.Op.Has(fsnotify.Create) && !ignoredDirectory
	if !inputChanged && !directoryChanged && !newDirectory {
		return
	}

	e.mu.Lock()
	e.lastEventTime[event.Name] = time.Now()
	e.mu.Unlock()
	e.log.Building("File changed, rebuilding...")
	select {
	case e.rebuild <- struct{}{}:
	default:
	}
}

// StartDebounce performs the initial build and starts a trailing-edge timer.
func (e *EventsHandler) StartDebounce(ctx context.Context) {
	e.mu.Lock()
	if e.started || e.stopped {
		e.mu.Unlock()
		return
	}
	e.started = true
	if !e.initialRun {
		e.initialRun = true
		e.mu.Unlock()
		e.restart()
	} else {
		e.mu.Unlock()
	}

	go func() {
		defer close(e.done)
		var timer *time.Timer
		var timerC <-chan time.Time
		defer func() {
			if timer != nil {
				timer.Stop()
			}
		}()

		for {
			select {
			case <-ctx.Done():
				e.log.Info("Debounce goroutine stopped")
				return
			case <-e.stopChan:
				e.log.Info("Debounce goroutine stopped")
				return
			case <-e.rebuild:
				if timer == nil {
					timer = time.NewTimer(e.debounceTime)
				} else {
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(e.debounceTime)
				}
				timerC = timer.C
			case <-timerC:
				timerC = nil
				e.restart()
			}
		}
	}()
}

// ProcessEvent is retained as a direct-dispatch API for callers that already
// have a fully formed event. The watcher itself uses HandleEvent so events are
// coalesced before rebuilding.
func (e *EventsHandler) ProcessEvent(event fsnotify.Event) {
	switch {
	case event.Op.Has(fsnotify.Create):
		e.OnCreate(event)
	case event.Op.Has(fsnotify.Write):
		e.OnWrite(event)
	case event.Op.Has(fsnotify.Remove):
		e.OnRemove(event)
	case event.Op.Has(fsnotify.Rename):
		e.OnRename(event)
	case event.Op.Has(fsnotify.Chmod):
		e.OnChmod(event)
	}
}

// OnCreate handles file or directory creation events.
func (e *EventsHandler) OnCreate(event fsnotify.Event) {
	e.log.Info("File or directory created", zap.String("file", event.Name))
	e.addDirectory(event.Name)
	e.restart()
}

// OnWrite handles file modification events.
func (e *EventsHandler) OnWrite(event fsnotify.Event) {
	e.log.Info("File or directory modified", zap.String("file", event.Name))
	e.addDirectory(event.Name)
	e.restart()
}

// OnRemove handles file removal events.
func (e *EventsHandler) OnRemove(event fsnotify.Event) {
	e.log.Info("File or directory removed", zap.String("file", event.Name))
	e.addDirectory(event.Name)
	e.restart()
}

// OnRename handles file rename events.
func (e *EventsHandler) OnRename(event fsnotify.Event) {
	e.log.Info("File renamed", zap.String("file", event.Name))
	e.addDirectory(event.Name)
	e.restart()
}

// OnChmod handles file permission change events.
func (e *EventsHandler) OnChmod(event fsnotify.Event) {
	e.log.Info("File permissions changed", zap.String("file", event.Name))
}

// OnError handles watcher errors.
func (e *EventsHandler) OnError(err error) {
	if err != nil {
		e.log.Error("Watcher error", zap.Error(err))
	}
}

// Stop stops the event handler goroutine. It is safe to call repeatedly.
func (e *EventsHandler) Stop() {
	e.stopOnce.Do(func() { close(e.stopChan) })
	e.mu.Lock()
	started := e.started
	e.stopped = true
	e.mu.Unlock()
	if started {
		<-e.done
	}
}

func (e *EventsHandler) addDirectory(path string) {
	if err := e.filesHandler.HandleFiles(path); err != nil {
		e.log.Error("Failed to add directory to watcher", zap.Error(err))
	}
}

func (e *EventsHandler) restart() {
	e.log.Building("Rebuilding...")
	if err := e.builder.RestartBinary(); err != nil {
		e.log.BuildError("Failed to restart: " + err.Error())
		return
	}
	e.log.BuildSuccess("Build successful")
}

func (e *EventsHandler) isStopped() bool {
	e.mu.Lock()
	stopped := e.stopped
	e.mu.Unlock()
	return stopped
}
