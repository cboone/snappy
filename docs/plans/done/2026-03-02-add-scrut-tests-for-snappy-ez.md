# 2026-03-02 Add Scrut Tests for snappy-ez

## Context

snappy-ez (`bin/snappy-ez`) is a standalone bash script that manages Time Machine
snapshots. It currently has no tests. The Go binary (`snappy`) has 8 scrut test
files in `tests/scrut/`, but none cover the shell script. Adding scrut tests will
catch regressions in log formatting, date conversion, dependency checks, and
startup behavior.

The main challenge is that snappy-ez runs an infinite loop with no CLI flags and
calls `tmutil` (which requires root). To test individual functions, we need to
source the script without triggering `main`. We also need to avoid `tmutil` calls
in unit-style tests.

## Changes

### 1. Make `bin/snappy-ez` sourceable

**File:** `bin/snappy-ez:283`

Replace the bare `main "${@}"` call with a `BASH_SOURCE` guard:

```bash
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "${@}"
fi
```

This is a standard shell best practice. The script behaves identically when
executed directly. When sourced (e.g., `source bin/snappy-ez`), it defines all
functions and constants without running the loop, enabling function-level testing.

### 2. Create scrut test files

All tests go in `tests/scrut/snappy-ez/` (subdirectory to separate from Go binary
tests). Tests use `${SNAPPY_EZ_BIN}` environment variable, mirroring the existing
`${SNAPPY_BIN}` pattern.

#### `tests/scrut/snappy-ez/log.md`

Tests the `log()` function output format `[YYYY-MM-DD HH:MM:SS] EVENT message`:

- Log with standard event type (STARTUP, ERROR, etc.)
- Log message content preserved exactly

#### `tests/scrut/snappy-ez/date-conversion.md`

Tests `snapshot_date_to_epoch()`:

- Valid snapshot date produces a non-zero epoch
- Invalid input returns `0`
- Empty string returns `0`

#### `tests/scrut/snappy-ez/dependency-checks.md`

Tests `require_macos()` and `require_tmutil()`:

- `require_macos` exits 0 on macOS (the test host)
- `require_tmutil` exits 0 when tmutil is present (the test host)

#### `tests/scrut/snappy-ez/startup.md`

Tests the startup log line by overriding `run_loop` after sourcing:

- STARTUP message includes interval, thin_age, and thin_cadence parameters
- Correct default values (60s, 600s, 300s)

#### `tests/scrut/snappy-ez/shutdown.md`

Tests the `cleanup()` trap handler:

- Produces SHUTDOWN log line with "Shutting down." message

#### `tests/scrut/snappy-ez/.markdownlint-cli2.jsonc`

Copy the existing config from `tests/scrut/.markdownlint-cli2.jsonc` (disables
MD014 for `$` prompts in scrut blocks).

### 3. Add Makefile targets

**File:** `Makefile`

Add these targets:

- `test-scrut-ez`: runs `SNAPPY_EZ_BIN="$(CURDIR)/bin/snappy-ez" scrut test tests/scrut/snappy-ez/`
- `test-scrut-ez-update`: runs scrut update for snappy-ez tests
- Update `test-all` to include `test-scrut-ez`
- Add `test-scrut-ez` to the `.PHONY` list

### 4. Update CI workflow (optional, low priority)

**File:** `.github/workflows/ci.yml`

Add `make test-scrut-ez` to the existing `test-scrut` job. No new dependencies
needed since snappy-ez is a tracked script, not a build artifact.

## Key patterns to reuse

- `${SNAPPY_BIN}` env var pattern from existing scrut tests (replicate as `${SNAPPY_EZ_BIN}`)
- `(glob)` matching for timestamps and dynamic values
- `{output_stream: stderr}` for error output testing
- `[1]` notation for non-zero exit codes
- `.markdownlint-cli2.jsonc` config for `$` prompt linting suppression

## Verification

1. Run `make test-scrut-ez` and confirm all tests pass
2. Run `bin/snappy-ez` directly and confirm it still works (BASH_SOURCE guard is transparent)
3. Run `make test-scrut` to confirm existing Go binary tests are unaffected
4. Run `make lint-md` to confirm new Markdown files pass linting
