# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.8.3] - 2026-03-07

### Fixed

- Handle launchctl bootstrap error 125 with retry logic
- Handle launchctl enable errors and use idiomatic error formatting
- Use consistent wording in bootstrap error message
- Remove trailing newline from caveats, add explanatory comment
- Use single-line caveats to avoid GoReleaser cask template bug

### Changed

- Clarify bootout retry comment to match actual behavior
- Update plan snippet to match final implementation
- Add plan for fix-service-install-bootstrap-error
- Add plan for fixing Homebrew install tap migration and caveats

## [0.8.2] - 2026-03-07

### Fixed

- Guarded DefaultLockPath against empty LogDir (#85)
- Rate-limited repeated tidemark WARN logs (#86)

### Changed

- Moved plan to done for guard-defaultlockpath-empty-logdir

## [0.8.1] - 2026-03-07

### Fixed

- Added persistent TUI lock for mutual exclusion between instances
- Deferred TUI lock release until auto-snapshots finish
- Removed double lock release from cmd/root.go success path
- Used system_command DSL in Homebrew cask postflight hook

## [0.8.0] - 2026-03-07

### Added

- Diagonal flash border effect on panel focus change
- Lock around TUI auto-snapshots to prevent race with daemon
- Tidemark fetch failure logging in TUI refresh
- Decoupled mouse wheel scroll from selection in TUI panels
- Derived service log path from installed plist instead of config

### Fixed

- Addressed Copilot review feedback
- Corrected IsHeld comment and removed premature guard in log viewport
- Kept log viewport stable on resize
- Ignored TUI autosnapshot lock in daemon sync
- Resolved lint issues from prior commits
- Used shared lock in IsHeld to eliminate race with Acquire
- Preserved log viewport position when new entries arrive
- Prevented log cursor drift when ring buffer is at capacity
- Excluded user-created snapshots from first-refresh FOUND count
- Used clipped snapshot content in flash path to prevent layout jump
- Used lipgloss.Width for consistent display-width measurement in flash title
- Updated stale comment and simplified constant expression
- Returned flash tick command from shift-tab handler
- Ignored stale flash ticks during focus changes
- Cached snapshot table render output for repaint performance

### Changed

- Used plist serializer instead of text/template for GeneratePlist
- Replaced NewModel parameter list with ModelParams struct
- Extracted helpers from renderInfoPanel to reduce cyclomatic complexity
- Derived snapVisibleRows default from table height constant
- Added gold glint shimmer effect on panel focus change
- Tuned flash animation speed per color scheme
- Reversed flash shimmer direction for shift-tab navigation
- Sped up flash animation tick from 30ms to 25ms
- Increased dark-mode contrast for unfocused panels and help bar

### Performance

- Only update AGE column on UI tick instead of full table rebuild

## [0.7.0] - 2026-03-06

### Added

- Background service with launchd plist generation and lifecycle management
- Single-instance lock via flock to prevent concurrent daemon runs
- Lock acquisition and shared logger for daemon mode
- `service` management commands (install, uninstall, start, stop, status)
- Service setup instructions added to install paths
- TUI detection of running daemon with auto-snapshot disabling
- Container reference, tidemark, and FormatBytes in platform layer
- XID delta column and tidemark display in TUI
- XID delta in CLI `list` output with container discovery
- Summarized first-run snapshot discovery as single FOUND log entry
- Seed ring buffer from existing log file on startup
- `l` key binding to open log directory from TUI
- Pinned indicator in TUI snapshot table for ESTALE failures
- Shift+Tab to cycle panel focus in reverse

### Fixed

- Propagated write errors from logLine in run daemon
- Logged resolved startup volume name at startup
- Suppressed redundant ADDED log after snapshot creation
- Suppressed redundant REMOVED log after thinning, added thinning start log
- Classified ESTALE as warning with user-friendly message
- Only logged refresh summary when data actually changes
- Rechecked daemon lock on each refresh tick
- Derived running state from launchctl pid for service status
- Warned when service binary path differs from installed path
- Avoided invalid future relative time labels
- Widened AGE column to fit "future" label
- Removed extra argument in doRefresh call after thin
- Showed positional-arg placeholders in usage template
- Combined usage line for commands with subcommands
- Used "service" instead of "daemon" in TUI status label

### Changed

- Extracted getDiskutilInfo to deduplicate plist parsing
- Replaced math.Log in FormatBytes with loop
- Removed mount point configuration
- Separated log Level from event Category in logger
- Migrated logger calls to Level + Category API
- Delegated dualLog to logLine for writer output
- Used howett.net/plist for readBinaryFromPlist
- Extracted inline Update cases to reduce cyclomatic complexity

### Performance

- Skipped UI tick when auto-snapshot is disabled and idle
- Cached default log entry style

## [0.6.0] - 2026-03-06

### Added

- Non-interactive CLI commands: `create`, `list`, `status`, `thin`, and `run`
- Bubbles table component for snapshot list with clickable line selection
- Clickable panel focus and mouse wheel scrolling for TUI panels
- Live-updating snapshot ages and auto-snap countdown timer (1-second tick)
- Long log message wrapping within the message column
- Human-friendly volume name display in info panel
- Dynamic status dot and operation labels in TUI header
- XID field in snapshot details
- Embedded border titles for snapshot, log, and info panels

### Changed

- Showed newest snapshots first in both panels for consistent scrolling
- Used ANSI color constants instead of hex strings for TUI styles
- Compacted relative time format for snapshot ages
- Moved APFS details from info panel to log
- Tuned panel and help bar colors for focus contrast
- Restyled panel titles and table headers with uniform column spacing
- Unified batch-delete and surfaced snapshot errors in status
- Optimized GitHub Actions runner usage

### Fixed

- Stabilized log cursor and prevented strict-mode leak when sourced
- Restored "ago" suffix in CLI list output
- Corrected column sizing math and border title overflow
- Aligned snapshot click selection with visible rows
- Bounded mouse hit-testing to panel regions
- Computed fixed layout height dynamically for proper terminal filling
- Preserved TUI output on exit instead of clearing screen
- Prevented panel content wrapping and handled narrow terminals
- Removed double `v` prefix from startup log
- Skipped pinned snapshot failures and prevented retry loops in thinning
- Handled ESTALE deletes as pinned skips in snappy-ez
- Used `cmd.Context()` for cancellation propagation
- Serialized release workflow runs across all tags
- Corrected install script `tmp_dir` scope and added source guard
- Used string instead of list for cask post-install hook

## [0.5.0] - 2026-03-02

### Added

- Standalone `snappy-ez` bash snapshot manager script, sourceable or directly executable

### Changed

- Made snappy-ez sourceable with BASH_SOURCE guard
- Hoisted loop locals and added VERSION constant in snappy-ez
- Added scrut tests for snappy-ez
- Added snappy-ez section to README
- Added community files for open-source readiness (CONTRIBUTING, CODE_OF_CONDUCT, SECURITY, PR template)

### Fixed

- Made `clean` target preserve tracked `bin/snappy-ez` script
- Updated `test-all` help comment to include scrut-ez
- Fixed bash 3.2 empty array expansion and demoted VERSION to comment
- Added macOS system bash support in snappy-ez

## [0.4.1] - 2026-03-02

### Changed

- Added `config` and `config init` usage documentation to README

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

[unreleased]: https://github.com/cboone/snappy/compare/v0.8.3...HEAD
[0.8.3]: https://github.com/cboone/snappy/compare/v0.8.2...v0.8.3
[0.8.2]: https://github.com/cboone/snappy/compare/v0.8.1...v0.8.2
[0.8.1]: https://github.com/cboone/snappy/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/cboone/snappy/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/cboone/snappy/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/cboone/snappy/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/cboone/snappy/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/cboone/snappy/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/cboone/snappy/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/cboone/snappy/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/cboone/snappy/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/cboone/snappy/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cboone/snappy/releases/tag/v0.1.0
