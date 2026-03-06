# XID Delta and Tidemark Display

## Context

Users want visibility into snapshot size and filesystem activity. The original
goal was to use `tmutil uniquesize` and `calculatedrift`, but research confirmed
these commands don't work with local APFS snapshots (uniquesize returns 0B,
calculatedrift rejects local machine directories). Per-snapshot size data
(Private Size, Cumulative Size) is available in the Disk Utility GUI but Apple
does not expose it via command-line tools or public APIs.

This plan implements two proxy metrics that are available:

1. **XID delta** per snapshot: the APFS transaction count difference between
   consecutive snapshots, displayed in the snapshot list table
2. **Tidemark**: the container resize minimum (`diskutil apfs resizeContainer
<container> limits -plist` -> `MinimumSizeNoGuard`), displayed in the info
   panel. This is the minimum container size constrained by file and snapshot
   usage, matching the "Tidemark" concept Apple uses in Disk Utility.

A separate GitHub issue will track revisiting `tmutil uniquesize` and
`calculatedrift` once Time Machine is enabled on a backup destination.

## Changes

### 1. GitHub issue

Create an issue to revisit snapshot size display with TM enabled, noting that
all testing was done with TM disabled and snappy-created snapshots only.

### 2. Platform layer: container tidemark

**`internal/platform/diskutil.go`**

- Add `APFSContainerReference string` field to `diskutilInfoPlist`
- Add new plist struct for the resize container response:

  ```go
  type containerLimitsPlist struct {
      MinimumSizeNoGuard int64 `plist:"MinimumSizeNoGuard"`
      ContainerCurrentSize int64 `plist:"ContainerCurrentSize"`
  }
  ```

- Add `GetContainerReference(ctx, r, mount) (string, error)`:
  calls `diskutil info -plist <mount>`, returns `APFSContainerReference`
- Add `GetContainerTidemark(ctx, r, container) (int64, error)`:
  calls `diskutil apfs resizeContainer <container> limits -plist`, parses
  `MinimumSizeNoGuard`, returns raw bytes

**`internal/platform/disk.go`**

- Add `FormatBytes(b int64) string`: formats bytes to human-readable
  (e.g., 2153406005248 -> "2.0 TB"). Use binary units to match `df -h`.

**`internal/platform/platform_test.go`**

- Add tests for `GetContainerReference`, `GetContainerTidemark`, `FormatBytes`
- Use existing `mockRunner` pattern with canned plist responses

### 3. TUI messages

**`internal/tui/messages.go`**

- Add `Tidemark int64` field to `RefreshResultMsg`

### 4. TUI model

**`internal/tui/model.go`**

- Add `apfsContainer string` and `tidemark string` fields to `Model`
- Update `NewModel()` signature: add `apfsContainer` parameter
- Store `apfsContainer` in the model

### 5. TUI commands

**`internal/tui/commands.go`**

- Update `doRefresh()` signature: add `apfsContainer string` parameter
- Inside `doRefresh()`, if `apfsContainer != ""`, call
  `platform.GetContainerTidemark(ctx, runner, apfsContainer)` and include the
  result in `RefreshResultMsg.Tidemark`
- Graceful degradation: if the call fails, set `Tidemark` to 0 (no display)

### 6. TUI update (snapshot table)

**`internal/tui/update.go`**

- Update all `doRefresh()` call sites to pass `m.apfsContainer`
- In `handleRefreshResult()`: format `msg.Tidemark` via `platform.FormatBytes()`
  and store as `m.tidemark`
- In `snapTableColumns()`: add DELTA column (width 7) between XID and UUID.
  Update `ncols` to 6. Adjust `fixedWidth` calculation. New column order:
  DATE | AGE | XID | DELTA | UUID | STATUS
- In `updateSnapViewContent()`: compute XID delta for each snapshot.
  Snapshots are stored ascending (oldest first). For snapshot i:
  - If `i == 0` or `snap.UUID == ""`: delta is blank
  - Otherwise: `delta = snap.XID - snapshots[i-1].XID`, formatted as integer
  - Rows are built newest-first (existing reverse loop), so look up the
    original index to find the predecessor

**`internal/tui/styles.go`**

- Update `contentWidth()` minimum from 55 to 65 to accommodate the new column.
  Update the comment explaining the floor calculation.

### 7. TUI view (info panel)

**`internal/tui/view.go`**

- In `renderInfoPanel()`: append tidemark to line 1 (alongside Volume and Disk):
  `label("Tidemark:") + " " + m.tidemark`
- If `m.tidemark == ""`, omit the tidemark segment entirely

### 8. Startup

**`cmd/root.go`**

- After `FindAPFSVolume()`, call `platform.GetContainerReference(ctx, runner,
cfg.MountPoint)` to discover the APFS container
- Pass `apfsContainer` to `tui.NewModel()`
- Log the container reference at startup if present

### 9. CLI list command

**`cmd/list.go`**

- In `writeListHuman()`: append XID delta to each snapshot line (for snapshots
  with APFS details and a predecessor)
- In `writeListJSON()`: add `"xid_delta"` field (omitempty) to `jsonSnapshot`

### 10. CLI list helpers

**`cmd/helpers.go`**

- No changes needed. XID is already populated in snapshots via `loadSnapshots()`.
  Delta is computed at display time.

## File summary

| File                                 | Change                                                    |
| ------------------------------------ | --------------------------------------------------------- |
| `internal/platform/diskutil.go`      | Add container reference + tidemark functions              |
| `internal/platform/disk.go`          | Add `FormatBytes()`                                       |
| `internal/platform/platform_test.go` | Tests for new functions                                   |
| `internal/tui/messages.go`           | Add `Tidemark` to `RefreshResultMsg`                      |
| `internal/tui/model.go`              | Add `apfsContainer`, `tidemark` fields; update `NewModel` |
| `internal/tui/commands.go`           | Fetch tidemark in `doRefresh()`                           |
| `internal/tui/update.go`             | DELTA column, tidemark handling, pass container           |
| `internal/tui/styles.go`             | Update minimum content width                              |
| `internal/tui/view.go`               | Display tidemark in info panel                            |
| `cmd/root.go`                        | Discover container at startup, pass to model              |
| `cmd/list.go`                        | Add XID delta to CLI output                               |

## Verification

1. `make build` compiles successfully
2. `make test` passes (new platform tests + existing tests)
3. Run `./snappy` and verify:
   - DELTA column appears in snapshot table with integer values
   - Oldest snapshot shows blank delta
   - Tidemark appears on info panel line 1 (e.g., "Tidemark: 2.0 TB")
   - Tidemark updates on each refresh cycle
4. Run `./snappy list` and verify XID delta in human-readable output
5. Run `./snappy list --json` and verify `xid_delta` field
6. `make lint` passes
7. `make test-scrut` passes (update scrut tests if needed)
