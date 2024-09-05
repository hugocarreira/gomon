package watcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/hugocarreira/gomon/builder"
	"github.com/hugocarreira/gomon/config"
	"github.com/hugocarreira/gomon/events"
	"github.com/hugocarreira/gomon/files"
	"github.com/hugocarreira/gomon/log"
	"go.uber.org/zap"
)

type Watcher struct {
	projectPath  string
	binaryPath   string
	log          log.ILogger
	watcher      *fsnotify.Watcher
	eventHandler events.IEventsHandler
	filesHandler *files.Files
}

func NewWatcher(projectPath string, config *config.Config, logger log.ILogger) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Fatal("Erro ao criar watcher", zap.Error(err))
		return nil, err
	}

	builder := builder.NewBuilder(projectPath, config.BinaryPath)
	filesHandler := files.NewFilesHandler(w)
	eventHandler := events.NewEventsHandler(logger, builder, filesHandler, config)

	return &Watcher{
		projectPath:  projectPath,
		binaryPath:   config.BinaryPath,
		log:          logger,
		watcher:      w,
		eventHandler: eventHandler,
		filesHandler: filesHandler,
	}, nil
}

func (w *Watcher) Start() error {
	err := w.filesHandler.AddDir(w.projectPath)
	if err != nil {
		w.log.Fatal("Erro ao adicionar diretório ao watcher", zap.Error(err))
		return err
	}

	w.log.Info("Watcher iniciado para o diretório", zap.String("path", w.projectPath))

	w.eventHandler.StartDebounce()

	for {
		select {
		case event := <-w.watcher.Events:
			w.eventHandler.HandleEvent(event)
		case err := <-w.watcher.Errors:
			w.eventHandler.OnError(err)
		}
	}
}
