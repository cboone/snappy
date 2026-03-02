# 2026-03-01: Comprehensive scrut CLI integration tests

## Context

The initial scrut test infrastructure was added in commit `71d47c6` with two
test files (`help.md` and `version.md`) covering basic `--help` and `--version`
flag output. This plan adds comprehensive CLI testing for error handling,
config flag behavior, and startup behavior, bringing the total from 4 to 17
test cases. It also fixes the CI to run scrut tests on macOS (since snappy is
macOS-only).

## CI fix: switch scrut tests from Ubuntu to macOS

Snappy requires macOS (`tmutil`, APFS volumes, Time Machine). The CI
`test-scrut` job currently runs on `ubuntu-latest`, which means any test that
reaches `RunE` would get the tmutil-not-found error instead of the real TUI
startup error. Switching to `macos-latest` (Apple Silicon / arm64) gives
accurate test results.

Changes to `.github/workflows/ci.yml` in the `test-scrut` job:

1. `runs-on: ubuntu-latest` changes to `runs-on: macos-latest`
2. Scrut download pattern changes from `scrut-v0.4.3-linux-x86_64.tar.gz` to
   `scrut-v0.4.3-macos-aarch64.tar.gz`
3. Extracted directory changes from `scrut-linux-x86_64` to
   `scrut-macos-aarch64`

## Constraints

- Snappy is macOS-only. Both local dev and CI run on macOS, so `tmutil` is
  always present. Tests that reach `RunE` fail at the TUI launch (no TTY in
  scrut/CI), producing `Error: running TUI: *`.
- Cobra's `SilenceUsage: true` and `SilenceErrors: true` mean errors are NOT
  printed by Cobra itself. Errors are returned to `main.go` which prints
  `Error: <msg>` to stderr and exits 1.
- Scrut's `{output_stream: stderr}` inline configuration is used to test
  stderr output in isolation.

## New test files

### 1. `tests/scrut/errors.md` -- Invalid CLI input

All tests here fail at the Cobra flag-parsing level (before `RunE`), so they
are platform-independent.

| Test case                       | Command                      | Expected stderr                            | Exit |
| ------------------------------- | ---------------------------- | ------------------------------------------ | ---- |
| Unknown long flag               | `--bogus`                    | `Error: unknown flag: --bogus`             | 1    |
| Unknown short flag              | `-z`                         | `Error: unknown shorthand flag: 'z' in -z` | 1    |
| Config flag without value       | `--config` (no arg)          | `Error: flag needs an argument: --config`  | 1    |
| Unknown flag after help flag    | `--help --bogus`             | `Error: unknown flag: --bogus`             | 1    |
| Unknown flag produces no stdout | `--unknown-flag 2>/dev/null` | (empty stdout)                             | 1    |

Key insights documented by these tests:

- `SilenceUsage: true` suppresses usage output on errors (only the error line
  appears, not the full help text).
- `--help --bogus` still errors because Cobra parses all flags before
  dispatching help.

### 2. `tests/scrut/config.md` -- Config flag behavior

Tests `--config` flag interactions with `--help` and `--version`, plus the
warning behavior for nonexistent config files.

| Test case                       | Command                               | Check  | Expected                                      | Exit |
| ------------------------------- | ------------------------------------- | ------ | --------------------------------------------- | ---- |
| Help with config (help first)   | `--help --config /nonexistent/...`    | stdout | Full help text                                | 0    |
| Help with config (config first) | `--config /nonexistent/... --help`    | stdout | Full help text                                | 0    |
| Version with config             | `--version --config /nonexistent/...` | stdout | `snappy version v*`                           | 0    |
| Nonexistent config warning      | `--config /nonexistent/...`           | stderr | Warning line + `Error: running TUI: * (glob)` | 1    |

Key insights:

- `--help` and `--version` short-circuit before `RunE`, so a bad config path
  does not prevent them from succeeding.
- When `--config` points to a file that does not exist, `initConfig()` prints
  a `Warning: config file error: ...` line to stderr (because the error is NOT
  a `viper.ConfigFileNotFoundError`), then execution continues to `RunE`.

### 3. `tests/scrut/startup.md` -- Startup behavior

Tests the binary's behavior when it actually attempts to run (no `--help`,
`--version`, or flag errors). On macOS without a TTY, the binary gets past
the `tmutil` check but fails launching the TUI.

| Test case                 | Command                      | Check  | Expected                       | Exit |
| ------------------------- | ---------------------------- | ------ | ------------------------------ | ---- |
| Bare invocation error     | (no args)                    | stderr | `Error: running TUI: * (glob)` | 1    |
| Bare invocation no stdout | (no args, stderr suppressed) | stdout | (empty)                        | 1    |
| Extra argument accepted   | `some-argument`              | stderr | `Error: running TUI: * (glob)` | 1    |
| Multiple extra arguments  | `arg1 arg2 arg3`             | stderr | `Error: running TUI: * (glob)` | 1    |

Key insight:

- The `Error: running TUI: *` pattern validates that the binary reached the
  TUI launch stage (past tmutil check, config loading, logger init, and APFS
  volume discovery).
- Cobra's default argument handling (`legacyArgs`) accepts arbitrary positional
  arguments when there are no subcommands. The binary proceeds to `RunE`
  without erroring on unknown args. If `Args: cobra.NoArgs` is added later,
  these tests will catch the behavior change.

## Implementation steps

1. Update `.github/workflows/ci.yml` to run scrut tests on macOS
2. Create `tests/scrut/errors.md`
3. Create `tests/scrut/config.md`
4. Create `tests/scrut/startup.md`
5. Run `make test-scrut` to verify all tests pass
6. Run `make lint` and `make fmt-check` to verify formatting/linting
7. Invoke `write-scrut-tests` skill for style review

## Files to modify

- `.github/workflows/ci.yml` -- switch `test-scrut` job to macOS

## Files to create

- `tests/scrut/errors.md`
- `tests/scrut/config.md`
- `tests/scrut/startup.md`

## Reference files

- `tests/scrut/help.md` -- formatting conventions and code fence syntax
- `tests/scrut/version.md` -- glob pattern syntax
- `cmd/root.go` -- all CLI behavior, flag definitions, error handling
- `main.go` -- error formatting (`Error: %s`) and exit code logic
- `.markdownlint-cli2.jsonc` -- markdown lint rules for test files

## Verification

1. `make test-scrut` passes locally (macOS)
2. `make lint` passes (markdownlint accepts new files)
3. `make fmt-check` passes (Prettier formatting)
4. CI passes on macOS runner

## Test inventory after implementation

| File                     | Test cases | Status   |
| ------------------------ | ---------- | -------- |
| `tests/scrut/help.md`    | 2          | Existing |
| `tests/scrut/version.md` | 2          | Existing |
| `tests/scrut/errors.md`  | 5          | New      |
| `tests/scrut/config.md`  | 4          | New      |
| `tests/scrut/startup.md` | 4          | New      |
| **Total**                | **17**     |          |
