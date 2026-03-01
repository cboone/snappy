# Delegate snapshot retention to macOS

## Context

Overnight analysis proved that macOS's `deleted` daemon already thins snapshots
to one per hour at the 60-minute mark and maintains hourly retention for 24+
hours. Snappy's thinning logic duplicates (and sometimes conflicts with) this
built-in behavior. Snappy should focus solely on creating frequent snapshots
within the first hour and let macOS handle all cleanup.

## Changes

All changes are in `bin/snappy`.

### 1. Remove deletion and thinning

Delete entirely:

- `RETENTION_TIERS` and `MAX_SNAPSHOT_AGE` constants (lines 32-38)
- `tm_delete_snapshot()` function (lines 260-276)
- `tm_delete_oldest()` function (lines 278-294)
- The "Tiered retention" section including `thin_snapshots()` (lines 478-574)

### 2. Two-phase auto-snapshot interval

Replace the single `AUTO_SNAPSHOT_INTERVAL` (currently 300s) with a two-phase
approach:

- Every 1 minute for the first 10 minutes after startup (or after toggling on)
- Every 5 minutes thereafter

Add a new constant for the initial interval (60s) and a threshold (600s). In
`maybe_auto_snapshot()`, compute elapsed time since `LAST_AUTO_SNAPSHOT_EPOCH`
was first set (i.e., since auto-snapshots started) and pick the appropriate
interval. Replace the single `AUTO_SNAPSHOT_INTERVAL` with
`AUTO_SNAPSHOT_INTERVAL_INITIAL` (60s) and `AUTO_SNAPSHOT_INTERVAL_NORMAL`
(300s), plus `AUTO_SNAPSHOT_RAMPUP` (600s) for the threshold.

Track when auto-snapshots were first activated via a new global
`AUTO_SNAPSHOT_START_EPOCH`. Set it at startup (line ~978) and when toggling
on (in `handle_keypress`). In `maybe_auto_snapshot()`, choose the interval
based on `now - AUTO_SNAPSHOT_START_EPOCH`.

### 3. Simplify `maybe_auto_snapshot()`

Remove the thinning calls and double re-list (lines 601-607). Keep a single
re-list after snapshot creation. Update the function comment to remove mention
of thinning. Add the two-phase interval logic from step 2.

### 4. Remove delete keybinding

- Remove `d|D` case from `handle_keypress()` (lines 872-876)
- Remove `[d] Delete oldest` from `draw_controls()` (line 831)

### 5. Clean up THINNED references

- Remove `THINNED` from `log_event()` docstring (line 184)
- Remove `THINNED` check from `draw_recent_log()` colorization (line 814)

### 6. Bookend snapshot list display

Rewrite `draw_snapshot_list()` to always show two newest and two oldest:

- Extract the per-snapshot rendering (lines 755-778) into a helper function
  `_draw_snapshot_line(index, count)`
- When count <= 4: show all snapshots newest-first
- When count > 4: show 2 newest, "... and N more ...", 2 oldest
- Remove the terminal-height truncation logic (`max_display`)

## Verification

1. `bash -n bin/snappy` (syntax check)
2. `shellcheck bin/snappy` (lint)
3. Manual run: confirm `[d]` is gone, auto-snapshots fire at 1-minute intervals
   initially then 5-minute intervals after 10 minutes, no thinning log events,
   and the snapshot list shows bookend display with 5+ snapshots
