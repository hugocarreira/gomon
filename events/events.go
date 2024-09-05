package events

import (
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hugocarreira/gomon/builder"
	"github.com/hugocarreira/gomon/config"
	"github.com/hugocarreira/gomon/files"
	"github.com/hugocarreira/gomon/log"
	"go.uber.org/zap"
)

type IEventsHandler interface {
	StartDebounce()
	HandleEvent(event fsnotify.Event)
	ProcessEvent(event fsnotify.Event)
	OnCreate(event fsnotify.Event)
	OnWrite(event fsnotify.Event)
	OnRemove(event fsnotify.Event)
	OnError(err error)
}

type EventsHandler struct {
	log           log.ILogger
	builder       builder.IBuilder
	filesHandler  *files.Files
	debounceTime  time.Duration
	eventQueue    chan fsnotify.Event
	lastEventTime map[string]time.Time
	minInterval   time.Duration
	initialRun    bool
}

func NewEventsHandler(logger log.ILogger, builder builder.IBuilder, filesHandler *files.Files, config *config.Config) *EventsHandler {
	return &EventsHandler{
		log:           logger,
		builder:       builder,
		filesHandler:  filesHandler,
		debounceTime:  config.DebounceTime,
		eventQueue:    make(chan fsnotify.Event, 1),
		lastEventTime: make(map[string]time.Time),
		minInterval:   100 * time.Millisecond,
		initialRun:    false,
	}
}

func (e *EventsHandler) HandleEvent(event fsnotify.Event) {
	now := time.Now()

	if lastTime, exists := e.lastEventTime[event.Name]; exists {
		if now.Sub(lastTime) < e.minInterval {
			return
		}
	}

	e.lastEventTime[event.Name] = now

	e.log.Info("Executando o Builder...")

	select {
	case e.eventQueue <- event:
	default:
		e.eventQueue <- event
	}
}

func (e *EventsHandler) StartDebounce() {
	if !e.initialRun {
		e.builder.RestartBinary()
		e.initialRun = true
	}

	go func() {
		for {
			time.Sleep(e.debounceTime * time.Millisecond)

			select {
			case event := <-e.eventQueue:
				e.ProcessEvent(event)
			default:
			}
		}
	}()
}

func (e *EventsHandler) ProcessEvent(event fsnotify.Event) {
	switch event.Op {
	case fsnotify.Create:
		e.OnCreate(event)
	case fsnotify.Write:
		e.OnWrite(event)
	case fsnotify.Remove:
		e.OnRemove(event)
	}
}

func (e *EventsHandler) OnCreate(event fsnotify.Event) {
	e.log.Info("Arquivo ou diretório criado", zap.String("file", event.Name))

	err := e.filesHandler.HandleFiles(event.Name)
	if err != nil {
		e.log.Error("Erro ao adicionar diretório ao watcher", zap.Error(err))
	}

	err = e.builder.RestartBinary()
	if err != nil {
		e.log.Error("Erro ao reiniciar o binário", zap.Error(err))
	}
}

func (e *EventsHandler) OnWrite(event fsnotify.Event) {
	e.log.Info("Arquivo ou diretório modificado", zap.String("file", event.Name))

	err := e.filesHandler.HandleFiles(event.Name)
	if err != nil {
		e.log.Error("Erro ao adicionar diretório ao watcher", zap.Error(err))
	}

	err = e.builder.RestartBinary()
	if err != nil {
		e.log.Error("Erro ao reiniciar o binário", zap.Error(err))
	}
}

func (e *EventsHandler) OnRemove(event fsnotify.Event) {
	e.log.Info("Arquivo ou diretório removido", zap.String("file", event.Name))

	err := e.filesHandler.HandleFiles(event.Name)
	if err != nil {
		e.log.Error("Erro ao adicionar diretório ao watcher", zap.Error(err))
	}

	err = e.builder.RestartBinary()
	if err != nil {
		e.log.Error("Erro ao reiniciar o binário", zap.Error(err))
	}
}

func (e *EventsHandler) OnError(err error) {
	e.log.Error("Erro no watcher", zap.Error(err))
}
