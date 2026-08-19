# GoMon

GoMon watches a Go project, rebuilds it after source changes, and restarts the
application when the new build succeeds.

## Requirements

- Go 1.25.0 or newer
- A Go project with a runnable package

## Installation

```bash
go install github.com/hugocarreira/gomon@latest
```

## Usage

```bash
gomon [flags] [project_path]
```

Examples:

```bash
gomon .
gomon --path ./services/api --binary ./tmp/api
gomon --config ./dev.gomon.yaml ./services/api
```

Flags override configuration-file values:

| Flag | Description |
| --- | --- |
| `--path` | Project directory; defaults to the current directory |
| `--binary` | Output binary path; relative paths are resolved from the project |
| `--config` | Explicit configuration file |
| `--debounce` | Debounce delay in milliseconds |
| `--log-level` | `debug`, `info`, `warn`, or `error` |
| `--version` | Print version, commit, and build date |
| `--help` | Print usage |

When `--binary` is omitted, GoMon uses a unique temporary executable and
removes it on shutdown. A failed build leaves the previous application running.

## Configuration

Create `.gomon.yaml` in the watched project:

```yaml
binary_path: ./tmp/api
debounce_time: 2s
log_level: info
```

The legacy `<project>/config/config.yaml` location is still accepted with a
deprecation warning. If no configuration file exists, built-in defaults are
used. `debounce_time` accepts Go duration strings such as `2s`; legacy numeric
values are interpreted as milliseconds.

By default GoMon rebuilds for changes to `.go`, `go.mod`, `go.sum`, `go.work`,
and `go.work.sum`. Ignored directories include `.git`, `.svn`, `.hg`,
`vendor`, `node_modules`, `.idea`, and `.vscode`. New directories are discovered
without restarting GoMon.

## Development

```bash
go mod download
go build -o gomon .
go test ./...
go test -race ./...
go vet ./...
```

The GitHub Actions workflow additionally checks formatting, module tidiness,
linting, and Linux/macOS/Windows builds. Release artifacts are produced by
GoReleaser for Linux, macOS, and Windows.

## Architecture

- `config`: project-aware configuration loading and validation
- `files`: recursive directory registration and build-input filtering
- `events`: filesystem event coalescing and rebuild scheduling
- `builder`: staged builds and child-process lifecycle management
- `watcher`: fsnotify loop and shutdown coordination
- `logger`: structured, colorized terminal output
