package main

import (
	"os"

	"github.com/hugocarreira/gomon/config"
	"github.com/hugocarreira/gomon/files"
	"github.com/hugocarreira/gomon/log"
	"github.com/hugocarreira/gomon/watcher"

	"go.uber.org/zap"
)

func main() {
	log := log.NewLogger()

	err := config.LoadConfig()
	if err != nil {
		log.Fatal("Erro ao carregar configurações", zap.Error(err))
	}

	projectPath, err := files.DefineProjectPath(os.Args)
	if err != nil {
		log.Fatal("Erro ao definir o diretório do projeto", zap.Error(err))
	} else {
		log.Debug("Diretório do projeto definido", zap.String("path", projectPath))
	}

	err = files.VerifyProjectPath(projectPath)
	if err != nil {
		log.Fatal("Diretório do projeto não encontrado", zap.String("projectPath", projectPath))
	}

	w, err := watcher.NewWatcher(projectPath, config.Global, log)
	if err != nil {
		log.Fatal("Erro ao criar watcher", zap.Error(err))
	}

	err = w.Start()
	if err != nil {
		log.Fatal("Erro ao iniciar watcher", zap.Error(err))
	}
}
