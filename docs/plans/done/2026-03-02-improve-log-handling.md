# Improve Log Handling

Addresses: #5 (log rotation), #13 (ring buffer memory leak)

## Context

The logger (`internal/logger/logger.go`) is a simple ring buffer with optional
file output. Two problems need fixing:

1. The log file grows indefinitely with no rotation or size limit.
2. The ring buffer trims with `l.entries = l.entries[1:]`, which re-slices
   without releasing the backing array, causing gradual memory growth.

## Changes

### 1. Fix ring buffer backing array leak (issue #13)

**File:** `internal/logger/logger.go` (lines 91-94)

Replace the current trim logic:

```go
l.entries = append(l.entries, entry)
if len(l.entries) > l.maxSize {
    l.entries = l.entries[1:]
}
```

With copy+overwrite that keeps the backing array bounded:

```go
if len(l.entries) < l.maxSize {
    l.entries = append(l.entries, entry)
} else {
    copy(l.entries, l.entries[1:])
    l.entries[l.maxSize-1] = entry
}
```

Also pre-allocate in `New()`: `make([]Entry, 0, maxEntries)`.

### 2. Introduce `Options` struct for `New()`

**File:** `internal/logger/logger.go`

Replace `New(logDir string, maxEntries int)` with `New(opts Options)`:

```go
type Options struct {
    LogDir     string
    MaxEntries int
    MaxSize    int64  // max file size in bytes; 0 = no rotation
    MaxFiles   int    // rotated backups to keep
}
```

Add fields to `Logger` struct: `filePath`, `maxBytes`, `maxFiles`.

This is an internal package with only two call sites, so the signature change
is safe.

### 3. Add size-based log rotation (issue #5)

**File:** `internal/logger/logger.go`

Add three private methods:

- `maybeRotate()`: called before each file write in `Log()`; stats the file,
  calls `rotateFiles()` if size reaches or exceeds `maxBytes`.
- `rotateFiles()`: closes current file, shifts backups
  (`.log.2` to `.log.3`, `.log.1` to `.log.2`, `.log` to `.log.1`),
  removes the oldest if at `maxFiles`, opens a fresh `.log`.
- `backupPath(n int) string`: returns the path for backup number n.

Rotation naming: `snappy.log` -> `snappy.log.1` -> `snappy.log.2` -> ...

Failures degrade gracefully (stderr warning, `l.file = nil`, ring buffer
continues).

### 4. Add config options

**File:** `internal/config/config.go`

New fields in `Config`:

| Field         | Viper key       | Env var                | Default   |
| ------------- | --------------- | ---------------------- | --------- |
| `LogMaxSize`  | `log_max_size`  | `SNAPPY_LOG_MAX_SIZE`  | `5242880` |
| `LogMaxFiles` | `log_max_files` | `SNAPPY_LOG_MAX_FILES` | `3`       |

Add to `SetDefaults()` and `Load()`.

### 5. Update call sites

**`cmd/root.go:88`**: Change `logger.New(cfg.LogDir, 50)` to use `Options`
struct, passing `cfg.LogMaxSize` and `cfg.LogMaxFiles`.

**`internal/tui/model_test.go:52`**: Change `logger.New("", 50)` to
`logger.New(logger.Options{MaxEntries: 50})`.

### 6. Update tests

**`internal/logger/logger_test.go`**:

- Update all existing `New()` calls to use `Options`.
- Add `TestRingBufferBackingArrayDoesNotGrow`: verifies `cap(l.entries)` stays
  constant after overflow.
- Add `TestRotationCreatesBackups`: small `MaxSize` triggers rotation, asserts
  `.log.1` exists.
- Add `TestRotationDeletesOldestBackup`: forces multiple rotations with
  `MaxFiles=2`, asserts `.log.3` does not exist.
- Add `TestNoRotationWhenMaxSizeZero`: confirms no `.log.1` when `MaxSize=0`.

**`internal/config/config_test.go`**:

- `TestLoadDefaults`: assert `LogMaxSize == 5*1024*1024`, `LogMaxFiles == 3`.
- `TestLoadEnvOverrides`: add subtests for `SNAPPY_LOG_MAX_SIZE` and
  `SNAPPY_LOG_MAX_FILES`.
- `TestLoadWithoutSetDefaults`: assert zero values for new fields.

### 7. Update README

**`README.md`**: Add `log_max_size` and `log_max_files` rows to the
configuration table.

## Files to modify

| File                             | Change                             |
| -------------------------------- | ---------------------------------- |
| `internal/logger/logger.go`      | Ring buffer fix, Options, rotation |
| `internal/logger/logger_test.go` | Update calls, add rotation tests   |
| `internal/config/config.go`      | New config fields and defaults     |
| `internal/config/config_test.go` | Test new config fields             |
| `cmd/root.go`                    | Update `logger.New()` call         |
| `internal/tui/model_test.go`     | Update `logger.New()` call         |
| `README.md`                      | Document new config options        |

## Commit strategy

1. `fix: use copy+reslice for ring buffer trimming (#13)` -- logger.go +
   logger_test.go
2. `feat: add size-based log file rotation (#5)` -- Options struct, rotation
   logic, config fields, call site updates, all remaining tests, README

## Verification

1. `make test` -- all existing and new tests pass
2. `make lint` -- no lint errors
3. Manual: run snappy, write entries, confirm rotation occurs at threshold by
   setting a small `SNAPPY_LOG_MAX_SIZE` (e.g., 1024) and checking for
   `snappy.log.1` in the log directory
