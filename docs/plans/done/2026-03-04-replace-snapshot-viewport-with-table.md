# 2026-03-04 Replace snapshot viewport with Bubbles table

## Context

The snapshot list currently uses a Bubbles viewport that displays pre-formatted text lines built by `formatSnapshotLine`. Replacing it with a Bubbles `table.Model` provides automatic column alignment, built-in row selection with cursor navigation, and a data-driven API. The table component is already available in the project's `charm.land/bubbles/v2` dependency but not yet imported.

## Files to modify

### `internal/tui/model.go`

- Add `"charm.land/bubbles/v2/table"` import
- Replace `snapView viewport.Model` with `snapTable table.Model`
- In `NewModel`: create a `table.Model` with initial columns, styles, height, width, and focused state
- Customize table styles to match the existing UI:
  - Header: bold, `colorSubtle` foreground
  - Cell: default padding
  - Selected: bold, `colorHighlight` foreground

### `internal/tui/update.go`

- Add `"charm.land/bubbles/v2/table"` import
- `handleWindowSize`: replace `m.snapView.SetWidth/SetHeight` with `m.snapTable.SetWidth/SetHeight`
- `handleScroll`: replace `m.snapView.Update(msg)` with `m.snapTable.Update(msg)`
- `handleMouseWheel`: replace `m.snapView.Update(msg)` with `m.snapTable.Update(msg)`
- `handleKey` (Tab case): call `m.snapTable.Focus()`/`m.snapTable.Blur()` when toggling focus
- `updateSnapViewContent`: build `[]table.Column` and `[]table.Row` instead of formatted strings; call `m.snapTable.SetColumns()` and `m.snapTable.SetRows()`
- Update the `fixedHeight` comment (the table header replaces the removed section title, so the arithmetic stays the same)

### `internal/tui/view.go`

- `renderSnapshotPanel`: replace `m.snapView.View()` with `m.snapTable.View()`
- Remove `formatSnapshotLine` (its logic moves into `updateSnapViewContent` as table row construction)

### `internal/tui/styles.go`

- Add a `tableStyles` field (or a helper) to `modelStyles` that returns a `table.Styles` derived from the existing color palette

## Column layout

Base columns (always shown):

| Column | Header | Approx width |
|--------|--------|--------------|
| #      | `#`    | 5            |
| Date   | `DATE` | 21           |
| Age    | `AGE`  | 14           |

APFS columns (shown when `m.apfsVolume != ""`):

| Column | Header   | Width        |
|--------|----------|--------------|
| UUID   | `UUID`   | flex (fills) |
| Status | `STATUS` | 22           |

Column widths recalculated on resize from `contentWidth(m.width)`. The UUID column gets the remaining space.

## Empty state

When there are no snapshots, set a single row with the placeholder text "(none, press 's' to create the first snapshot)" in the Date column and empty strings elsewhere.

## Key bindings

Keep the current approach: app-level keys (s, r, a, q, tab) handled in `handleKey`; j/k/up/down forwarded to the table via `handleScroll`. The table's internal KeyMap handles cursor movement.

## Verification

- `make build` compiles cleanly
- `make test` passes
- `make lint` passes
- Run `bin/snappy` and confirm:
  - Column headers appear in the snapshot panel
  - Snapshots are aligned in columns
  - j/k and arrow keys move the selection cursor
  - Mouse wheel scrolls the snapshot list
  - Tab switches focus between panels
  - Selected row is highlighted
