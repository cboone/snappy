# Add unit tests for config.Load() Viper integration

Resolves: #12

## Context

`internal/config/config.go` has minimal test coverage. The existing
`config_test.go` only tests `RefreshInterval` parsing (5 cases) and its
default. No tests exist for the other six Config fields, environment variable
overrides, or the full `SetDefaults()` + `Load()` integration. This plan adds
comprehensive tests to close those gaps.

## File to modify

- `internal/config/config_test.go`

## Tests to add

### 1. `TestLoadDefaults` -- verify all defaults after `SetDefaults()`

Call `viper.Reset()`, `SetDefaults()`, then `Load()`. Assert every field:

| Field                  | Expected default     |
|------------------------|----------------------|
| `RefreshInterval`      | `60s`                |
| `MountPoint`           | `"/"`                |
| `LogDir`               | `""`                 |
| `AutoEnabled`          | `true`               |
| `AutoSnapshotInterval` | `60s`                |
| `ThinAgeThreshold`     | `600s` (10 min)      |
| `ThinCadence`          | `300s` (5 min)       |

### 2. `TestLoadEnvOverrides` -- verify environment variable binding

For each field, set the corresponding `SNAPPY_*` env var via `t.Setenv`,
then call `Load()` and verify the override took effect. Requires calling
`viper.SetEnvPrefix("SNAPPY")` and `viper.AutomaticEnv()` in test setup
(mirroring `cmd/root.go`).

Subtests:

| Subtest                          | Env var                          | Value     | Expected                  |
|----------------------------------|----------------------------------|-----------|---------------------------|
| `mount override`                 | `SNAPPY_MOUNT`                   | `"/Volumes/Backup"` | `MountPoint == "/Volumes/Backup"` |
| `log_dir override`               | `SNAPPY_LOG_DIR`                 | `"/tmp/logs"` | `LogDir == "/tmp/logs"`      |
| `auto_enabled override`          | `SNAPPY_AUTO_ENABLED`            | `"false"` | `AutoEnabled == false`    |
| `refresh override`               | `SNAPPY_REFRESH`                 | `"120"`   | `RefreshInterval == 120s` |
| `auto_snapshot_interval override`| `SNAPPY_AUTO_SNAPSHOT_INTERVAL`  | `"30"`    | `AutoSnapshotInterval == 30s` |
| `thin_age_threshold override`    | `SNAPPY_THIN_AGE_THRESHOLD`      | `"900"`   | `ThinAgeThreshold == 900s`|
| `thin_cadence override`          | `SNAPPY_THIN_CADENCE`            | `"600"`   | `ThinCadence == 600s`     |

### 3. `TestLoadDurationFieldParsing` -- verify duration parsing for remaining fields

The existing `TestLoadRefreshIntervalParsing` covers `RefreshInterval` well.
Add an analogous table-driven test covering `AutoSnapshotInterval`,
`ThinAgeThreshold`, and `ThinCadence` with representative cases (duration
string, numeric string, integer, invalid fallback).

### 4. `TestLoadWithoutSetDefaults` -- verify behavior with no defaults

Call `viper.Reset()` (no `SetDefaults()`) then `Load()`. Verify that the
fallback values baked into `Load()` still apply for duration fields, and
that string/bool fields get zero values (`""`, `false`).

## Conventions to follow

- Standard library `testing` only (no testify)
- Table-driven tests with `t.Run()` subtests
- `viper.Reset()` at the start of each test/subtest for isolation
- `t.Setenv()` for env vars (auto-cleaned up by Go test framework)
- Error format: `t.Errorf("Field = %v, want %v", got, want)`

## Verification

```bash
make test
make lint
```
