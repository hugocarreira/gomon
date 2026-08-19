# Progress

## Current status

The GoMon reliability hardening is implemented on `fix/reliability-hardening`.
The branch has an open draft PR: https://github.com/hugocarreira/gomon/pull/4

## Completed

- Refreshed agent guidance and documented the Go 1.25 baseline.
- Added optional project configuration, CLI precedence, duration validation, and log-level support.
- Added staged builds, temporary outputs, process-group/Job Object cleanup, and idempotent shutdown.
- Added build-input filtering, dynamic directory discovery, trailing-edge debounce, and fsnotify bitmask handling.
- Added regression and integration tests across CLI, config, files, events, builder, and watcher packages.
- Updated dependencies, CI gates, GoReleaser metadata, and README usage documentation.

## Verification

- 66 tests pass, including race-detector coverage.
- `go vet ./...`, golangci-lint v2.12.2, `go mod tidy -diff`, and formatting checks pass.
- Linux build plus macOS and Windows cross-builds pass.
- Statement coverage is 80.2%.

Pre-existing untracked `CHANGELOG.md` and `LICENSE` remain outside the branch commits.
