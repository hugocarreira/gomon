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
		printUsage(stdout)
	}

	if err := flagSet.Parse(args); err != nil {
		return 2
	}
	if showVersion {
		fmt.Fprintf(stdout, "gomon version %s (commit: %s, date: %s)\n", version, commit, date)
		return 0
	}
	if help {
		printUsage(stdout)
		return 0
	}

	positional := flagSet.Args()
	if len(positional) > 1 {
		fmt.Fprintln(stderr, "only one positional project path is allowed")
		return 2
	}
	if projectPath != "" && len(positional) == 1 {
		fmt.Fprintln(stderr, "use either --path or a positional project path, not both")
		return 2
	}
	if debounce < 0 {
		fmt.Fprintln(stderr, "debounce must not be negative")
		return 2
	}
	if logLevel != "" && !config.ValidLogLevel(logLevel) {
		fmt.Fprintf(stderr, "unsupported log level %q\n", logLevel)
		return 2
	}
	logLevel = strings.ToLower(strings.TrimSpace(logLevel))

	pathArgs := []string{"gomon"}
	pathArgs = append(pathArgs, positional...)
	actualPath, err := files.DefineProjectPathWithFlag(projectPath, pathArgs)
	if err != nil {
		fmt.Fprintf(stderr, "failed to determine project path: %v\n", err)
		return 1
	}
	actualPath, err = filepath.Abs(actualPath)
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve project path: %v\n", err)
		return 1
	}
	if err := files.VerifyProjectPath(actualPath); err != nil {
		fmt.Fprintf(stderr, "project directory not found: %v\n", err)
		return 1
	}

	baseLog := logger.NewLogger()
	config.AppLogger = baseLog
	cfg, err := config.LoadConfigForProject(actualPath, configPath)
	if err != nil {
		baseLog.Error("Failed to load configuration", zap.Error(err))
		return 1
	}
	if binaryPath != "" {
		cfg.BinaryPath = binaryPath
	}
	if debounce > 0 {
		cfg.DebounceTime = config.DurationFromMilliseconds(debounce)
	}
	if logLevel != "" {
		cfg.LogLevel = logLevel
	}
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

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "GoMon usage:")
	fmt.Fprintln(out, "  gomon [flags] [project_path]")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  -path string       Path to the Go project to watch (default: current directory)")
	fmt.Fprintln(out, "  -binary string     Path to the output binary")
	fmt.Fprintln(out, "  -config string     Path to a configuration file")
	fmt.Fprintln(out, "  -debounce int      Debounce time in milliseconds (default: 2000)")
	fmt.Fprintln(out, "  -log-level string  Log level: debug, info, warn, or error")
	fmt.Fprintln(out, "  -version           Show version information")
	fmt.Fprintln(out, "  -help              Show this help message")
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
