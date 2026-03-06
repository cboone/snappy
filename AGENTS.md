# Snappy

## Overview

Automatically increase your Time Machine snapshot frequency. **macOS only.**
Depends on `tmutil`, `diskutil`, and APFS snapshots.

## Structure

```text
main.go                          CLI entry point
cmd/root.go                      Cobra root command, launches TUI
bin/snappy-ez                    Standalone bash snapshot manager (sourceable)
internal/
  config/config.go               Viper-backed configuration
  platform/
    exec.go                      CommandRunner interface
    tmutil.go                    tmutil command wrappers
    diskutil.go                  APFS plist parsing (howett.net/plist)
    disk.go                      df disk info wrapper
  snapshot/
    snapshot.go                  Snapshot type, date parsing, relative time
    diff.go                      Snapshot list diffing
    auto.go                      Auto-snapshot timer and thinning logic
  logger/logger.go               Ring buffer + file logger
  service/
    lock.go                      flock-based single-instance locking
    launchd.go                   launchd plist generation + lifecycle
  tui/
    model.go                     Bubbletea model + Init
    update.go                    Update (message dispatch)
    view.go                      View (header/list/log/controls)
    styles.go                    Lipgloss style definitions
    messages.go                  Custom tea.Msg types
    commands.go                  tea.Cmd factories for async operations
tests/scrut/                     Scrut CLI tests for the Go binary
tests/scrut/snappy-ez/           Scrut tests for snappy-ez bash script
docs/proof-of-concept/snappy     Original bash TUI (reference)
docs/plans/                      Design plans and decision records
```

## Development

Build:

```bash
make build
```

Run tests:

```bash
make test              # Go unit tests
make test-scrut        # Scrut CLI tests (Go binary)
make test-scrut-ez     # Scrut tests (snappy-ez bash script)
make test-all          # All tests (unit + scrut + scrut-ez)
```

Lint:

```bash
make lint          # Run all linters (golangci-lint, markdownlint, actionlint)
make lint-go       # Run golangci-lint only
make lint-md       # Lint Markdown files
make lint-actions  # Lint GitHub Actions workflows
make vet           # Run go vet
```

Format:

```bash
make fmt           # Format all code (Go, Markdown, JSON, YAML, shell)
make fmt-check     # Check formatting without modifying files (used in CI)
```

See all targets:

```bash
make help
```
