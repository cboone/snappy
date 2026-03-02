# Auto-snapshot creation and tiered retention

## Context

Snappy currently only creates and deletes snapshots on manual keypress. The user wants snapshots created automatically every 5 minutes, with old snapshots thinned so that density decreases as you go back in time. This is the same idea as Time Machine's own retention at larger scales (hourly for 24h, daily for a month, weekly thereafter), applied at a finer granularity for local snapshots.

## How tiered retention works

Snapshots are grouped by age into tiers. Within each tier, snapshots are thinned to a minimum gap. Newer tiers keep more detail; older tiers keep less.

| Age range      | Keep one every | ~Snapshots | Rationale                             |
| -------------- | -------------- | ---------- | ------------------------------------- |
| 0 - 1 hour     | 5 min (all)    | 12         | Fine-grained recovery for recent work |
| 1 - 6 hours    | 1 hour         | 5          | Moderate detail for the work session  |
| 6 - 24 hours   | 4 hours        | 5          | Enough to recover today's work        |
| 1 - 14 days    | 1 day          | 13         | Daily safety net                      |
| Older than 14d | Delete all     | 0          | macOS often purges these anyway       |
| **Total**      |                | **~35**    |                                       |

## Changes to `bin/snappy`

### 1. New constants

```bash
AUTO_SNAPSHOT_INTERVAL="${SNAPPY_AUTO_INTERVAL:-300}"  # 5 min, not readonly (toggleable)
AUTO_SNAPSHOT_ENABLED="${SNAPPY_AUTO_ENABLED:-true}"   # not readonly (toggleable)
```

Retention tiers as a constant array, each entry `"min_age:max_age:min_gap"` in seconds:

```bash
readonly -a RETENTION_TIERS=(
  "0:3600:300"          # 0-1h:   5 min gap
  "3600:21600:3600"     # 1-6h:   1h gap
  "21600:86400:14400"   # 6-24h:  4h gap
  "86400:1209600:86400" # 1-14d:  1d gap
)
readonly MAX_SNAPSHOT_AGE=1209600  # 14 days
```

### 2. New global state

```bash
declare -i LAST_AUTO_SNAPSHOT_EPOCH=0
```

### 3. New function: `snapshot_date_to_epoch()`

Extract the date-to-epoch conversion from `format_relative_time` into a reusable function (the same parsing logic at lines 466-477). `format_relative_time` then calls it internally, avoiding duplication.

### 4. New function: `tm_delete_snapshot()`

General-purpose deletion by date. The existing `tm_delete_oldest` is hardcoded to index 0; refactor it to call `tm_delete_snapshot`.

```bash
function tm_delete_snapshot() {
  local snapshot_date="${1}"
  local output
  if output=$(tmutil deletelocalsnapshots "${snapshot_date}" 2>&1); then
    return 0
  else
    log_event "ERROR" "Failed to delete snapshot ${snapshot_date}: ${output}"
    return 1
  fi
}
```

### 5. New function: `thin_snapshots()`

Algorithm:

1. Convert all `CURR_SNAPSHOTS` to `(date, epoch)` pairs
1. Mark all snapshots older than `MAX_SNAPSHOT_AGE` for deletion
1. For each tier, collect snapshots in that age range, walk chronologically, keep snapshots >= `min_gap` apart, mark the rest for deletion
1. Execute all deletions, logging each as a `THINNED` event

### 6. New function: `maybe_auto_snapshot()`

Called from `do_refresh`. Checks if `AUTO_SNAPSHOT_INTERVAL` seconds have elapsed since `LAST_AUTO_SNAPSHOT_EPOCH`. If so: create a snapshot, re-list snapshots (so thinning sees the new one), then run `thin_snapshots`. Advances the timer even on creation failure to avoid retrying every 60s refresh cycle.

### 7. Modified: `do_refresh()`

Call `maybe_auto_snapshot` after the standard refresh operations, before `draw_ui`.

### 8. Modified: `draw_header()`

Add auto-snapshot status line showing on/off state, interval, and countdown to next auto-snapshot.

### 9. Modified: `draw_controls()`

Add `[a]` toggle key for auto-snapshots.

### 10. Modified: `handle_keypress()`

Add `a`/`A` case to toggle `AUTO_SNAPSHOT_ENABLED`.

### 11. Modified: `draw_recent_log()`

Colorize `AUTO` events (cyan) and `THINNED` events (yellow).

### 12. Modified: `main()`

Update startup log to include auto-snapshot config.

## Verification

1. `bash -n bin/snappy` and `shellcheck bin/snappy` pass
1. Run snappy, verify auto-snapshot status line appears in header
1. Wait 5+ minutes, verify auto-creation fires (AUTO log event)
1. Verify thinning log events appear after auto-creation
1. Press `a` to toggle auto-snapshots off, verify header updates
1. Test with `SNAPPY_AUTO_INTERVAL=60` for faster feedback
1. Test with `SNAPPY_AUTO_ENABLED=false` to confirm it starts disabled
