# 2026-03-06 Indicate pinned snapshots in TUI table

## Context

When auto-thinning attempts to delete a snapshot, macOS sometimes returns exit code 70 (ESTALE / stale NFS file handle), meaning the kernel holds a reference and the snapshot cannot be deleted. The thinning logic already tracks these "pinned" snapshots in `Model.thinPinned` to avoid retrying them, but there is no visual indication in the snapshot list table. Users have no way to see which snapshots are stuck.

## Goal

Show a "pinned" indicator in the STATUS column of the snapshot table for any snapshot whose date is present in `m.thinPinned`.

## Changes

### 1. Add indicator constant (`internal/tui/styles.go`)

Add a new constant at line 13:

```go
indicatorPinned = "📌"
```

### 2. Show pinned status in table rows (`internal/tui/update.go`)

In `updateSnapViewContent()` (lines 618-624), replace the simple `LimitsShrink` check with a slice-based approach that can combine multiple status indicators:

```go
if snap.UUID != "" {
    xid = fmt.Sprintf("%d", snap.XID)
    uuid = snap.UUID

    var parts []string
    if _, pinned := m.thinPinned[snap.Date]; pinned {
        parts = append(parts, indicatorPinned+" pinned")
    }
    if snap.LimitsShrink {
        parts = append(parts, indicatorWarning+" limits shrink")
    }
    status = strings.Join(parts, " ")

    // Compute XID delta ...
}
```

This handles the case where both pinned and LimitsShrink are true for the same snapshot.

### 3. Add tests (`internal/tui/model_test.go`)

Two new test functions, following the pattern of `TestViewAPFSDetails` (line 474):

**`TestViewPinnedIndicator`**: Create a model with two snapshots, add one snapshot's date to `thinPinned`, call `updateSnapViewContent()`, and assert the rendered view contains "pinned" for the pinned snapshot.

**`TestViewPinnedAndLimitsShrink`**: Same setup but with `LimitsShrink = true` on the pinned snapshot. Assert both "pinned" and "limits shrink" appear in the rendered view.

## Files to modify

- `internal/tui/styles.go` - add `indicatorPinned` constant (1 line)
- `internal/tui/update.go` - modify `updateSnapViewContent()` status logic (~7 lines changed)
- `internal/tui/model_test.go` - add 2 test functions (~50 lines)

## Notes

- The pushpin emoji (📌) is a wide character (2 columns in most terminals). Since STATUS is the last column, this has no alignment impact on other columns. If it looks off in practice, it can be swapped for a narrow Unicode symbol.
- The `statusMin = 20` constant in `snapTableColumns()` is sufficient for the single-indicator case ("📌 pinned" is ~10 display columns). The combined case ("📌 pinned ⚠ limits shrink") is ~28 columns and may get truncated on narrow terminals, but both conditions being true simultaneously is rare.

## Verification

1. `make test` - unit tests pass, including the two new tests
2. `make lint` - no lint errors
3. Manual: run the TUI, observe that after an ESTALE failure the affected snapshot shows "📌 pinned" in the STATUS column, and it clears after a manual refresh (r key)
