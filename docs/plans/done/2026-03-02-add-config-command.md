# Add config command and config init subcommand

## Context

Snappy's Viper-backed configuration system is mature (env vars, config file,
defaults), but there is no CLI way to inspect the effective configuration or
bootstrap a config file. This plan adds `snappy config` to show effective
settings and `snappy config init` to create a default config file with
comments. Addresses issue #6. Issue #11 is already resolved (all three duration
fields are registered in `SetDefaults()` and covered by tests) and should be
closed.

## Step 1: Add helper functions to `internal/config/config.go`

Three new exported functions:

**`DefaultConfigPath() (string, error)`** -- returns
`~/.config/snappy/config.yaml`. Centralizes the path that is currently
duplicated between `cmd/root.go:57` and the `--config` flag help text.

**`WriteDefaultConfig(w io.Writer) error`** -- writes a commented YAML
template to `w` using `text/template`. The template embeds actual default
values from a struct mirroring `SetDefaults()`, keeping the generated file
in sync. Template content:

```yaml
# Snappy configuration
# See: https://github.com/cboone/snappy

# How often to refresh the snapshot list.
# Accepts Go duration strings (e.g., "60s", "2m") or plain seconds.
refresh: 60s

# Mount point to monitor for Time Machine snapshots.
mount: "/"

# Directory for log files. Leave empty for the default (~/.local/share/snappy).
log_dir: ""

# Maximum log file size in bytes before rotation (default: 5 MB).
log_max_size: 5242880

# Number of rotated backup log files to keep.
log_max_files: 3

# Whether to enable auto-snapshots at startup.
auto_enabled: true

# Interval between automatic snapshots.
auto_snapshot_interval: 60s

# Snapshots older than this threshold are candidates for thinning.
thin_age_threshold: 600s

# Minimum time gap to preserve between snapshots when thinning.
thin_cadence: 300s
```

**`FormatConfig(w io.Writer, cfg *Config, configFile string) error`** --
writes a human-readable display of the config file path and all effective
values. Output format:

```text
Config file: ~/.config/snappy/config.yaml

refresh: 1m0s
mount: /
log_dir:
log_max_size: 5242880
log_max_files: 3
auto_enabled: true
auto_snapshot_interval: 1m0s
thin_age_threshold: 10m0s
thin_cadence: 5m0s
```

Uses `text/template` for formatting. Duration values use Go's standard
`Duration.String()` format. The config file line shows the actual file in use
(from `viper.ConfigFileUsed()`) or "none" if no file was loaded.

## Step 2: Add tests to `internal/config/config_test.go`

Follow existing patterns: table-driven, `viper.Reset()`, standard library only.

- **`TestDefaultConfigPath`**: Verify path ends with
  `.config/snappy/config.yaml` and starts with the home directory.
- **`TestWriteDefaultConfig`**: Write to `bytes.Buffer`. Verify output contains
  all 9 Viper keys, contains comment lines (`#`), and is non-empty.
- **`TestFormatConfig`**: Given a `Config` with known values, verify output
  contains expected key-value pairs and the config file path line.

## Step 3: Create `cmd/config.go`

New file following `cmd/version.go` pattern. Self-registers via `init()`.

**`configCmd`** (parent):
- `Use: "config"`, `Short: "Show or manage snappy configuration"`
- `RunE: runConfigShow` -- calls `config.Load()` and `config.FormatConfig()`
  with `viper.ConfigFileUsed()` for the file path
- `Args: cobra.NoArgs`

**`configInitCmd`** (subcommand):
- `Use: "init"`, `Short: "Create a default config file"`
- `RunE: runConfigInit`
- `Args: cobra.NoArgs`

`runConfigInit` logic:
1. Determine target path: use `cfgFile` (from `--config` persistent flag) if
   set, otherwise `config.DefaultConfigPath()`
2. Check if file exists with `os.Stat` -- error if it does
3. Create parent directory with `os.MkdirAll(dir, 0o755)`
4. Create file, call `config.WriteDefaultConfig(f)`
5. Print `"Config file created: <path>\n"` to `cmd.OutOrStdout()`

## Step 4: Create `cmd/config_test.go`

Go unit tests for the command layer using `t.TempDir()` for isolation:

- **`TestConfigShow`**: Execute `configCmd` with captured output. Verify output
  contains all config keys and the "Config file:" header line.
- **`TestConfigInitCreatesFile`**: Point `--config` to temp path. Verify file
  is created, contains expected YAML content and comments, and stdout shows
  success message.
- **`TestConfigInitCreatesDirectory`**: Point to
  `<tmpdir>/new/subdir/config.yaml`. Verify directory and file are created.
- **`TestConfigInitErrorsOnExistingFile`**: Pre-create the target file. Verify
  error message contains "already exists".
- **`TestConfigInitUsesDefaultPath`**: With `cfgFile` unset, verify it attempts
  to use the default path (may need to mock or set HOME).

## Step 5: Refactor `cmd/root.go`

Replace the manual path construction in `initConfig()` (lines 55-60) with
`config.DefaultConfigPath()` to eliminate duplication:

```go
func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        defaultPath, err := config.DefaultConfigPath()
        if err == nil {
            viper.AddConfigPath(filepath.Dir(defaultPath))
            viper.SetConfigName("config")
            viper.SetConfigType("yaml")
        }
    }
    // ... rest unchanged
}
```

## Step 6: Create `tests/scrut/config-command.md`

New scrut test file for `config` and `config init`. Tests:

- **Config show displays current configuration**: `snappy config` outputs all
  keys with default values and "Config file:" header
- **Config show with env override**: `SNAPPY_MOUNT="/Volumes/Test" snappy config`
  shows the overridden value
- **Config init creates default file**: Use `--config` pointing to a temp path,
  verify success message
- **Config init fails when file exists**: Second invocation to the same path
  errors with "already exists"
- **Config init file contains comments**: Verify created file has `#` lines
- **Config help**: `snappy config --help` shows subcommand list with `init`
- **Config init help**: `snappy config init --help` shows usage

Temp file cleanup: use `rm -f` before the first test, then rely on sequential
ordering within the file for the "already exists" test.

## Step 7: Update scrut help output in existing files

Adding the `config` subcommand changes the `Available Commands` section in all
help output. Every scrut test capturing help output needs the new line:

```text
Available Commands:
  config      Show or manage snappy configuration
  help        Help about any command
  version     Print the version number of snappy
```

Files to update (same set that changed when `version` was added):

| File                             | Tests affected |
| -------------------------------- | -------------- |
| `tests/scrut/help.md`            | 4              |
| `tests/scrut/config.md`         | 2              |
| `tests/scrut/flag-precedence.md` | 4              |
| `tests/scrut/environment.md`     | 1              |

Approach: build, run `make test-scrut-update`, review diffs.

## Step 8: Close issue #11

Issue #11 ("make auto-snapshot interval, thin age, and thin cadence
configurable via Viper") is already resolved. `SetDefaults()` registers all
three keys, `Load()` reads them, and Go tests cover parsing, defaults, and env
overrides. Close the issue with a comment.

## Files

| File                               | Action | Description                              |
| ---------------------------------- | ------ | ---------------------------------------- |
| `internal/config/config.go`        | Modify | Add DefaultConfigPath, WriteDefaultConfig, FormatConfig |
| `internal/config/config_test.go`   | Modify | Add tests for new functions              |
| `cmd/config.go`                    | Create | config command and config init subcommand |
| `cmd/config_test.go`               | Create | Go unit tests for commands               |
| `cmd/root.go`                      | Modify | Use DefaultConfigPath in initConfig      |
| `tests/scrut/config-command.md`    | Create | Scrut tests for config and config init   |
| `tests/scrut/help.md`              | Modify | Add config to Available Commands         |
| `tests/scrut/config.md`            | Modify | Add config to Available Commands         |
| `tests/scrut/flag-precedence.md`   | Modify | Add config to Available Commands         |
| `tests/scrut/environment.md`       | Modify | Add config to Available Commands         |

## Verification

1. `make build` compiles successfully
2. `make test` passes (Go unit tests)
3. `make test-scrut` passes (CLI integration tests)
4. `make lint` and `make fmt` pass
5. Manual smoke tests:
   - `./bin/snappy config` shows effective config with file path
   - `./bin/snappy config init --config /tmp/test-config.yaml` creates file
   - `./bin/snappy config init --config /tmp/test-config.yaml` errors (exists)
   - `./bin/snappy config --help` shows init subcommand
   - `./bin/snappy --help` shows config in Available Commands
   - `cat /tmp/test-config.yaml` shows commented defaults
