# AGENTS.md - GoMon Development Guide

This document provides essential information for agentic coding agents working on the GoMon project.

## Project Overview

GoMon is a file watcher for Golang projects that automatically rebuilds and restarts the binary when source files change (similar to nodemon for Node.js).

- **Language**: Go 1.23.0
- **Module**: github.com/hugocarreira/gomon
- **Build Output**: `gomon` binary

---

## Build & Run Commands

### Running the Application

```bash
# Build the binary
go build -o gomon .

# Run the application
go run . [project_path]

# Run with specific config
./gomon ./your-go-project
```

### Testing

```bash
# Run all tests
go test -v ./...

# Run tests for a specific package
go test -v ./watcher

# Run a single test
go test -v -run TestFunctionName ./...

# Run with coverage
go test -v -cover ./...
```

### Linting

```bash
# Run golangci-lint (version v1.60 as per CI)
golangci-lint run

# Run golangci-lint on specific files
golangci-lint run ./...

# Lint with auto-fix
golangci-lint run --fix
```

### Build Release

```bash
# Using goreleaser (see .goreleaser.yaml)
goreleaser build --snapshot --clean
```

---

## Code Style Guidelines

### General Principles

- Follow standard Go conventions (effective Go)
- Use `gofmt` for code formatting (included in golangci-lint)
- Keep functions small and focused
- Use interfaces for abstraction (already established pattern)

### Imports

**Standard Grouping** (required order):
1. Standard library packages
2. Third-party packages

```go
import (
    "os"
    "time"

    "github.com/fsnotify/fsnotify"
    "github.com/hugocarreira/gomon/config"
    "go.uber.org/zap"
)
```

- **Avoid** alias imports unless necessary (e.g., `log "github.com/sirupsen/logrus"`)

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Packages | lowercase, short | `watcher`, `config`, `builder` |
| Interfaces | PascalCase, I-prefix | `IBuilder`, `ILogger`, `IEventsHandler` |
| Structs | PascalCase | `Watcher`, `Builder`, `Logger` |
| Functions | PascalCase (exported), camelCase (unexported) | `NewWatcher`, `LoadConfig` |
| Variables | camelCase | `projectPath`, `binaryPath` |
| Constants | PascalCase | `DefaultTimeout` |
| Error variables | Err-prefix for sentinel errors | `ErrNotFound` |

### Types

- Use struct tags for configuration (`mapstructure:"field_name"`)
- Prefer explicit types over generic `any` / `interface{}`
- Use time.Duration for time intervals

```go
type Config struct {
    BinaryPath   string        `mapstructure:"binary_path"`
    DebounceTime time.Duration `mapstructure:"debounce_time"`
}
```

### Error Handling

- **Always** handle errors explicitly - never ignore with `_`
- Return errors from functions, don't panic except for unrecoverable states
- Use structured logging with zap for error context

```go
// Good
func NewWatcher(projectPath string, config *config.Config, logger log.ILogger) (*Watcher, error) {
    w, err := fsnotify.NewWatcher()
    if err != nil {
        logger.Fatal("Erro ao criar watcher", zap.Error(err))
        return nil, err
    }
    return &Watcher{...}, nil
}

// Bad - empty catch
func someFunc() {
    if err != nil {
        // do nothing
    }
}
```

### Logging

- Use the custom ILogger interface from `log` package
- Use zap fields for structured logging: `zap.String()`, `zap.Error()`, etc.
- Error messages can be in Portuguese (existing pattern) or English (prefer English for new code)

```go
logger.Info("Watcher iniciado para o diretório", zap.String("path", projectPath))
logger.Error("Erro ao carregar configurações", zap.Error(err))
```

### Interfaces (Established Pattern)

Define interfaces for dependencies to enable testing:

```go
type IBuilder interface {
    BuildProject() error
    RunBinary() (*exec.Cmd, error)
    RestartBinary() error
    KillProcess(cmd *exec.Cmd) error
}
```

---

## Project Structure

```
/home/hugoubuntu/work/gomon/
├── main.go              # Entry point
├── config/              # Configuration loading (viper)
├── watcher/             # File watcher (fsnotify)
├── builder/             # Go build/restart logic
├── events/              # Event handling with debouncing
├── files/               # File path utilities
├── log/                 # Logger (zap + fatih/color)
├── .github/workflows/   # CI (golangci-lint)
└── .goreleaser.yaml     # Release configuration
```

---

## Configuration

- Config file: `config/config.yaml`
- Uses viper for configuration management
- Default values set in code (see `config/config.go`)

---

## CI/CD

- **Linting**: golangci-lint v1.60 (runs on push to main/master and PRs)
- **Release**: goreleaser (see `.goreleaser.yaml`)

---

## Testing Guidelines

- Test files should be named `*_test.go`
- Place tests in the same package as the code being tested
- Use table-driven tests when testing multiple scenarios

```go
func TestBuildProject(t *testing.T) {
    tests := []struct {
        name    string
        wantErr bool
    }{
        {"valid project", false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

---

## Common Development Tasks

```bash
# Add a new dependency
go get github.com/some/package

# Update dependencies
go mod tidy

# Run the application with live reload
go run . .

# Check for issues
go vet ./...
```

---

## Notes for Agents

- This codebase has **no existing tests** - prioritize adding tests when modifying code
- The project uses Portuguese error messages in some places - maintain consistency with existing code
- Interfaces are used throughout for dependency injection and testability
- The project follows a clean package-based architecture
