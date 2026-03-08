# Guard DefaultLockPath against empty LogDir

Issue: #85

## Context

`DefaultLockPath(logDir)` in `internal/service/lock.go` calls `filepath.Join(logDir, "snappy.lock")`. When `logDir` is an empty string, this produces a relative path `"snappy.lock"` in the current working directory. This is surprising behavior and could cause lock acquisition to fail or target the wrong location.

In normal operation, `config.Load()` defaults `LogDir` to `~/.local/share/snappy`, but if `os.UserHomeDir()` fails, `LogDir` remains empty. The TUI already guards against empty `LogDir` in several places (`update.go:199`, `update.go:426`, `commands.go:178`), but `DefaultLockPath` itself does not.

## Approach

Add a fallback inside `DefaultLockPath`: when `logDir` is empty, use `os.TempDir()` instead. On macOS, `os.TempDir()` returns the user-specific `TMPDIR` value (e.g., `/var/folders/.../T/`), so the fallback is already per-user.

This is the most minimal, centralized fix: all four call sites (`cmd/root.go:186`, `cmd/run.go:40`, `internal/tui/update.go:409`, `internal/tui/update.go:432`) are automatically protected without any changes.

## Changes

### 1. `internal/service/lock.go` (DefaultLockPath)

Add an empty-string guard at the top of `DefaultLockPath`:

```go
func DefaultLockPath(logDir string) string {
    if logDir == "" {
        logDir = os.TempDir()
    }
    return filepath.Join(logDir, "snappy.lock")
}
```

This requires adding `"os"` to the import block (already imported for `os.OpenFile` etc., so no new dependency).

### 2. `internal/service/lock_test.go` (new test case)

Add a test verifying the empty-string fallback produces a path under `os.TempDir()`:

```go
func TestDefaultLockPathEmptyLogDir(t *testing.T) {
    got := DefaultLockPath("")
    want := filepath.Join(os.TempDir(), "snappy.lock")
    if got != want {
        t.Errorf("DefaultLockPath(\"\") = %q, want %q", got, want)
    }
}
```

## Verification

1. `make test` - unit tests pass, including the new test case
2. `make lint` - no lint issues introduced
3. Manual check: confirm `DefaultLockPath("")` returns an absolute path under `os.TempDir()`
