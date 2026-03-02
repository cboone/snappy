# Interactive Snapshot Manager PoC

## Context

Snappy aims to increase the frequency of Time Machine local snapshots beyond the default once-per-day schedule. Before building a full tool, we need a proof-of-concept to answer key questions: does `tmutil localsnapshot` work without Time Machine configured? How and when does macOS automatically purge "extra" snapshots?

This plan covers the first iteration: a single interactive bash script that lets you manually trigger snapshots, watch the snapshot list refresh in real time, and log everything to observe the system's cleanup behavior over time.

### Key findings from exploration

- **No sudo required.** All relevant `tmutil` commands work without elevated privileges.
- **No TM configuration required.** `tmutil localsnapshot` creates snapshots even with no backup destination configured.
- **Snapshots are purgeable.** tmutil itself warns: "local snapshots are considered purgeable and may be removed at any time by deleted(8)." The `deleted` daemon manages purgeable space system-wide and will remove snapshots under disk pressure.
- **Listing formats:**
  - `tmutil listlocalsnapshotdates /` outputs dates as `YYYY-MM-DD-HHMMSS`
  - `tmutil listlocalsnapshots /` outputs full names like `com.apple.TimeMachine.YYYY-MM-DD-HHMMSS.local`
- **Deletion works too.** `tmutil deletelocalsnapshots <date>` works without sudo.

## Approach

Create a single interactive bash script at `bin/snappy` with a terminal UI.

### User interaction model

The script runs in a loop, auto-refreshing the snapshot list every N seconds (default 60). Between refreshes, the user can press a key to trigger actions immediately. No sudo, no background daemons, no external dependencies beyond `tmutil`, `df`, and `tput`.

**Controls:**

- `s` -- create a new snapshot
- `r` -- force-refresh the snapshot list
- `d` -- delete the oldest snapshot (for cleanup during testing)
- `q` -- quit

### Terminal display layout

```text
================================================================================
  SNAPPY v0.1.0 -- Time Machine Local Snapshot Manager
================================================================================
  Volume: /    |  Refresh: 60s  |  Last: 2026-02-28T18:45:30
  Time Machine: Not configured (snapshots work regardless)
  Disk: 7.3Ti total, 11Gi used, 7.0Ti available (1%)
================================================================================

  LOCAL SNAPSHOTS (3)                                    [+1 added, 0 removed]
  ------------------------------------------------------------------------------
   1.  2026-02-28-190000   (3m ago)
   2.  2026-02-28-183000   (33m ago)
   3.  2026-02-28-181500   (48m ago)

================================================================================
  RECENT LOG
================================================================================
  [18:50:00] CREATED  Snapshot created: 2026-02-28-190000
  [18:45:30] INFO     Refresh: 2 snapshots, disk 1% used
  [18:45:00] STARTUP  snappy v0.1.0 | volume=/ | refresh=60s

================================================================================
  [s] Snapshot   [r] Refresh   [d] Delete oldest   [q] Quit
================================================================================
```

When no snapshots exist, the list section shows: `(none -- press 's' to create the first snapshot)`

### Logging

Log to both terminal (colored) and a persistent file (`~/.local/share/snappy/snappy.log`, plain text).

Format: `[HH:MM:SS] TYPE    message`

Event types:

- `STARTUP` -- script started, with configuration summary
- `INFO` -- periodic refresh summaries, disk space
- `CREATED` -- snapshot successfully created
- `ADDED` -- snapshot appeared that wasn't in the previous list
- `REMOVED` -- snapshot disappeared between refreshes (the key signal for observing cleanup)
- `ERROR` -- any tmutil command failure

The `ADDED`/`REMOVED` tracking is the core observability feature: by diffing the snapshot list between refresh cycles, we can detect when macOS purges snapshots and correlate that with disk pressure or time elapsed.

## Implementation

### File: `bin/snappy`

Single executable bash script. Structure from top to bottom:

1. **Shebang and header:** `#!/usr/bin/env bash`, set -euo pipefail, TRACE support
2. **Constants:** version, exit codes, defaults for refresh interval / mount point / log dir (all overridable via environment variables)
3. **Color setup:** `setup_colors()` -- detect terminal color support, set ANSI globals (degrades to no-op strings if not a TTY)
4. **Dependency check:** `check_dependencies()` -- verify `tmutil` exists (fatal if not), warn if `tput` missing
5. **Log directory setup:** `setup_log_dir()` -- `mkdir -p`, disable file logging on failure
6. **Cleanup trap:** `cleanup()` -- restore cursor, reset terminal, registered on EXIT/INT/TERM
7. **tmutil wrappers:**
   - `tm_check_status()` -- run `tmutil destinationinfo`, set status text
   - `tm_list_snapshots()` -- run `tmutil listlocalsnapshotdates /`, populate `CURR_SNAPSHOTS` array via `readarray -t`
   - `tm_create_snapshot()` -- run `tmutil localsnapshot`, capture output, log result
   - `tm_delete_oldest()` -- delete the oldest entry from `CURR_SNAPSHOTS`, log result
   - `tm_get_disk_info()` -- parse `df -h /` output into a summary string
8. **Snapshot diffing:**
   - `compute_snapshot_diff()` -- compare `PREV_SNAPSHOTS` and `CURR_SNAPSHOTS` using associative arrays, log additions/removals
   - `update_snapshot_state()` -- copy current to previous, update counts, call diff
9. **Display functions:**
   - `draw_header()` -- title bar, volume/TM/disk info
   - `draw_snapshot_list()` -- numbered list (newest first) with relative timestamps, truncated if list exceeds terminal height
   - `draw_recent_log()` -- tail of log file, filling remaining space
   - `draw_controls()` -- key legend at bottom
   - `draw_ui()` -- clear screen, call all draw functions in sequence
10. **Input handling:** `handle_keypress()` -- case statement dispatching to actions
11. **Main loop:** `run_main_loop()` -- refresh, draw, `read -r -s -n 1 -t $interval`, handle input, repeat
12. **Entry point:** `main()` -- validate env, setup, detect TM status, initial refresh, hide cursor, enter loop

### Key implementation details

- **`read -t` as event loop:** `read -r -s -n 1 -t "${REFRESH_INTERVAL}"` blocks for up to N seconds, returning immediately on any keypress. Timeout returns non-zero, handled with `|| true` to avoid tripping `set -e`.
- **Snapshot diffing:** Uses `declare -A` (bash associative array) to build a set from `PREV_SNAPSHOTS`, then iterates `CURR_SNAPSHOTS` to find additions and `PREV_SNAPSHOTS` to find removals. O(n) comparison.
- **Relative timestamps:** Parse the `YYYY-MM-DD-HHMMSS` format with `date -j -f` to get epoch seconds, compute delta from now, format as "Xm ago" / "Xh ago" / "Xd ago".
- **Screen redraw:** Full `clear` + reprint on each cycle. Acceptable for a 60s refresh interval. Cursor hidden during the session (`tput civis`), restored in cleanup (`tput cnorm`).
- **SIGWINCH:** Trap on terminal resize to trigger an immediate redraw.
- **No subshells in the main loop:** Use `readarray -t CURR_SNAPSHOTS < <(...)` (process substitution) instead of pipes, so array assignments persist in the loop's scope.

### Environment variables

| Variable         | Default                 | Purpose                                 |
| ---------------- | ----------------------- | --------------------------------------- |
| `SNAPPY_REFRESH` | `60`                    | Seconds between auto-refresh cycles     |
| `SNAPPY_MOUNT`   | `/`                     | Volume mount point to query             |
| `SNAPPY_LOG_DIR` | `~/.local/share/snappy` | Log file directory                      |
| `TRACE`          | (unset)                 | Set to any value to enable bash tracing |

### File: `.shellcheckrc`

```ini
enable=all
```

### File: `README.md`

Update the Usage section with invocation instructions and environment variable documentation.

## Verification

1. **Syntax check:** `bash -n bin/snappy`
2. **Lint:** `shellcheck bin/snappy` (should pass clean)
3. **Run it:** Launch `bin/snappy`, verify the display renders correctly
4. **Create a snapshot:** Press `s`, confirm the snapshot appears in the list on the next refresh (or press `r` to see it immediately)
5. **Observe diffing:** Create a second snapshot, confirm the `ADDED` log entry appears
6. **Delete oldest:** Press `d`, confirm the `REMOVED` log entry appears on next refresh
7. **Check log file:** `cat ~/.local/share/snappy/snappy.log` should show all events with timestamps
8. **Test degradation:** Rename `tmutil` temporarily (or run on a non-macOS system) to verify the dependency check fires
9. **Leave it running:** Let it run for an extended period to observe whether macOS automatically purges any snapshots (the core research question)
