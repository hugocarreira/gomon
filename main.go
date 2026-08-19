package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

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

// run executes the CLI and returns a process exit code. Keeping this separate
// from main makes argument handling and startup failures testable.
func run(args []string, stdout, stderr io.Writer) int {
	flagSet := flag.NewFlagSet("gomon", flag.ContinueOnError)
	flagSet.SetOutput(stderr)

	var projectPath string
	var binaryPath string
	var configPath string
	var debounce int
	var logLevel string
	var help bool
	var showVersion bool

	flagSet.StringVar(&projectPath, "path", "", "Path to the Go project to watch (default: current directory)")
	flagSet.StringVar(&binaryPath, "binary", "", "Path to the output binary")
	flagSet.StringVar(&configPath, "config", "", "Path to a configuration file")
	flagSet.IntVar(&debounce, "debounce", 0, "Debounce time in milliseconds (default: 2000)")
	flagSet.StringVar(&logLevel, "log-level", "", "Log level: debug, info, warn, or error")
	flagSet.BoolVar(&showVersion, "version", false, "Show version information")
	flagSet.BoolVar(&help, "help", false, "Show usage information")
	flagSet.Usage = func() {
		_ = printUsage(stdout)
	}

	if err := flagSet.Parse(args); err != nil {
		return 2
	}
	if showVersion {
		if _, err := fmt.Fprintf(stdout, "gomon version %s (commit: %s, date: %s)\n", version, commit, date); err != nil {
			return 1
		}
		return 0
	}
	if help {
		if err := printUsage(stdout); err != nil {
			return 1
		}
		return 0
	}

	positional := flagSet.Args()
	if len(positional) > 1 {
		if _, err := fmt.Fprintln(stderr, "only one positional project path is allowed"); err != nil {
			return 1
		}
		return 2
	}
	if projectPath != "" && len(positional) == 1 {
		if _, err := fmt.Fprintln(stderr, "use either --path or a positional project path, not both"); err != nil {
			return 1
		}
		return 2
	}
	if debounce < 0 {
		if _, err := fmt.Fprintln(stderr, "debounce must not be negative"); err != nil {
			return 1
		}
		return 2
	}
	if logLevel != "" && !config.ValidLogLevel(logLevel) {
		if _, err := fmt.Fprintf(stderr, "unsupported log level %q\n", logLevel); err != nil {
			return 1
		}
		return 2
	}
	logLevel = strings.ToLower(strings.TrimSpace(logLevel))

	pathArgs := []string{"gomon"}
	pathArgs = append(pathArgs, positional...)
	actualPath, err := files.DefineProjectPathWithFlag(projectPath, pathArgs)
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "failed to determine project path: %v\n", err); writeErr != nil {
			return 1
		}
		return 1
	}
	actualPath, err = filepath.Abs(actualPath)
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "failed to resolve project path: %v\n", err); writeErr != nil {
			return 1
		}
		return 1
	}
	if err := files.VerifyProjectPath(actualPath); err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "project directory not found: %v\n", err); writeErr != nil {
			return 1
		}
		return 1
	}

	baseLog := logger.NewLogger()
	config.AppLogger = baseLog
	cfg, err := config.LoadConfigForProject(actualPath, configPath)
	if err != nil {
		baseLog.Error("Failed to load configuration", zap.Error(err))
		return 1
	}
	cfg.SetupOverrides(binaryPath, logLevel, debounce)
	log := logger.NewLoggerWithLevel(cfg.LogLevel)

	w, err := watcher.NewWatcher(actualPath, cfg, log)
	if err != nil {
		log.Error("Failed to create watcher", zap.Error(err))
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)
	go func() {
		select {
		case sig := <-sigChan:
			log.Info("Received shutdown signal", zap.String("signal", sig.String()))
			cancel()
		case <-ctx.Done():
		}
	}()

	if err := w.Start(ctx); err != nil {
		log.Error("Failed to start watcher", zap.Error(err))
		return 1
	}
	return 0
}

func printUsage(out io.Writer) error {
	lines := []string{
		"GoMon usage:",
		"  gomon [flags] [project_path]",
		"",
		"Flags:",
		"  -path string       Path to the Go project to watch (default: current directory)",
		"  -binary string     Path to the output binary",
		"  -config string     Path to a configuration file",
		"  -debounce int      Debounce time in milliseconds (default: 2000)",
		"  -log-level string  Log level: debug, info, warn, or error",
		"  -version           Show version information",
		"  -help              Show this help message",
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
