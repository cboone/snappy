# Expand scrut CLI tests for new TUI

## Context

The TUI was substantially reworked (Charmbracelet v2 modernization, viewport
components, adaptive styling, spinner, scroll focus). The Go unit tests
(`internal/tui/model_test.go`) cover TUI internals well (20+ tests), but the
scrut CLI tests (17 tests in 5 files) only cover basic flag parsing and error
handling. Three significant areas have zero scrut coverage: environment
variables, config file content scenarios, and flag precedence behavior.

**Goal:** Expand from 17 to 38 scrut tests across 3 new files and 2 updated
files, covering environment variable integration, config file edge cases, and
flag interaction behavior.

## Files to create

### 1. `tests/scrut/environment.md` (9 tests)

Tests the `SNAPPY_*` environment variable system (`viper.SetEnvPrefix("SNAPPY")`
and `viper.AutomaticEnv()` in `cmd/root.go:62-63`).

| Test                                        | Command                                                                        | Validates                                    |
| ------------------------------------------- | ------------------------------------------------------------------------------ | -------------------------------------------- |
| SNAPPY_LOG_DIR env var affects logger setup  | `SNAPPY_LOG_DIR="/dev/null/snappy" "${SNAPPY_BIN}"`                            | Log dir env var read and applied             |
| SNAPPY_MOUNT env var reaches TUI stage      | `SNAPPY_MOUNT="/Volumes/Test" "${SNAPPY_BIN}"`                                 | String env var accepted, reaches TUI         |
| SNAPPY_MOUNT env var produces no stdout     | Same + `2>/dev/null`                                                           | No stdout leak with env vars                 |
| SNAPPY_AUTO_ENABLED=false reaches TUI stage | `SNAPPY_AUTO_ENABLED=false "${SNAPPY_BIN}"`                                    | Boolean env var accepted                     |
| SNAPPY_REFRESH with numeric value           | `SNAPPY_REFRESH=30 "${SNAPPY_BIN}"`                                            | Numeric duration env var works end-to-end |
| SNAPPY_REFRESH with Go duration string      | `SNAPPY_REFRESH="2m" "${SNAPPY_BIN}"`                                          | Duration string env var works end-to-end  |
| Multiple SNAPPY env vars together           | `SNAPPY_MOUNT=... SNAPPY_REFRESH=30 SNAPPY_AUTO_ENABLED=false "${SNAPPY_BIN}"` | Multiple env vars coexist                 |
| Env var with help flag                      | `SNAPPY_MOUNT="/Volumes/Test" "${SNAPPY_BIN}" --help`                          | Env vars don't interfere with --help      |
| Env var with version flag                   | `SNAPPY_MOUNT="/Volumes/Test" "${SNAPPY_BIN}" --version`                       | Env vars don't interfere with --version   |

All startup tests expect `Error: running TUI: * (glob)` on stderr with exit
code 1 (TUI fails without a TTY). Help and version tests expect normal output
with exit code 0.

### 2. `tests/scrut/config-files.md` (4 tests)

Tests config file content scenarios beyond the nonexistent path (already in
`config.md`). Uses `/dev/null` as an empty config input (Viper reports
unsupported config type) and `/tmp` as a directory-instead-of-file case.

| Test                              | Command                                                           | Validates                                                   |
| --------------------------------- | ----------------------------------------------------------------- | ----------------------------------------------------------- |
| Config pointing to a directory    | `"${SNAPPY_BIN}" --config /tmp`                                   | Warning for directory path                                  |
| Empty config file (via /dev/null) | `"${SNAPPY_BIN}" --config /dev/null`                              | Warning for unsupported config type, then reaches TUI stage |
| Empty config produces no stdout   | Same + `2>/dev/null`                                              | No stdout leak                                              |
| Config flag combined with env var | `SNAPPY_MOUNT="/Volumes/Test" "${SNAPPY_BIN}" --config /dev/null` | Full config pipeline (env > file > defaults)                |

### 3. `tests/scrut/flag-precedence.md` (4 tests)

Documents flag interaction behavior when multiple mode-switching flags are
combined. Locks down Cobra's precedence so version upgrades don't silently
change behavior.

| Test                             | Command                                               | Validates                          |
| -------------------------------- | ----------------------------------------------------- | ---------------------------------- |
| Help flag wins over version flag | `"${SNAPPY_BIN}" --help --version`                    | --help takes precedence            |
| Version flag before help flag    | `"${SNAPPY_BIN}" --version --help`                    | Order does not matter, --help wins |
| Short help and version flags     | `"${SNAPPY_BIN}" -v -h`                               | Short flags follow same precedence |
| All three flags together         | `"${SNAPPY_BIN}" --help --version --config /dev/null` | --help wins even with --config     |

**Note:** The --help vs --version precedence tests should be validated
empirically with `make test-scrut-update` first, since Cobra's exact behavior
depends on its internal flag processing order. The tests should capture and lock
down whatever the actual behavior is.

## Files to update

### 4. `tests/scrut/config.md` (+2 tests)

Add to existing file (currently 4 tests):

| Test                              | Command                                                            | Validates                                    |
| --------------------------------- | ------------------------------------------------------------------ | -------------------------------------------- |
| Config flag with equals syntax    | `"${SNAPPY_BIN}" --config=/nonexistent/path/config.yaml`           | `--config=value` works like `--config value` |
| Version with config equals syntax | `"${SNAPPY_BIN}" --version --config=/nonexistent/path/config.yaml` | Equals syntax with short-circuit flag        |

### 5. `tests/scrut/errors.md` (+2 tests)

Add to existing file (currently 5 tests):

| Test                  | Command                       | Validates                                      |
| --------------------- | ----------------------------- | ---------------------------------------------- |
| Double-dash separator | `"${SNAPPY_BIN}" -- --help`   | `--` stops flag parsing, --help treated as arg |
| Empty config value    | `"${SNAPPY_BIN}" --config ""` | Empty string config path behavior              |

## Implementation sequence

1. Create `tests/scrut/environment.md`
2. Create `tests/scrut/config-files.md`
3. Create `tests/scrut/flag-precedence.md`
4. Update `tests/scrut/config.md` with equals-syntax tests
5. Update `tests/scrut/errors.md` with separator and empty-value tests
6. Run `make test-scrut-update` to validate expectations against actual output
7. Adjust any tests where actual behavior differs from expectations
8. Run `make test-scrut` to confirm all tests pass
9. Run `make lint-md` to check Markdown formatting

## Key source files

- `cmd/root.go` - CLI flags, `initConfig()`, error paths
- `main.go` - Error format (`Error: %s\n`), exit codes
- `internal/config/config.go` - Config keys, defaults, env var names
- `tests/scrut/config.md` - Existing pattern for scrut test structure

## Verification

1. `make build` to build the binary
2. `make test-scrut` to run all scrut tests (should pass)
3. `make test` to verify Go unit tests still pass (no regressions)
4. `make lint-md` to verify Markdown formatting
