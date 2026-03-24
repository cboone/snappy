# Add In-TUI Help (Issue #56)

## Context

The Snappy TUI displays a minimal help bar at the bottom showing primary keybindings
(snapshot, refresh, auto-snap, open log, quit) via the bubbles `help.Model` component.
Navigation keys (scroll, tab) are hidden from users, and there is no way to discover them
without reading documentation. Issue #56 requests in-TUI help so users can discover all
keybindings from within the application.

The bubbles `help.Model` already supports a `ShowAll` boolean that toggles between
`ShortHelp()` (compact, one-line) and `FullHelp()` (expanded, multi-row columns). This
plan wires up a `?` keybinding to toggle that flag, adjusts the layout to accommodate the
expanded help, and adds tests.

## Changes

### 1. Add `Help` binding to `keyMap` (`internal/tui/model.go`)

- Add `Help key.Binding` field to the `keyMap` struct (after `ShiftTab`)
- In `defaultKeyMap()`, add:

  ```go
  Help: key.NewBinding(
      key.WithKeys("?"),
      key.WithHelp("?", "help"),
  ),
  ```

- Update `ShortHelp()` to append `k.Help` as the last item
- Update `FullHelp()` to append `k.Help` to the second (navigation) group, making both
  groups 5 items tall for a clean rectangular layout

### 2. Extract layout helper and add help height logic (`internal/tui/update.go`)

- Add a `helpBarHeight()` method:

  ```go
  const fullHelpHeight = 5

  func (m *Model) helpBarHeight() int {
      if m.help.ShowAll {
          return fullHelpHeight
      }
      return 1
  }
  ```

- Extract layout logic from `handleWindowSize` into a `recalcLayout()` method that uses
  `m.helpBarHeight()` instead of the hardcoded `1` in `fixedHeight`
- Simplify `handleWindowSize` to set width/height then call `m.recalcLayout()`

### 3. Add `?` key handler (`internal/tui/update.go`)

In `handleKey`, add a new case (after `ShiftTab`, before `ScrollUp`/`ScrollDown`):

```go
case key.Matches(msg, m.keys.Help):
    m.help.ShowAll = !m.help.ShowAll
    if m.help.ShowAll {
        m.keys.Help.SetHelp("?", "close help")
    } else {
        m.keys.Help.SetHelp("?", "help")
    }
    m.recalcLayout()
    return m, nil
```

The `recalcLayout()` call ensures the snapshot and log panels resize when the help bar
expands from 1 to 5 lines (or shrinks back).

### 4. Add tests (`internal/tui/model_test.go`)

- **`TestHelpToggle`**: Press `?`, verify `ShowAll` is true and help description changes
  to "close help". Press `?` again, verify it reverts.
- **`TestHelpToggleRelayoutsPanels`**: Send a `WindowSizeMsg`, record `snapVisibleRows`.
  Press `?`, verify `snapVisibleRows` decreased (panels shrank to make room).
- **`TestViewShowsHelpHint`**: Render default view, verify `"?"` appears in output.
- **`TestFullHelpShowsAllBindings`**: Press `?`, render, verify navigation binding
  descriptions ("scroll up", "scroll down", "next panel", "prev panel") appear.

## Files to modify

| File                         | What changes                                                                                          |
| ---------------------------- | ----------------------------------------------------------------------------------------------------- |
| `internal/tui/model.go`      | Add `Help` field to `keyMap`, update `ShortHelp`, `FullHelp`, `defaultKeyMap`                         |
| `internal/tui/update.go`     | Extract `recalcLayout`, add `helpBarHeight`, add `?` case in `handleKey`, refactor `handleWindowSize` |
| `internal/tui/model_test.go` | Add 4 test functions                                                                                  |

No new files. No changes to `view.go`, `styles.go`, `messages.go`, or `commands.go`.

## Verification

1. `make test` passes (unit tests including new help tests)
2. `make lint` passes
3. `make build && ./bin/snappy` manual check:
   - Help bar shows `? help` at the end of the short help line
   - Pressing `?` expands to full help showing all keybindings in two columns
   - Snapshot and log panels visibly shrink to accommodate expanded help
   - Pressing `?` again collapses back to the single-line help bar
   - Help text changes between "help" and "close help"
4. `make test-scrut` passes (no scrut changes expected since this is TUI-only, not CLI help)
