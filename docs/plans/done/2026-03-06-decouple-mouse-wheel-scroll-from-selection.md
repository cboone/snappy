# 2026-03-06 Decouple Mouse Wheel Scroll from Selection

## Context

Both scrollable panels (snapshot table, log panel) currently move the selected/highlighted item when the mouse wheel is scrolled. The pane viewport scrolls secondarily, only to keep the selection visible. The user wants the reverse: mouse wheel should scroll the pane viewport, and selection should only change via keyboard (j/k, up/down) or mouse click.

## Approach

### Log Panel (simple)

The log panel already has a `viewport.Model` (`logView`) and a separate `logCursor int`. Currently, `handleMouseWheel` calls `moveLogCursor()` which moves the cursor AND adjusts the viewport. The fix is to scroll the viewport directly without touching the cursor.

**Add `scrollLogView` method** in `update.go`:

```go
func (m *Model) scrollLogView(delta int) {
    offset := m.logView.YOffset() + delta
    maxOffset := max(m.logTotalLines-m.logView.Height(), 0)
    m.logView.SetYOffset(max(min(offset, maxOffset), 0))
}
```

**Change `handleMouseWheel`** (log section): call `m.scrollLogView(-1)` / `m.scrollLogView(1)` instead of `m.moveLogCursor(-1)` / `m.moveLogCursor(1)`.

Keyboard scrolling (`handleScroll`) continues to use `moveLogCursor` unchanged.

### Snapshot Panel (workaround for private table viewport)

The Bubbles v2 `table.Model` tightly couples its cursor (selection) with its internal viewport. All movement methods (`MoveUp`, `MoveDown`, `SetCursor`) change both. The viewport field is private with no independent scroll API.

**Strategy**: Set the table height large enough to render ALL rows (disabling internal clipping), then manually clip the output in the view layer using our own scroll offset.

#### Model changes (`model.go`)

Add two fields to `Model`:

```go
snapScrollOffset int // first visible data row in snapshot panel
snapVisibleRows  int // number of data rows visible in snapshot panel
```

#### `handleWindowSize` (`update.go`, line 117)

- Store `snapVisibleRows = snapH - 1` (panel content height minus header line)
- Remove `m.snapTable.SetHeight(snapH)` (table height is now set in `updateSnapViewContent`)
- Keep `m.snapTable.SetWidth(cw)` as-is

#### `updateSnapViewContent` (`update.go`, line 602)

After `m.snapTable.SetRows(rows)`, add:

```go
m.snapTable.SetHeight(max(len(rows)+1, 2))
m.clampSnapScroll()
```

`SetHeight(N+1)` makes the internal viewport height = N (it subtracts the header), so all N rows render. `max(..., 2)` prevents a zero-height viewport on empty tables.

#### New helper methods (`update.go`)

**`clampSnapScroll`**: Clamp offset to valid range after row count or panel size changes.

```go
func (m *Model) clampSnapScroll() {
    maxOffset := max(len(m.snapTable.Rows())-m.snapVisibleRows, 0)
    m.snapScrollOffset = max(min(m.snapScrollOffset, maxOffset), 0)
}
```

**`ensureSnapCursorVisible`**: After keyboard cursor movement, auto-scroll so the cursor stays in view.

```go
func (m *Model) ensureSnapCursorVisible() {
    cursor := m.snapTable.Cursor()
    if cursor < m.snapScrollOffset {
        m.snapScrollOffset = cursor
    } else if cursor >= m.snapScrollOffset+m.snapVisibleRows {
        m.snapScrollOffset = cursor - m.snapVisibleRows + 1
    }
    m.clampSnapScroll()
}
```

#### `handleMouseWheel` (`update.go`, line 254)

Snap panel section: replace `m.snapTable.MoveUp(1)` / `MoveDown(1)` with:

```go
case tea.MouseWheelUp:
    m.snapScrollOffset--
case tea.MouseWheelDown:
    m.snapScrollOffset++
```

Follow with `m.clampSnapScroll()`.

#### `handleScroll` (`update.go`, line 277)

Snap panel case: after `m.snapTable.Update(msg)` (which moves the cursor), add `m.ensureSnapCursorVisible()`.

#### `handleMouseClick` (`update.go`, line 240)

Replace `snapRowAtVisualLine` approach with direct index calculation:

```go
case msg.Y >= m.snapPanelY:
    m.setFocusPanel(panelSnap)
    line := msg.Y - m.snapPanelY - 2 // -1 border, -1 header
    if line >= 0 && line < m.snapVisibleRows {
        row := line + m.snapScrollOffset
        if row >= 0 && row < len(m.snapTable.Rows()) {
            m.snapTable.SetCursor(row)
        }
    }
```

#### Remove `snapRowAtVisualLine` (`update.go`, line 777)

No longer needed; direct index calculation via `snapScrollOffset` is simpler and more reliable.

#### `renderSnapshotPanel` (`view.go`, line 107)

Split `table.View()` into header + body, clip body using `snapScrollOffset`:

```go
tableOut := m.snapTable.View()
parts := strings.SplitN(tableOut, "\n", 2)
header := parts[0]

var clipped string
if len(parts) > 1 {
    bodyLines := strings.Split(parts[1], "\n")
    end := min(m.snapScrollOffset+m.snapVisibleRows, len(bodyLines))
    start := min(m.snapScrollOffset, end)
    visible := bodyLines[start:end]
    for len(visible) < m.snapVisibleRows {
        visible = append(visible, "")
    }
    clipped = header + "\n" + strings.Join(visible, "\n")
} else {
    clipped = header
}

rendered := style.Width(sw).Render(clipped)
```

## Files to Modify

| File | Summary |
|------|---------|
| `internal/tui/model.go` | Add `snapScrollOffset`, `snapVisibleRows` fields |
| `internal/tui/update.go` | Modify `handleMouseWheel`, `handleScroll`, `handleMouseClick`, `handleWindowSize`, `updateSnapViewContent`; add `clampSnapScroll`, `ensureSnapCursorVisible`, `scrollLogView`; remove `snapRowAtVisualLine` |
| `internal/tui/view.go` | Modify `renderSnapshotPanel` to split and clip table output |
| `internal/tui/model_test.go` | Update existing scroll/click tests; add new tests |

## Tests

### Existing tests to update

- `TestMouseClickSnapshotSelectsTopVisibleRowWhenTableIsOffset`: Rework to use `snapScrollOffset` instead of relying on table internal scroll.
- `TestSnapRowAtVisualLineMatchesRenderedRows`: Remove or replace; the function it tests is being removed.
- `TestSnapshotPanelKeepsViewportHeightWhenEmpty`: Verify padding logic still produces consistent panel height.

### New tests to add

- Mouse wheel on snap panel scrolls `snapScrollOffset` without changing `snapTable.Cursor()`
- Mouse wheel on log panel scrolls `logView.YOffset()` without changing `logCursor`
- Keyboard scroll on snap panel moves cursor and auto-scrolls to keep cursor visible
- Click on snap panel with non-zero `snapScrollOffset` selects the correct row
- `snapScrollOffset` is clamped when row count decreases
- `ensureSnapCursorVisible` scrolls down/up correctly
- `renderSnapshotPanel` clips body lines correctly with non-zero offset

## Verification

1. `make test` passes (all unit tests)
2. `make lint` passes
3. `make build` succeeds
4. Manual testing: mouse wheel scrolls panes without changing selection
5. Manual testing: keyboard up/down still changes selection and scrolls to follow
6. Manual testing: mouse click still selects the correct item, even when scrolled
7. Manual testing: window resize preserves scroll position (clamped if needed)
