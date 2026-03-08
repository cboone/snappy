# TUI Persistent Locking (Issue #87)

## Context

Multiple TUI instances can run simultaneously with no mutual exclusion. Both instances show identical green "on" status, both independently create snapshots and thin, with no awareness of each other. The daemon already holds a persistent flock for its entire lifetime, but the TUI only acquires the lock transiently (per auto-snapshot) or not at all (manual snapshots). This plan adds a persistent lock to the TUI so that the second instance detects the first and defers, using the same mechanism that already works for daemon-to-TUI coordination.

## Approach

Have the TUI acquire a persistent flock when auto-snapshots are enabled and release it when disabled or on quit. Other TUI instances (and `syncDaemonState`) detect this lock via `IsHeld` and disable their own auto-snapshots, exactly as they already do for the daemon.

### Key design decisions

- **`lock *service.LockFile` pointer on Model**: Pointer survives Bubbletea's value-copy semantics (same pattern as `*config.Config`, `*logger.Logger`, `*snapshot.AutoManager`).
- **Keep `daemonActive` field name**: Its semantics generalize to "external process holds the auto-snapshot lock." Renaming would cause unnecessary churn.
- **Keep "service" label in view**: The daemon is by far the most common external lock holder. Log messages provide enough context ("Another snappy process detected").
- **Manual snapshots unchanged**: User-initiated snapshots (pressing 's') are intentional and harmless to duplicate. No lock gating.
- **flock is per-fd, not per-process**: When the TUI holds a persistent lock on one fd, `doAutoCreateSnapshot` cannot acquire on a different fd from a goroutine. Solved by passing a `holdingLock` flag to skip transient acquisition.

## Changes

### 1. `internal/tui/model.go`

- Add `lock *service.LockFile` field to `Model` struct
- Add `Lock *service.LockFile` to `ModelParams`
- In `NewModel`, store `params.Lock` into `m.lock`

### 2. `cmd/root.go`

- After `daemonActive := service.IsHeld(lockPath)`, if `!daemonActive && cfg.AutoEnabled`, call `service.Acquire(lockPath)`
  - On `ErrLocked`: set `daemonActive = true` (race between IsHeld and Acquire)
  - On other error: log warning, proceed without lock
  - On success: store `*LockFile`
- Pass `Lock` in `ModelParams`
- After `p.Run()` returns, release the lock if non-nil
- Add `"errors"` import

### 3. `internal/tui/update.go`

**`syncDaemonState`** (lines 425-449):

- Add early return when `m.lock != nil` (TUI holds the lock, skip self-detection)
- Keep existing `m.autoSnapshotting` guard as secondary safety
- Update log message from "Background service detected" to "Another snappy process detected" for accuracy

**'a' key handler** (lines 175-196):

- On enable: acquire lock via `service.Acquire`. On `ErrLocked`, set `daemonActive = true` and log. On success, store in `m.lock`, then toggle auto on.
- On disable: toggle auto off, then release `m.lock` and set to nil.
- Guard: if `m.cfg.LogDir == ""`, skip lock operations (no lock path available).

**Quit handler** (lines 208-211):

- Release `m.lock` if non-nil, set to nil before `tea.Quit`.

**`handleTick`** (lines 395-423):

- Pass `m.lock != nil` as `holdingLock` argument to `doAutoCreateSnapshot` (line 410).

- Add `"errors"` import

### 4. `internal/tui/commands.go`

**`doAutoCreateSnapshot`** (lines 99-117):

- Add `holdingLock bool` parameter
- When `holdingLock` is true, skip transient `Acquire`/`Release` and create snapshot directly
- When false, keep existing transient lock behavior (for edge cases where TUI runs without persistent lock)

### 5. `internal/tui/view.go`

No structural changes. The existing `daemonActive` path in `formatAutoStatus` (showing "service") and `buildDotIndicator` (green dot) already handle this state correctly.

### 6. `internal/tui/model_test.go`

New tests:

- **`TestRefreshTickIgnoresPersistentTUILock`**: TUI holds persistent lock, `syncDaemonState` should not set `daemonActive`
- **`TestAutoToggleAcquiresAndReleasesLock`**: Press 'a' to enable acquires lock (verified via `IsHeld`); press 'a' again releases it
- **`TestAutoToggleFailsWhenExternalLockHeld`**: External process holds lock, press 'a' gets `ErrLocked`, sets `daemonActive`
- **`TestQuitReleasesLock`**: Press 'q' releases the persistent lock
- **`TestStartupAcquiresLockWhenAutoEnabled`**: Verify `cmd/root.go` flow via model: when `Lock` is passed in `ModelParams`, model stores it

Update existing tests:

- **`TestRefreshTickDisablesAutoWhenDaemonLockAppears`**: Update log message assertion if wording changes
- **`TestRefreshTickClearsDaemonActiveWhenLockReleased`**: Same

### 7. Existing functions reused (no new abstractions)

- `service.Acquire(path)` at `internal/service/lock.go:31`
- `service.IsHeld(path)` at `internal/service/lock.go:72`
- `(*LockFile).Release()` at `internal/service/lock.go:53`
- `service.DefaultLockPath(logDir)` at `internal/service/lock.go:24`
- `service.ErrLocked` at `internal/service/lock.go:14`

## Edge cases

- **Race between `IsHeld` and `Acquire` at startup**: Caught by handling `ErrLocked` from `Acquire` as fallback.
- **Crash/kill**: OS releases flock automatically. Next instance acquires cleanly.
- **`LogDir` empty**: Lock operations skipped (guarded in `syncDaemonState` and 'a' handler).
- **Quit during in-flight auto-snapshot**: Quit handler releases persistent lock. Goroutine has no lock of its own (`holdingLock = true` means it skipped transient acquire). Goroutine's `SnapshotCreatedMsg` arrives after quit and is dropped by Bubbletea.
- **Value semantics**: `*service.LockFile` is a pointer, shared across Bubbletea's value copies within one Update cycle. Mutations to `m.lock` are reflected in the returned model.

## Verification

1. `make build` compiles successfully
2. `make test` passes all existing and new unit tests
3. `make lint` passes
4. Manual test: launch two TUI instances simultaneously
   - First instance: green dot, "on" status, creates auto-snapshots
   - Second instance: green dot, "service" status, auto-snapshots disabled, log says "Another snappy process detected"
   - Quit first instance, press 'a' in second: acquires lock, auto-snapshots resume
5. Manual test: `snappy run` daemon + TUI: existing behavior preserved (TUI shows "service")
