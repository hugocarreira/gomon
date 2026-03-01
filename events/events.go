package events

import (
	"context"
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

// EventsHandler processes file system events and triggers rebuilds.
type EventsHandler struct {
	log           logger.ILogger
	builder       builder.IBuilder
	filesHandler  *files.Files
	debounceTime  time.Duration
	eventQueue    chan fsnotify.Event
	lastEventTime map[string]time.Time
	minInterval   time.Duration
	initialRun    bool
	mu            sync.RWMutex
	stopChan      chan struct{}
}

// NewEventsHandler creates a new EventsHandler instance.
func NewEventsHandler(log logger.ILogger, builder builder.IBuilder, filesHandler *files.Files, config *config.Config) *EventsHandler {
	return &EventsHandler{
		log:           log,
		builder:       builder,
		filesHandler:  filesHandler,
		debounceTime:  config.DebounceTime,
		eventQueue:    make(chan fsnotify.Event, 1),
		lastEventTime: make(map[string]time.Time),
		minInterval:   100 * time.Millisecond,
		initialRun:    false,
		stopChan:      make(chan struct{}),
	}
}

// HandleEvent processes a file system event.
func (e *EventsHandler) HandleEvent(event fsnotify.Event) {
	if !files.ShouldWatchFile(event.Name) {
		return
	}

	now := time.Now()

	e.mu.RLock()
	lastTime, exists := e.lastEventTime[event.Name]
	e.mu.RUnlock()

	if exists {
		e.mu.RLock()
		interval := now.Sub(lastTime)
		e.mu.RUnlock()
		if interval < e.minInterval {
			return
		}
	}

	e.mu.Lock()
	e.lastEventTime[event.Name] = now
	e.mu.Unlock()

	e.log.Building("File changed, rebuilding...")

	select {
	case e.eventQueue <- event:
	default:
		e.eventQueue <- event
	}
}

// StartDebounce starts the debounce goroutine for processing events.
func (e *EventsHandler) StartDebounce(ctx context.Context) {
	if !e.initialRun {
		err := e.builder.RestartBinary()
		if err != nil {
			e.log.BuildError("Failed to start: " + err.Error())
		} else {
			e.log.Running("Application started")
		}
		e.initialRun = true
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				e.log.Info("Debounce goroutine stopped")
				return
			case <-e.stopChan:
				e.log.Info("Debounce goroutine stopped")
				return
			default:
				time.Sleep(e.debounceTime * time.Millisecond)

				select {
				case event := <-e.eventQueue:
					e.ProcessEvent(event)
				default:
				}
			}
		}
	}()
}

// ProcessEvent dispatches the event to the appropriate handler.
func (e *EventsHandler) ProcessEvent(event fsnotify.Event) {
	switch event.Op {
	case fsnotify.Create:
		e.OnCreate(event)
	case fsnotify.Write:
		e.OnWrite(event)
	case fsnotify.Remove:
		e.OnRemove(event)
	case fsnotify.Rename:
		e.OnRename(event)
	case fsnotify.Chmod:
		e.OnChmod(event)
	}
}

// OnCreate handles file creation events.
func (e *EventsHandler) OnCreate(event fsnotify.Event) {
	e.log.Info("File or directory created", zap.String("file", event.Name))

	err := e.filesHandler.HandleFiles(event.Name)
	if err != nil {
		e.log.Error("Failed to add directory to watcher", zap.Error(err))
	}

	e.log.Building("Rebuilding...")
	err = e.builder.RestartBinary()
	if err != nil {
		e.log.BuildError("Failed to restart: " + err.Error())
	} else {
		e.log.BuildSuccess("Build successful")
	}
}

// OnWrite handles file modification events.
func (e *EventsHandler) OnWrite(event fsnotify.Event) {
	e.log.Info("File or directory modified", zap.String("file", event.Name))

	err := e.filesHandler.HandleFiles(event.Name)
	if err != nil {
		e.log.Error("Failed to add directory to watcher", zap.Error(err))
	}

	e.log.Building("Rebuilding...")
	err = e.builder.RestartBinary()
	if err != nil {
		e.log.BuildError("Failed to restart: " + err.Error())
	} else {
		e.log.BuildSuccess("Build successful")
	}
}

// OnRemove handles file removal events.
func (e *EventsHandler) OnRemove(event fsnotify.Event) {
	e.log.Info("File or directory removed", zap.String("file", event.Name))

	err := e.filesHandler.HandleFiles(event.Name)
	if err != nil {
		e.log.Error("Failed to handle removed file", zap.Error(err))
	}

	e.log.Building("Rebuilding...")
	err = e.builder.RestartBinary()
	if err != nil {
		e.log.BuildError("Failed to restart: " + err.Error())
	} else {
		e.log.BuildSuccess("Build successful")
	}
}

// OnRename handles file rename events.
func (e *EventsHandler) OnRename(event fsnotify.Event) {
	e.log.Info("File renamed", zap.String("file", event.Name))
	err := e.filesHandler.HandleFiles(event.Name)
	if err != nil {
		e.log.Error("Failed to handle renamed file", zap.Error(err))
	}
}

// OnChmod handles file permission change events.
func (e *EventsHandler) OnChmod(event fsnotify.Event) {
	e.log.Info("File permissions changed", zap.String("file", event.Name))
}

// OnError handles watcher errors.
func (e *EventsHandler) OnError(err error) {
	e.log.Error("Watcher error", zap.Error(err))
}

// Stop stops the event handler goroutine.
func (e *EventsHandler) Stop() {
	close(e.stopChan)
}
