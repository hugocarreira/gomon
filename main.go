package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hugocarreira/gomon/config"
	"github.com/hugocarreira/gomon/files"
	"github.com/hugocarreira/gomon/logger"
	"github.com/hugocarreira/gomon/watcher"

	"go.uber.org/zap"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {

	helpFlag := flag.Bool("help", false, "Show usage information")
	projectPath := flag.String("path", "", "Path to the Go project to watch (default: current directory)")
	binaryPath := flag.String("binary", "", "Path to the output binary")
	debounce := flag.Int("debounce", 0, "Debounce time in milliseconds")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("gomon version %s (commit: %s, date: %s)\n", version, commit, date)
		os.Exit(0)
	}

	if *helpFlag {
		fmt.Println("GoMon usage:")
		fmt.Println("  -path string    Path to the Go project to watch (default: current directory)")
		fmt.Println("  -binary string  Path to the output binary")
		fmt.Println("  -debounce int   Debounce time in milliseconds (default: 2000)")
		fmt.Println("  -version        Show version information")
		fmt.Println("  -help           Show this help message")
		os.Exit(0)
	}

	log := logger.NewLogger()

	err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration", zap.Error(err))
	}

	if *projectPath != "" {
		config.Global.BinaryPath = *binaryPath
	}
	if *debounce > 0 {
		config.Global.DebounceTime = time.Duration(*debounce) * time.Millisecond
	}

	actualPath, err := files.DefineProjectPathWithFlag(*projectPath, os.Args)
	if err != nil {
		log.Fatal("Failed to determine project path", zap.Error(err))
	}
	log.Debug("Project path set", zap.String("path", actualPath))

	err = files.VerifyProjectPath(actualPath)
	if err != nil {
		log.Fatal("Project directory not found", zap.String("projectPath", actualPath))
	}

	w, err := watcher.NewWatcher(actualPath, config.Global, log)
	if err != nil {
		log.Fatal("Failed to create watcher", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	}()

	err = w.Start(ctx)
	if err != nil {
		log.Fatal("Failed to start watcher", zap.Error(err))
	}
}
