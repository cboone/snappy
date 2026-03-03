# Add Non-Interactive CLI Commands

## Context

Snappy's TUI provides snapshot creation, listing, status display, auto-snapshot
looping, and thinning, but none of these are accessible without launching the
interactive interface. This makes it impossible to script Snappy, integrate it
with launchd/cron, or use it in headless environments. Adding non-interactive
commands achieves full CLI parity with the TUI.

## Commands

Five new flat commands, matching the user's preference:

| Command         | Description                                     |
| --------------- | ----------------------------------------------- |
| `snappy create` | Create a new local Time Machine snapshot        |
| `snappy list`   | List local snapshots with details               |
| `snappy status` | Show Time Machine and disk status               |
| `snappy thin`   | Thin old snapshots based on configured cadence   |
| `snappy run`    | Run the auto-snapshot loop (foreground daemon)   |

All commands inherit the existing `--config` persistent flag. A new `--json`
persistent flag on rootCmd enables machine-readable output for `create`, `list`,
`status`, and `thin`. The `run` command ignores `--json` since it emits
streaming log lines.

## Output Examples

### `snappy create`

```text
Snapshot created: 2026-03-03-142530
```

```json
{"date":"2026-03-03-142530"}
```

### `snappy list`

```text
3 snapshot(s) on /

  1. 2026-03-03-142530   (3m ago)    AB12...   purgeable
  2. 2026-03-03-141500   (13m ago)   CD34...   purgeable
  3. 2026-03-03-140000   (28m ago)   EF56...   pinned   limits shrink
```

```json
{
  "mount": "/",
  "count": 3,
  "snapshots": [
    {
      "date": "2026-03-03-142530",
      "relative": "3m ago",
      "uuid": "AB12...",
      "purgeable": true,
      "limits_shrink": false
    }
  ]
}
```

### `snappy status`

```text
Time Machine: Configured
Mount: /
APFS volume: disk3s5
Disk: 466Gi total, 280Gi used, 186Gi available (60%)
Snapshots: 3 local, 2 other
Auto-snapshot: enabled (every 60s, thin >10m0s to 5m0s)
```

### `snappy thin`

```text
Thinned 2 snapshot(s)
```

or:

```text
No snapshots to thin
```

### `snappy run`

```text
[2026-03-03 14:25:00] STARTUP  snappy run (interval=60s, thin >10m0s to 5m0s)
[2026-03-03 14:25:30] SNAPSHOT Created: 2026-03-03-142530
[2026-03-03 14:25:31] THIN     Thinned 1 snapshot(s)
[2026-03-03 14:25:31] LIST     3 snapshot(s)
```

## New Files

### `cmd/helpers.go` -- shared infrastructure

Extracted logic reusable across commands:

- `var jsonFlag bool` -- the `--json` flag value
- `requireTmutil() error` -- extracted from `cmd/root.go:92-94`, returns
  `"tmutil not found: this tool requires macOS with Time Machine support"`
- `newRunner() platform.CommandRunner` -- returns `platform.OSRunner{}`
- `loadSnapshots(ctx, runner, cfg) ([]snapshot.Snapshot, string, int, error)` --
  extracts the snapshot-fetching + APFS-enrichment logic from
  `internal/tui/commands.go:doRefresh` (lines 16-75) into a reusable function.
  Calls `platform.ListSnapshots`, `snapshot.ParseDate`, `platform.FindAPFSVolume`,
  `platform.GetSnapshotDetails`, and merges APFS details into snapshots.
  Returns (snapshots, apfsVolume, otherSnapCount, error).
- `writeJSON(w io.Writer, v any) error` -- marshals to indented JSON and writes

### `cmd/create.go` -- `snappy create`

- `cobra.NoArgs`, `RunE: runCreate`
- Calls `requireTmutil()`, then `platform.CreateSnapshot(ctx, runner)` with
  1-minute timeout
- Human output: `"Snapshot created: <date>"`
- JSON output: `{"date":"<date>"}`

### `cmd/list.go` -- `snappy list`

- `cobra.NoArgs`, `RunE: runList`
- Calls `loadSnapshots()` with 30-second timeout
- Human output: count header, then newest-first numbered list with relative
  time, UUID, purgeable/pinned flags, limits-shrink warning (plain text, no ANSI)
- JSON output: object with mount, count, snapshots array
- Reuses `snapshot.FormatRelativeTime()` for relative timestamps

### `cmd/status.go` -- `snappy status`

- `cobra.NoArgs`, `RunE: runStatus`
- Calls `platform.CheckStatus`, `platform.FindAPFSVolume`, `platform.GetDiskInfo`,
  `platform.ListSnapshots`, `platform.GetSnapshotDetails` with 30-second timeout
- Human output mirrors TUI info panel: TM status, mount, APFS volume, disk
  usage (via `DiskInfo.String()`), snapshot counts, auto-snapshot config
- JSON output: structured object with all fields

### `cmd/run.go` -- `snappy run`

- `cobra.NoArgs`, `RunE: runDaemon`
- Uses `signal.NotifyContext` for SIGINT/SIGTERM handling
- Creates `snapshot.AutoManager`, runs first iteration immediately, then loops
  on `time.NewTicker(cfg.AutoSnapshotInterval)`
- Each iteration: create snapshot, load snapshots, compute thin targets, delete
  targets, log counts
- Log format: `[2006-01-02 15:04:05] EVENT    message` (matches snappy-ez)
- Private `runIteration()` and `logLine()` helpers in same file

### `cmd/thin.go` -- `snappy thin`

- `cobra.NoArgs`, `RunE: runThin`
- Calls `loadSnapshots()`, creates `AutoManager` with `enabled=true`, calls
  `ComputeThinTargets()`, deletes each target with individual 30-second timeouts
- Reports successful deletions; returns error if any deletions failed
- Human output: `"Thinned N snapshot(s)"` or `"No snapshots to thin"`
- JSON output: `{"thinned": N}`

## Modified Files

### `cmd/root.go`

Add `--json` persistent flag in `init()`:

```go
rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "output in JSON format")
```

The `jsonFlag` variable lives in `cmd/helpers.go`.

### Scrut test files (help output updates)

Adding five new commands and the `--json` global flag changes help output.
These files need updated `Available Commands` and `Global Flags` sections:

- `tests/scrut/help.md` -- root help, help subcommand
- `tests/scrut/config-command.md` -- config/init help Global Flags
- `tests/scrut/config.md` -- if capturing help output
- `tests/scrut/flag-precedence.md` -- if capturing help output
- `tests/scrut/environment.md` -- if capturing help output
- `tests/scrut/errors.md` -- if capturing help output

Strategy: update with `make test-scrut-update`, then review diffs for correctness.

## New Test Files

### Scrut CLI tests

- `tests/scrut/create-command.md` -- help output, argument rejection
- `tests/scrut/list-command.md` -- help output, default invocation, `--json`
  validation
- `tests/scrut/status-command.md` -- help output, default invocation, `--json`
  validation
- `tests/scrut/thin-command.md` -- help output, argument rejection
- `tests/scrut/run-command.md` -- help output

Note: `create`, `thin`, and `run` require sudo for actual tmutil operations, so
scrut tests focus on help output and argument validation. `list` and `status`
can run without sudo (read-only operations).

### Go unit tests

- `cmd/helpers_test.go` -- `TestLoadSnapshots`, `TestWriteJSON` with mock runner
- `cmd/create_test.go` -- mock runner, verify human and JSON output
- `cmd/list_test.go` -- empty list, with snapshots, JSON structure
- `cmd/status_test.go` -- mock all platform calls, verify output fields
- `cmd/thin_test.go` -- no targets, with targets, partial failure
- `cmd/run_test.go` -- `runIteration` with mock, shutdown on context cancel

Mock runner pattern (in `cmd/mock_test.go` or per-file):

```go
type mockRunner struct {
    responses map[string]mockResponse
}
```

## Key Reusable Code

- `platform.CreateSnapshot` -- `internal/platform/tmutil.go`
- `platform.ListSnapshots` -- `internal/platform/tmutil.go`
- `platform.DeleteSnapshot` -- `internal/platform/tmutil.go`
- `platform.CheckStatus` -- `internal/platform/tmutil.go`
- `platform.GetDiskInfo` / `DiskInfo.String()` -- `internal/platform/disk.go`
- `platform.FindAPFSVolume` -- `internal/platform/diskutil.go`
- `platform.GetSnapshotDetails` -- `internal/platform/diskutil.go`
- `snapshot.ParseDate` -- `internal/snapshot/snapshot.go`
- `snapshot.FormatRelativeTime` -- `internal/snapshot/snapshot.go`
- `snapshot.NewAutoManager` / `ComputeThinTargets` -- `internal/snapshot/auto.go`
- `config.Load` -- `internal/config/config.go`

The `loadSnapshots` helper in `cmd/helpers.go` extracts lines 16-75 of
`internal/tui/commands.go:doRefresh` into a standalone function. The TUI's
`doRefresh` wraps this in a `tea.Cmd` closure and returns a `RefreshResultMsg`;
the new helper returns the data directly.

## Implementation Order

1. `cmd/helpers.go` -- shared infrastructure (jsonFlag, requireTmutil,
   loadSnapshots, writeJSON, newRunner)
2. `cmd/root.go` -- add `--json` persistent flag
3. `cmd/create.go` -- simplest command, validates the pattern
4. `cmd/list.go` -- exercises loadSnapshots and formatted output
5. `cmd/status.go` -- exercises different platform calls
6. `cmd/thin.go` -- introduces AutoManager usage outside TUI
7. `cmd/run.go` -- most complex, builds on thin + create patterns
8. Go unit tests for each command
9. Scrut tests for new commands
10. Update existing scrut tests for changed help output

## Verification

1. `make build` -- compiles successfully
2. `make test` -- Go unit tests pass
3. `make test-scrut` -- scrut CLI tests pass (including updated help output)
4. Manual testing on macOS:
   - `snappy create` creates a snapshot (requires sudo)
   - `snappy list` shows snapshots with relative times and APFS details
   - `snappy list --json | jq .` produces valid, parseable JSON
   - `snappy status` shows TM status, disk info, snapshot counts
   - `snappy status --json | jq .` produces valid JSON
   - `snappy thin` runs thinning pass
   - `snappy run` starts daemon loop, Ctrl-C shuts down cleanly
5. `make lint` -- no lint errors
6. `make fmt-check` -- formatting clean
