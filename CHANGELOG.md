# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[unreleased]: https://github.com/cboone/snappy/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/cboone/snappy/releases/tag/v0.1.0
