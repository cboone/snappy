# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-03-02

### Added

- `config` command and `config init` subcommand for managing configuration
- `DefaultConfigPath`, `WriteDefaultConfig`, and `FormatConfig` config helpers
- Size-based log file rotation (#5)
- `help` and `version` subcommands

### Fixed

- Guarded home-dir prefix check with path separator
- Derived `--config` flag help text from `DefaultConfigPath`
- Resolved default `LogDir` in `config.Load` and used constants for duration defaults
- Made config init atomic and unified config defaults
- Preserved log data when active file rename fails during rotation
- Guarded nil file after rotation and truncated rotated log
- Warned on rename failure during log rotation, used `max` builtin
- Guarded logger edge cases for zero limits
- Printed version subcommand output to stdout
- Used copy+reslice for ring buffer trimming (#13)

### Changed

- Migrated goreleaser config from `brews` to `homebrew_casks`

## [0.3.1] - 2026-03-02

### Fixed

- Moved shell instructions to `.github/instructions/` directory

## [0.3.0] - 2026-03-02

### Added

- Modernized TUI with Charmbracelet v2 components (Bubbles viewport, spinner)
- Shell script installer for GitHub Releases

### Fixed

- Guarded loading flag in handleThinResult when snapshotting
- Guarded loading flag and refreshed viewport content on resize/theme change
- Aligned TUI viewport behavior with plan
- Used chronological log order and strengthened spinner test
- Hardened install script per review feedback
- Scoped MD014 markdownlint disable to scrut test files only
- Used $(OUTDIR) variable in scrut Makefile targets
- Broadened version glob pattern to match untagged CI builds

### Changed

- Aligned struct fields in TUI styles
- Applied shfmt formatting to install script
- Switched all CI workflows from Ubuntu to macOS runners
- Added scrut CLI integration tests for help, version, errors, config, and startup
- Documented macOS-only platform requirement

## [0.2.0] - 2026-03-01

### Added

- Go TUI built with Bubbletea, Cobra, and Viper, replacing the bash proof-of-concept
- Linting and formatting toolchain (golangci-lint, Prettier, markdownlint, EditorConfig, shfmt, Actionlint)

### Changed

- Moved bash reference implementation to docs/proof-of-concept
- Hoisted snapshot date extraction regex to package level

### Fixed

- Added in-flight guards for snapshot creation and thinning
- Fixed race between auto-snapshot creation and refresh in handleTick
- Added context timeouts to startup probes and APFS volume operations
- Preserved APFS volume on error during discovery
- Added warning log when APFS volume discovery fails
- Cleared stale diff counters and refreshPending on error path
- Hardened refresh edge cases
- Fixed CI lint and test failures (shfmt installation, Prettier `--ignore-unknown`)

## [0.1.0] - 2026-03-01

### Added

- Interactive snapshot manager TUI for browsing and deleting local snapshots
- APFS snapshot UUID and purgeable details display
- LimitingContainerShrink flag per snapshot
- Auto-snapshot creation with two-phase snapshot interval
- Delegation of snapshot retention to macOS

### Fixed

- Use `+=` instead of post-increment to avoid `set -e` exit
- Check for Time Machine snapshots when selecting APFS volume

### Changed

- Add shellcheckrc with all checks enabled

[unreleased]: https://github.com/cboone/snappy/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/cboone/snappy/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/cboone/snappy/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/cboone/snappy/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/cboone/snappy/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cboone/snappy/releases/tag/v0.1.0
