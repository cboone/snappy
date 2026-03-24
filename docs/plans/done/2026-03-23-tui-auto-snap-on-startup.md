# 2026-03-23 TUI auto-snap on startup (#106)

## Context

When TUI auto-snapshotting is enabled, the first snapshot waits one full interval (default 60s) because `NewAutoManager` sets `lastAutoTime = now`. The service (`snappy run`) creates a snapshot immediately on startup via `runIteration`. The TUI should behave the same way, while guarding against double-snapshotting if a recent snapshot already exists.

## Approach

Trigger a startup snapshot from `handleRefreshResult` after the first successful refresh. At that point the snapshot list is available, so we can check the age of the newest snapshot to avoid double-snapshotting.

## Changes

### 1. Add `maybeStartupSnapshot` method to `internal/tui/update.go`

New private method on `*Model`:

```go
func (m *Model) maybeStartupSnapshot(cmds []tea.Cmd) []tea.Cmd
```

Logic:

1. Return early if `!m.auto.Enabled()` or `m.snapshotting`
2. If snapshots exist, check the newest one (last element, sorted ascending):
   - If `age < m.auto.Interval()`: skip creation, but call `m.auto.RecordSnapshot(newest.Time)` to align the timer so the next auto-snapshot fires relative to the last actual snapshot, not TUI startup. Return.
3. If no snapshots or newest is stale: set `m.snapshotting`, `m.autoSnapshotting`, `m.loading` to true. Call `m.auto.RecordSnapshot(now)`. Log "Creating startup auto-snapshot..." under `logger.CatAuto`. Append `doAutoCreateSnapshot(...)` and `m.spinner.Tick` to cmds.

### 2. Call it from `handleRefreshResult` in `internal/tui/update.go`

Two insertions:

**Before `m.logDiffChanges()`** (~line 752): capture `isFirstRefresh := !m.hadFirstRefresh`

**After `var cmds []tea.Cmd`** (~line 758): add:

```go
if isFirstRefresh {
    cmds = m.maybeStartupSnapshot(cmds)
}
```

This placement ensures `m.snapshots` is populated, diff logging has run, and the startup snapshot check only fires once.

### 3. Add tests to `internal/tui/model_test.go`

| Test | Scenario | Key assertion |
| ------ | ---------- | --------------- |
| `TestFirstRefreshTriggersStartupSnapshot` | Newest snapshot older than interval | `snapshotting == true`, log contains "startup auto-snapshot" |
| `TestFirstRefreshSkipsStartupSnapshotWhenRecent` | Newest snapshot younger than interval | `snapshotting == false`, timer aligned (~40s remaining) |
| `TestFirstRefreshTriggersStartupSnapshotWhenEmpty` | No existing snapshots | `snapshotting == true` |
| `TestFirstRefreshNoStartupSnapshotWhenAutoDisabled` | Auto-snap disabled | `snapshotting == false` |
| `TestSecondRefreshDoesNotTriggerStartupSnapshot` | `hadFirstRefresh = true` | No startup snapshot on second refresh |
| `TestFirstRefreshStartupSnapshotAlignsTimer` | Recent snapshot (20s old, 60s interval) | `NextIn(now) == 40s` |
| `TestFirstRefreshStartupSnapshotSkippedWhenDaemonActive` | `DaemonActive: true` | `snapshotting == false` |

## Files modified

- `internal/tui/update.go`: new `maybeStartupSnapshot` method + two-line insertion in `handleRefreshResult`
- `internal/tui/model_test.go`: 7 new test functions

## Files unchanged

- `internal/snapshot/auto.go`: no new methods needed; reuse `RecordSnapshot`, `Enabled`, `Interval`
- `internal/tui/model.go`: no new fields; reuse `hadFirstRefresh`
- `internal/tui/commands.go`: reuse `doAutoCreateSnapshot` as-is
- `internal/tui/messages.go`: no new message types

## Edge cases

- **Snapshot error on first refresh**: `handleRefreshResult` returns early at line 743, `hadFirstRefresh` stays false, so the next successful refresh still triggers the startup check.
- **User presses 's' before first refresh completes**: `m.snapshotting` guard in `maybeStartupSnapshot` prevents a duplicate.
- **Service installed or daemon active**: Auto-snap is disabled in `NewModel`, so `maybeStartupSnapshot` returns immediately.

## Verification

```bash
make test          # All unit tests pass, including the 7 new ones
make test-all      # Full suite including scrut CLI tests
make lint          # No lint issues
```

Manual: launch `snappy` with auto-snap enabled. The log panel should show "Creating startup auto-snapshot..." within a few seconds of startup (after the first refresh), rather than waiting 60s.
