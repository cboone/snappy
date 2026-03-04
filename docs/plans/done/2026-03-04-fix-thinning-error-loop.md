# 2026-03-04: Fix Thinning Error Loop

## Context

When auto-snapshot thinning fails to delete a snapshot (e.g., `ESTALE` from a stale file handle left by the Time Machine viewer), the TUI enters a tight loop: `handleThinResult()` triggers `doRefresh()`, which recomputes the same thin targets, which launches `doThinSnapshots()` again, which fails again. This repeats many times per second, flooding the log with errors and causing the screen to flash.

The underlying cause is that `tmutil deletelocalsnapshots` returns errno 70 (`ESTALE`, "Stale NFS file handle") when macOS holds a kernel-level reference on a snapshot. This is not predictable from APFS metadata; it only surfaces on deletion attempt. Investigation of the Time Machine UI interaction is tracked in issue #38.

## Root Cause

`handleThinResult()` unconditionally calls `doRefresh()` (update.go:310-311), regardless of whether thinning succeeded or failed. Since the failed snapshot remains in the list, `ComputeThinTargets()` deterministically returns the same targets, creating an immediate retry loop with no backoff.

## Fix: Three Parts

### Part 1: Break the tight loop

In `handleThinResult()`, only trigger an immediate refresh when at least one deletion succeeded (`msg.Deleted > 0`). When all deletions failed (`msg.Deleted == 0`), return nil and let the next regular tick (60s) handle the refresh.

### Part 2: Track pinned snapshots

Record which snapshot dates failed deletion in a `thinPinned map[string]struct{}` on the Model. These represent snapshots the system refuses to delete (stale handles, etc.). Filter them out in `handleRefreshResult()` before passing targets to `doThinSnapshots()`.

Clear `thinPinned` on:

- Manual refresh ('r' key): user explicitly asked for a fresh look
- Auto toggle on ('a' key): user re-enabled auto-snapshots
- Fully successful thin result (`msg.Err == nil`): conditions may have changed

### Part 3: Adjust thinning cadence around pinned snapshots

Pass pinned dates into `ComputeThinTargets()` so it treats them as "kept" snapshots. When the algorithm encounters a pinned snapshot during its walk, it updates `lastKeptTime` to that snapshot's time (since the snapshot will remain in place) rather than marking it for deletion. This preserves even spacing around undeletable snapshots instead of leaving gaps or clustering deletions.

Example: with 5-minute cadence and snapshots at :00, :01, :02, :05, :06, :07, :10, if :05 is pinned, the algorithm keeps :00, skips (keeps) :05 because it's pinned, and keeps :10. Without this adjustment, it would try to delete :05, fail, and the remaining distribution would be uneven.

## Files to Modify

### `internal/snapshot/auto.go`

Change `ComputeThinTargets` signature to accept pinned dates:

```go
func (a *AutoManager) ComputeThinTargets(snapshots []Snapshot, now time.Time, pinned map[string]struct{}) []string
```

In the walk loop, when a snapshot's date is in `pinned`, update `lastKeptTime` to its time and skip it (don't add to targets). This treats pinned snapshots as anchor points for cadence spacing.

### `internal/snapshot/auto_test.go`

Add tests for pinned snapshot behavior:

- Pinned snapshot is never included in targets
- Pinned snapshot resets cadence (snapshots after it are measured from its time)
- Empty/nil pinned map preserves existing behavior

### `internal/tui/messages.go`

Add `FailedDates []string` to `ThinResultMsg` so the handler knows which specific dates failed (currently only a formatted error string is returned).

### `internal/tui/commands.go`

In `doThinSnapshots()`, collect failed dates into a separate slice and populate `ThinResultMsg.FailedDates`. Keep the existing `Err` field for log display.

### `internal/tui/model.go`

Add `thinPinned map[string]struct{}` to the Model struct. Initialize with `make(map[string]struct{})` in `NewModel()`.

### `internal/tui/update.go`

Five changes:

1. **`handleThinResult()`**: Record `msg.FailedDates` into `m.thinPinned`. On full success, clear `thinPinned`. Only call `doRefresh()` when `msg.Deleted > 0`.

2. **`handleRefreshResult()`**: Pass `m.thinPinned` to `ComputeThinTargets()`. After computing targets, filter out any dates in `m.thinPinned` before dispatching `doThinSnapshots()` (belt and suspenders: `ComputeThinTargets` already skips them, but this ensures they're never passed to the delete function).

3. **`handleKey()` (refresh)**: Clear `m.thinPinned` when 'r' is pressed.

4. **`handleKey()` (auto toggle)**: Clear `m.thinPinned` when auto-snapshots are re-enabled.

5. **`handleTick()`**: No changes needed; the tick naturally triggers refresh which uses the updated `ComputeThinTargets`.

### `internal/tui/model_test.go`

Update existing tests and add new ones:

- Update `TestDoThinSnapshotsReportsDeleteFailures` to verify `FailedDates`
- Update `TestRefreshResultStartsSpinnerWhenThinning` to pass pinned arg
- `TestThinResultErrorNoRefresh`: complete failure returns nil cmd, records pinned dates
- `TestThinResultPartialSuccessRefreshes`: partial success triggers refresh and records pinned dates
- `TestThinPinnedDatesFilteredFromTargets`: refresh skips known-pinned dates
- `TestManualRefreshClearsThinPinned`: 'r' key clears the pinned set
- `TestAutoToggleOnClearsThinPinned`: toggling auto on clears the pinned set
- `TestSuccessfulThinClearsThinPinned`: full success clears the pinned set

### `cmd/helpers.go` and `cmd/thin.go` and `cmd/run.go`

Update call sites for `ComputeThinTargets` to pass `nil` for pinned (the non-interactive commands don't track pinned state across invocations).

### `bin/snappy-ez`

In `thin_snapshots()`, replace the `if tmutil ...; then ... else ... fi` pattern with exit code capture so ESTALE (exit code 70) is handled distinctly. Currently (lines 236-241) all failures log as "ERROR". Change to:

```bash
local -i rc=0
tmutil deletelocalsnapshots "${snap_to_del}" > /dev/null 2>&1 || rc=$?
if ((rc == 0)); then
  log "THIN" "Deleted old snapshot: ${snap_to_del}"
elif ((rc == 70)); then
  log "THIN" "Skipped pinned snapshot: ${snap_to_del} (stale handle)"
else
  log "ERROR" "Failed to delete snapshot: ${snap_to_del} (exit ${rc})"
fi
```

This keeps the script running normally; it just treats ESTALE as a non-error skip.

### `tests/scrut/snappy-ez/`

Add or update scrut tests to cover the ESTALE handling in `thin_snapshots`.

## Verification

1. `make test` to run all unit tests including the new ones
2. `make lint` to verify code quality
3. Manual testing: `make build`, run the TUI, enable auto-snapshot, and verify that a failed `tmutil deletelocalsnapshots` does not cause rapid error loops and that thinning adapts its spacing around the pinned snapshot
