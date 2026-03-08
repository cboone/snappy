# Fix `snappy service install` bootout exit 125 on disabled services

## Context

After the v0.8.3 release, `snappy service install` still fails when a previous
service registration is stuck in launchd:

```text
Error: launchctl bootstrap: service already bootstrapped and bootout failed:
  launchctl bootout: Boot-out failed: 125: Domain does not support specified action
  (exit status 125) (original: Bootstrap failed: 125: Domain does not support
  specified action, exit status 125)
```

The previous fix (2026-03-07) added retry logic: when `bootstrap` fails with
exit 125 ("already bootstrapped"), try `bootout` then `enable` then re-bootstrap.
However, the retry `bootout` itself also fails with exit 125. The code returns
an error on line 158 before ever reaching the `enable` step on line 164.

Root cause: when a launchd service is **disabled**, it is registered in the
domain (so `bootstrap` says "already bootstrapped") but cannot be booted out
(so `bootout` says "domain does not support specified action"). This creates a
deadlock that neither bootstrap nor bootout can break.

## Changes

All changes in two files:

- `internal/service/launchd.go`
- `internal/service/launchd_test.go`

### 1. Add `isDomainNotSupportedError()` helper

New function to detect the exit 125 "domain does not support specified action"
error, which appears in both bootstrap and bootout contexts with different
meanings:

```go
func isDomainNotSupportedError(output string) bool {
    return strings.Contains(strings.ToLower(output), "domain does not support specified action")
}
```

Refactor `isAlreadyBootstrappedError()` to call it:

```go
func isAlreadyBootstrappedError(output string) bool {
    return isDomainNotSupportedError(output) ||
        strings.Contains(strings.ToLower(output), "service is already loaded")
}
```

### 2. Add `launchctl unload` fallback to `bootout()`

After both modern (`bootout gui/<uid>/<label>`) and legacy
(`bootout gui/<uid> <plistPath>`) approaches fail, try the deprecated
`launchctl unload <plistPath>` as a last resort. This handles services loaded
via `launchctl load` or stuck in a disabled state where `bootout` refuses to
act. The `unload` subcommand is deprecated since macOS 10.10 but remains
functional through macOS 15+.

Add after the legacy bootout attempt and before the `isNotLoadedError` check:

```go
// Last resort: try deprecated "launchctl unload".
if _, statErr := os.Stat(plistPath); statErr == nil {
    unloadCmd := exec.Command("launchctl", "unload", plistPath)
    unloadOut, unloadErr := unloadCmd.CombinedOutput()
    if unloadErr == nil {
        return nil
    }
    outStr = strings.TrimSpace(string(unloadOut))
    err = unloadErr
}
```

### 3. Reorder `Install()`: enable before retry bootout

Move `launchctl enable` to before the retry bootout call. A disabled service
cannot be booted out; enabling it first breaks the deadlock.

Current retry flow (lines 155-177):

```text
bootout retry -> [if fails, return error] -> enable -> bootstrap retry
```

New retry flow:

```text
best-effort enable -> bootout retry -> [if fails, return error] -> hard enable -> bootstrap retry
```

The best-effort enable before bootout is safe: `launchctl enable` on an
already-enabled service is a no-op (returns exit 0). The hard enable before
bootstrap remains to catch the case where bootout cleared the registration
and the service needs re-enabling.

```go
// Enable first (best-effort). A disabled service resists bootout.
enableCmd := exec.Command("launchctl", "enable", serviceTarget(cfg.Label))
_ = enableCmd.Run()

if bootoutErr := bootout(cfg.Label, plistPath); bootoutErr != nil {
    return fmt.Errorf(...)
}

// Hard enable before re-bootstrap (in case bootout cleared state).
enableCmd2 := exec.Command("launchctl", "enable", serviceTarget(cfg.Label))
enableOut, enableErr := enableCmd2.CombinedOutput()
if enableErr != nil {
    return fmt.Errorf(...)
}

retryOut, retryErr := runBootstrap(domainTarget(), plistPath)
```

### 4. Add tests

- `TestIsDomainNotSupportedError`: table-driven test with positive and negative
  cases
- Verify existing `TestIsAlreadyBootstrappedError` still passes after refactor

## Verification

1. `make test` -- unit tests pass
2. `make test-scrut` -- CLI integration tests pass
3. `make lint` -- no new lint issues
4. Manual: `snappy service install` (fresh install, clean state)
5. Manual: `snappy service install` (reinstall, service already running)
6. Manual: `snappy service install` (reinstall, service disabled/stuck)
7. Manual: `snappy service uninstall && snappy service install` (round-trip)
