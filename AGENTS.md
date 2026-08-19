# Repository Guidelines

## Project Structure & Module Organization

GoMon is a Go 1.25 CLI that watches projects, rebuilds binaries, and restarts
processes. The entry point is `main.go`. Core packages are organized by
responsibility: `config/` loads and validates settings, `files/` manages
directories and build inputs, `events/` coalesces filesystem events, `builder/`
handles builds and child processes, `watcher/` owns the fsnotify loop, and
`logger/` provides terminal logging. Tests live beside their package code as
`*_test.go`; platform-specific process code is in `builder/process_unix.go` and
`builder/process_windows.go`. GitHub Actions workflows are under
`.github/workflows/`.

## Build, Test, and Development Commands

```bash
go mod download       # Download dependencies
go build -o gomon .   # Build the CLI
go run . .            # Run against the current project
go test ./...         # Run all tests
go test -race ./...   # Run tests with the race detector
go vet ./...          # Run static checks
gofmt -w .            # Format Go sources
```

CI also requires `go mod tidy -diff`, clean formatting, and
golangci-lint v2.12.2. Use the repository’s Go version from `go.mod`.

## Coding Style & Naming Conventions

Use idiomatic, gofmt-formatted Go with standard-library imports separated from
third-party imports. Use PascalCase for exported identifiers and camelCase for
locals. Keep functions focused, handle errors explicitly, wrap errors with
context, and use structured zap fields in logs. Preserve interfaces when they
support dependency injection or testing.

## Testing Guidelines

Add regression tests for behavior changes, keep tests next to the implementation,
and prefer table-driven cases for related inputs. Run both `go test ./...` and
`go test -race ./...` before submitting changes; include `go vet ./...` for
static validation.

## Commit & Pull Request Guidelines

Recent commits use concise prefixes such as `(fix)`, `(test)`, `(docs)`,
`(ci)`, and `(chore)`. Follow that pattern with an imperative, focused subject.
Pull requests should explain the behavioral change, mention configuration or
platform effects, link related issues when applicable, and list validation
commands run. Screenshots are not normally relevant for this CLI.
