# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[unreleased]: https://github.com/cboone/snappy/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/cboone/snappy/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cboone/snappy/releases/tag/v0.1.0
