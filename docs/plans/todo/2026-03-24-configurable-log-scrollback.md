# 2026-03-24 Add configurable log scrollback (issue #104)

## Context

The TUI log ring buffer is hard-coded to 50 entries (`cmd/root.go:152`). Issue #104 asks for more log scrollback. The fix: make the number of displayed log entries configurable via the existing config system and raise the default to 200.

The `logger.Options.MaxEntries` field and ring buffer logic already handle arbitrary sizes, so no logger changes are needed. `LoadTail()` also uses `maxSize`, so it automatically benefits.

## Naming

The config variable is named `log_scrollback` (Go field: `LogScrollback`) to make clear this controls how many log entries the TUI displays, not an internal logger limit. The config comment, logger option doc comment, and all related descriptions reinforce this framing.

## Changes

### 1. `internal/config/config.go`

- Add `defaultLogScrollback = 200` constant (alongside `defaultLogMaxSize`, `defaultLogMaxFiles`)
- Add `LogScrollback int` field to the `Config` struct (after `LogMaxFiles`)
- Add `LogScrollback: viper.GetInt("log_scrollback")` in `Load()`
- Add `viper.SetDefault("log_scrollback", defaultLogScrollback)` in `SetDefaults()`
- Add config template entry in `defaultConfigTmpl`:
  ```yaml
  # Number of log entries to display in the TUI scrollback.
  log_scrollback: {{.LogScrollback}}
  ```
- Add `LogScrollback int` + value to `WriteDefaultConfig()` template data struct
- Add `log_scrollback: {{.LogScrollback}}` to `formatConfigTmpl`
- Add `LogScrollback int` + value to `FormatConfig()` template data struct

### 2. `cmd/root.go` (line 152)

Replace `MaxEntries: 50` with `MaxEntries: cfg.LogScrollback`.

### 3. `internal/config/config_test.go`

- `TestLoadDefaults`: assert `LogScrollback == 200`
- `TestLoadEnvOverrides`: add case for `SNAPPY_LOG_SCROLLBACK=500`
- `TestWriteDefaultConfig`: add `"log_scrollback:"` to expected keys
- `TestFormatConfig`: add `LogScrollback: 200` to test Config, `"log_scrollback: 200"` to expected
- `TestFormatConfigNoFile`: add `LogScrollback: 200` to Config struct
- `TestLoadWithoutSetDefaults`: assert `LogScrollback == 0`

### 4. `cmd/config_test.go`

- `TestConfigShow`: add `"log_scrollback:"` to expected keys

### 5. `tests/scrut/config-command.md`

- "Config show" test: add `log_scrollback: 200` after `log_max_files: 3`
- "Config init file contains comments": update count from `11` to `12`

## Files to modify

- `internal/config/config.go`
- `cmd/root.go`
- `internal/config/config_test.go`
- `cmd/config_test.go`
- `tests/scrut/config-command.md`

## Verification

```bash
make test          # Go unit tests pass
make test-scrut    # Scrut CLI tests pass
make lint          # No lint issues
make build         # Builds cleanly
```
