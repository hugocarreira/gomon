# GoMon - Nodemon inspired for Golang

A practical development routine for Go projects.


## Why this repo exists
- A practical, self-contained dev workflow you can drop into any Go project.
- A repeatable routine for building, watching, testing, linting, and shipping, with sensible defaults.
- Clear guidance on how to extend and customize the workflow for your team.

## Installation

```
go install github.com/hugocarreira/gomon@latest
```

## Usage (bare minimum)
```
gomon ./path/to/your/go/project
```

This will start the watcher, rebuild on changes, and restart the binary as you edit code.

Notes:
- Requires Go 1.25.0.
- The dev routine assumes a conventional Go module layout and a main package in the target project.

## Development routine (your daily workflow)
This section describes a repeatable, developer-friendly cycle that uses this lib as a core part of a daily workflow.

- Start the watcher for a project you’re actively developing.
- Edit code and rely on automatic rebuilds/tests when changes occur.
- Run unit tests and lint locally to ensure quality before commits.
- Iterate on bug fixes, new features, and improvements in a fast feedback loop.
- Use the included commands to verify build, tests, and lint status before pushing.

Architecture overview:
- watcher: watches filesystem changes and triggers rebuilds
- builder: builds the Go project into a binary
- runner: restarts the binary on successful builds
- config: settings loaded via patterns (where applicable)
- tests: unit tests for core components (where present)

This layout keeps the runtime lean while providing a robust development loop.

## Development

Prerequisites:
- Go 1.25.0

Install
```
go mod download
go build -o gomon .
```

Usage
```
./gomon /path/to/your/go/project
```

Configuration (where to customize)
- If you need to customize paths or behavior, edit the relevant config in the project (see config/*) or pass flags as supported by the repo.

Tests and linting
```
go test ./...
golangci-lint run
```
