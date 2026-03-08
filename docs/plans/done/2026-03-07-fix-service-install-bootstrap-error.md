# Fix `snappy service install` bootstrap error 125

## Context

`snappy service install` fails with:

```text
Error: launchctl bootstrap: Bootstrap failed: 125: Domain does not support specified action (exit status 125)
```

Error 125 indicates the service is already registered in the launchd domain.
The current `Install()` function calls `bootout()` before `bootstrap()` to handle
this, but the approach has two flaws:

1. **Ordering bug**: `bootout()` runs before the plist is written to disk.
   The legacy fallback inside `bootout()` checks `os.Stat(plistPath)` and skips
   the domain-target+plist approach if the file does not exist. On a reinstall
   where the plist was deleted but the service remains registered, this fallback
   is silently skipped.

2. **No recovery path**: `bootout()` errors are unconditionally ignored
   (`_ = bootout(...)`). If bootout fails for any reason other than "not loaded",
   the service stays registered and the subsequent `bootstrap` call fails with
   no retry.

## Changes

All changes are in two files:

- `internal/service/launchd.go`
- `internal/service/launchd_test.go`

### 1. Reorder `Install()`: write plist before bootout

Move plist generation and `os.WriteFile` to happen **before** the `bootout()` call.
This guarantees the legacy bootout fallback always has the plist file available.

Current order:

```text
bootout -> GeneratePlist -> MkdirAll -> WriteFile -> bootstrap
```

New order:

```text
GeneratePlist -> MkdirAll -> WriteFile -> bootout -> bootstrap
```

### 2. Add `isAlreadyBootstrappedError()` helper

New function to detect bootstrap errors that mean "service already registered":

```go
func isAlreadyBootstrappedError(output string) bool {
    lower := strings.ToLower(output)
    return strings.Contains(lower, "domain does not support specified action") ||
        strings.Contains(lower, "service is already loaded")
}
```

### 3. Extract `runBootstrap()` helper

Thin wrapper to avoid duplicating the `exec.Command` call between the initial
attempt and the retry:

```go
func runBootstrap(domain, plistPath string) ([]byte, error) {
    //nolint:gosec // arguments are controlled, not user input
    cmd := exec.Command("launchctl", "bootstrap", domain, plistPath)
    return cmd.CombinedOutput()
}
```

### 4. Add retry logic to `Install()`

If `bootstrap` fails with an already-bootstrapped error:

1. Call `bootout()` again (plist is now on disk, so legacy fallback works)
2. Call `launchctl enable` (handles edge case where service was disabled)
3. Retry `bootstrap` once

```go
out, err := runBootstrap(domainTarget(), plistPath)
if err == nil {
    return nil
}

outStr := strings.TrimSpace(string(out))
if !isAlreadyBootstrappedError(outStr) {
    return fmt.Errorf("launchctl bootstrap: %s (%w)", outStr, err)
}

// Service already registered. Bootout and retry once.
if bootoutErr := bootout(cfg.Label, plistPath); bootoutErr != nil {
    return fmt.Errorf(
        "launchctl bootstrap: service already loaded and bootout failed: %v (original: %s, %w)",
        bootoutErr, outStr, err,
    )
}

// Ensure the service is enabled (a disabled service cannot be bootstrapped).
enableCmd := exec.Command("launchctl", "enable", serviceTarget(cfg.Label))
enableOut, enableErr := enableCmd.CombinedOutput()
if enableErr != nil {
    return fmt.Errorf("launchctl enable: %s (%w)",
        strings.TrimSpace(string(enableOut)), enableErr)
}

retryOut, retryErr := runBootstrap(domainTarget(), plistPath)
if retryErr != nil {
    return fmt.Errorf("launchctl bootstrap (after bootout retry): %s (%w)",
        strings.TrimSpace(string(retryOut)), retryErr)
}
return nil
```

### 5. Add tests

Add to `internal/service/launchd_test.go`:

- `TestIsAlreadyBootstrappedError`: table-driven test covering known error strings
  and negative cases

## Verification

1. `make test` -- unit tests pass
2. `make test-scrut` -- CLI integration tests pass
3. `make lint` -- no new lint issues
4. Manual: `snappy service install` (fresh install, clean state)
5. Manual: `snappy service install` (reinstall, service already running)
6. Manual: `snappy service uninstall && snappy service install` (round-trip)
