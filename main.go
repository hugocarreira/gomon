package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/hugocarreira/gomon/config"
	"github.com/hugocarreira/gomon/log"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

func main() {
	var projectPath string

	binaryPath := "/tmp/main"

	err := config.LoadConfig()
	if err != nil {
		log.Fatal("Erro ao carregar configurações", zap.Error(err))
	}

	log.InitLogger()

	if len(os.Args) < 2 {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal("Erro ao obter o diretório atual", zap.Error(err))
		}
		projectPath = cwd
		log.Warn("Diretório do projeto não informado, usando o diretório atual", zap.String("projectPath", projectPath))
	} else {
		projectPath = os.Args[1]
		log.Warn("Diretório do projeto informado", zap.String("projectPath", projectPath))
	}

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		log.Fatal("Diretório do projeto não encontrado", zap.String("projectPath", projectPath))
		os.Exit(1)
	}

	watcher(projectPath, binaryPath)
}

func watcher(projectPath, binaryPath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Erro ao criar watcher", zap.Error(err))
	}
	defer watcher.Close()

	log.Info("Watcher iniciado para o diretório", zap.String("projectPath", projectPath))

	var currentProcess *exec.Cmd

	currentProcess, err = restartBinary(currentProcess, projectPath, binaryPath)
	if err != nil {
		log.Fatal("Erro ao compilar o projeto", zap.Error(err))
	}

	debounce := time.AfterFunc(time.Millisecond*config.Global.DebounceTime, func() {})
	defer debounce.Stop()

	lastEventTime := make(map[string]time.Time)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {

					now := time.Now()
					if lastEventTime[event.Name].After(now.Add(-1 * time.Second)) {
						continue
					}
					lastEventTime[event.Name] = now

					debounce.Reset(500 * time.Millisecond)
					go func() {
						log.Debug("Alteração detectada no arquivo", zap.String("fileName", event.Name))

						time.Sleep(500 * time.Millisecond)
						currentProcess, err = restartBinary(currentProcess, projectPath, binaryPath)
						if err != nil {
							log.Error("Erro ao compilar o projeto", zap.Error(err))
						}
					}()
				}

			case err := <-watcher.Errors:
				log.Error("Erro no watcher", zap.Error(err))
			}
		}
	}()

	err = watcher.Add(projectPath)
	if err != nil {
		log.Fatal("Erro ao adicionar diretório ao watcher", zap.Error(err))
	}

	<-make(chan struct{})
}

func buildProject(projectDir string) error {
	cmd := exec.Command("go", "build", "-C", projectDir, "-o", config.Global.BinOutputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runBinary(binaryPath string) (*exec.Cmd, error) {
	cmd := exec.Command(binaryPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

func restartBinary(currentProcess *exec.Cmd, projectPath, binaryPath string) (*exec.Cmd, error) {
	var err error

	err = buildProject(projectPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao compilar o projeto: %v", err)
	}

	if currentProcess != nil {
		err = killProcess(currentProcess)
		if err != nil {
			return nil, fmt.Errorf("erro ao matar processo anterior: %v", err)
		}
	}

	currentProcess, err = runBinary(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar binário: %v", err)
	}

	return currentProcess, nil
}

func killProcess(cmd *exec.Cmd) error {
	if cmd.Process != nil {
		return cmd.Process.Signal(syscall.SIGTERM)
	}

	return nil
}
