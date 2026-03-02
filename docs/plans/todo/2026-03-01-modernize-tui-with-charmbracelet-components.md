# 2026-03-01 Modernize TUI with Charmbracelet Components

## Context

The Snappy TUI is functionally complete but visually stuck in its bash proof-of-concept style: 80-character ASCII separator lines (`====`, `----`), raw `fmt.Fprintf` string concatenation, basic ANSI colors (1-6), no responsive layout, and no use of Charmbracelet's component library. The terminal width/height are captured via `WindowSizeMsg` but never used. This plan converts the TUI to a modern Charmbracelet design that fills the full terminal height and width, using bordered panels, adaptive colors, responsive layout, and three `bubbles` components: help (key bindings footer), viewport (scrollable snapshot list), and spinner (async operation feedback).

This plan targets the **Charmbracelet v2 ecosystem** (`charm.land` vanity imports), which reached stable v2.0.0 on February 24, 2026. The v2 release includes major API changes: `View()` returns `tea.View` (declarative) instead of `string`, `tea.KeyMsg` is replaced by `tea.KeyPressMsg`, `lipgloss.AdaptiveColor` is replaced by the pure `lipgloss.LightDark` function with `tea.BackgroundColorMsg`, and bubbles sub-models use getter/setter methods instead of direct field access.

## Dependency Migration

Upgrade the existing Charmbracelet dependencies from v1 (`github.com/charmbracelet/...`) to v2 (`charm.land/.../v2`), and add the new `bubbles` dependency:

| Old v1 module path                   | New v2 module path        |
| ------------------------------------ | ------------------------- |
| `github.com/charmbracelet/bubbletea` | `charm.land/bubbletea/v2` |
| `github.com/charmbracelet/lipgloss`  | `charm.land/lipgloss/v2`  |
| (new)                                | `charm.land/bubbles/v2`   |

The `charm.land/bubbles/v2` module provides:

- `bubbles/help` + `bubbles/key` for the key bindings footer
- `bubbles/viewport` for the scrollable snapshot list and scrollable log panel
- `bubbles/spinner` for async operation indicators

Run:

```bash
go get charm.land/bubbletea/v2 charm.land/lipgloss/v2 charm.land/bubbles/v2
```

Then update all import statements project-wide:

```go
// Before (v1)
import tea "github.com/charmbracelet/bubbletea"
import "github.com/charmbracelet/lipgloss"

// After (v2)
import tea "charm.land/bubbletea/v2"
import "charm.land/lipgloss/v2"
import "charm.land/bubbles/v2/help"
import "charm.land/bubbles/v2/key"
import "charm.land/bubbles/v2/viewport"
import "charm.land/bubbles/v2/spinner"
```

Remove the old v1 module paths from `go.mod` after updating all imports (`go mod tidy` handles this).

## Files to Modify

### 1. `internal/tui/styles.go` (complete rewrite)

Delete the ASCII separator constants (`separator`, `thinSeparator`) and the old style variables (`bold`, `dim`, `green`, `yellow`, `red`, `cyan`, `magenta`, `titleStyle`, `keyStyle`).

Replace with:

**Light/dark adaptive colors** using Lip Gloss v2's pure `lipgloss.LightDark` function (replaces the removed `lipgloss.AdaptiveColor`). The model stores a `hasDarkBG bool` field (populated from `tea.BackgroundColorMsg`), and `styles.go` provides a helper that returns the appropriate color:

```go
func adaptiveColor(hasDarkBG bool, light, dark string) color.Color {
    return lipgloss.LightDark(hasDarkBG)(lipgloss.Color(light), lipgloss.Color(dark))
}
```

Color variables become functions or are computed at render time:

- `colorSubtle`, `colorHighlight`, `colorGreen`, `colorYellow`, `colorRed`, `colorCyan`, `colorMagenta`, `colorDim`, `colorBorder`, `colorTitleBg`, `colorTitleFg`

**Inline text styles** (renamed with `text` prefix for clarity):

- `textBold`, `textDim`, `textGreen`, `textYellow`, `textRed`, `textCyan`, `textMagenta`

**Section styles** using Lipgloss borders and padding:

- `titleBarStyle`: full-width colored background bar, bold, centered text
- `sectionStyle`: rounded border, border foreground color, horizontal padding
- `sectionTitleStyle`: bold with highlight color
- `helpBarStyle`: subtle foreground, horizontal padding
- `statusOnStyle` / `statusOffStyle`: bold green/red for auto-snapshot indicator
- `keyBindStyle`: bold highlight for `[s]`, `[r]`, etc.
- `snapshotNumberStyle`: green bold for the `1.`, `2.` numbering
- `spinnerStyle`: highlight color for the spinner dots

**Unicode indicator constants:**

- `indicatorOn` (`●`) / `indicatorOff` (`○`) for auto-snapshot status
- `indicatorPurge` (`◇`) / `indicatorPinned` (`◆`) for snapshot flags
- `indicatorWarning` (`⚠`) for limits-shrink

**Helper functions:**

- `contentWidth(termWidth int) int`: returns usable width inside a bordered section (accounts for border + padding), with a minimum of 40.
- `flexPanelHeights(termHeight, fixedHeight int) (snapH, logH int)`: calculates the viewport heights for the snapshot and log panels by subtracting fixed-height sections (title bar, info panel, section title lines, borders, help bar) from the terminal height, then splitting the remaining space equally (1:1). Returns minimum heights of 3 each to remain usable.

### 2. `internal/tui/model.go` (add sub-models and key bindings)

**Add key binding type** implementing `help.KeyMap` (imports from `charm.land/bubbles/v2/key` and `charm.land/bubbles/v2/help`):

```go
type keyMap struct {
    Snapshot  key.Binding
    Refresh   key.Binding
    AutoSnap  key.Binding
    Quit      key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Snapshot, k.Refresh, k.AutoSnap, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{k.ShortHelp()}
}
```

**Add new fields to Model struct:**

```go
keys        keyMap
help        help.Model
snapView    viewport.Model  // scrollable snapshot list
logView     viewport.Model  // scrollable log panel
spinner     spinner.Model
loading     bool            // true during async operations (refresh, create, thin)
focusLog    bool            // true when log panel has scroll focus (default: snapshots)
hasDarkBG   bool            // true when terminal has dark background (from tea.BackgroundColorMsg)
```

**Add key bindings** to `keyMap`:

```go
ScrollUp   key.Binding  // k / up arrow
ScrollDown key.Binding  // j / down arrow
Tab        key.Binding  // tab to switch focus between snapshot list and log
```

Include scroll/tab in `FullHelp()` but not `ShortHelp()` to keep the footer compact.

**Update `NewModel`** to initialize the sub-models:

- `keys`: define bindings for s/r/a/q with descriptions, plus scroll and tab bindings
- `help`: `help.New()`, set width to default 80. Use `help.DefaultDarkStyles()` or `help.DefaultLightStyles()` based on `hasDarkBG` (v2 removes auto-detection; style is set explicitly)
- `snapView`: `viewport.New()` then `m.snapView.SetWidth(76)` and `m.snapView.SetHeight(10)` (v2 uses getter/setter; initial size is recalculated on `WindowSizeMsg`)
- `logView`: `viewport.New()` then `m.logView.SetWidth(76)` and `m.logView.SetHeight(10)` (same pattern)
- `spinner`: `spinner.New()` with `spinner.Dot` style and highlight color
- `width`/`height`: default to 80/24
- `focusLog`: default `false` (snapshot list has initial focus)
- `hasDarkBG`: default `true` (safe default; updated on first `tea.BackgroundColorMsg`)

### 3. `internal/tui/update.go` (forward sub-model messages, manage spinner)

**`Update` method changes:**

- Forward `spinner.TickMsg` to `m.spinner.Update(msg)` when `m.loading` is true
- On `tea.BackgroundColorMsg` (new in v2): set `m.hasDarkBG = msg.IsDark()`, then rebuild styles that depend on the light/dark palette. Bubble Tea v2 sends this message at startup and whenever the terminal background changes.
- On `tea.WindowSizeMsg`: update `m.help.Width`, recalculate and resize both viewports using `m.snapView.SetWidth(w)` / `m.snapView.SetHeight(h)` and same for `m.logView` (v2 getter/setter pattern)
- On `tea.KeyPressMsg` (renamed from v1's `tea.KeyMsg`):
  - Use `key.Matches(msg, m.keys.Snapshot)` etc. instead of string matching (more idiomatic, supports the help component)
  - Tab: toggle `m.focusLog` to switch scroll focus between snapshot list and log
  - Forward scroll keys (j/k/up/down) to the focused viewport (`m.snapView` or `m.logView` depending on `m.focusLog`)

**Viewport height calculation on WindowSizeMsg:**

```text
fixedHeight = titleBar(1) + infoPanelHeight + sectionTitleOverhead(2 each) + helpBar(1)
flexibleHeight = termHeight - fixedHeight
snapViewHeight = flexibleHeight / 2
logViewHeight  = flexibleHeight - snapViewHeight  // handles odd pixel
```

The section title overhead accounts for the "LOCAL SNAPSHOTS (n)" and "RECENT LOG" heading lines plus border lines within each bordered panel.

Use `m.snapView.SetHeight(snapViewHeight)` and `m.logView.SetHeight(logViewHeight)` (v2 setter pattern) instead of direct field assignment.

**Spinner lifecycle:**

- Set `m.loading = true` and start spinner (`m.spinner.Tick`) when initiating async operations:
  - `handleKey` 's': creating snapshot
  - `handleKey` 'r': refreshing
  - `handleTick` when auto-snapshot fires
- Set `m.loading = false` when results arrive:
  - `handleRefreshResult`
  - `handleSnapshotCreated`
  - `handleThinResult`

**Viewport content updates:**

- **Snapshot viewport**: In `handleRefreshResult`, rebuild the full snapshot list content (all snapshots rendered as lines, newest first) and call `m.snapView.SetContent(...)`. This replaces the old bookend/ellipsis logic with a scrollable list of ALL snapshots. Note: in bubbles v2, `SetContent` is a method (not direct field assignment).
- **Log viewport**: In `handleRefreshResult`, `handleSnapshotCreated`, `handleThinResult`, and `handleKey` (for 'a' toggle), rebuild the log content from `m.log.Entries()` (colorized, newest first) and call `m.logView.SetContent(...)`. Call `m.logView.GotoBottom()` to auto-scroll to the newest entry.

**Add helper methods** on Model:

- `updateSnapViewContent()`: renders all snapshot lines into a string and sets it on the viewport
- `updateLogViewContent()`: renders all log entries (colorized) into a string and sets it on the viewport

**Visual focus indicator**: The focused viewport's border can use `colorHighlight` while the unfocused one uses `colorBorder`, so the user knows which panel responds to scroll keys.

### 4. `internal/tui/view.go` (rewrite with Lipgloss composition)

Change all `render*` methods from writing to a `*strings.Builder` parameter to returning `string`. Compose the full view with `lipgloss.JoinVertical`.

**`View()` method (Bubble Tea v2: returns `tea.View`):**

In Bubble Tea v2, `View()` returns a `tea.View` struct instead of a plain `string`. This enables declarative control over alt screen, mouse mode, window title, and cursor. Use `tea.NewView(content)` to create the view, then set declarative fields.

The TUI fills the full terminal height. Fixed-height sections (title, info, log, help) are rendered first, and the snapshot viewport expands to fill all remaining vertical space.

```go
func (m Model) View() tea.View {
    if m.quitting {
        return tea.NewView("")
    }
    w := m.width
    if w == 0 {
        w = 80
    }

    titleBar := m.renderTitleBar(w)
    infoPanel := m.renderInfoPanel(w)
    logPanel := m.renderLogPanel(w)
    helpBar := m.renderHelpBar(w)
    snapPanel := m.renderSnapshotPanel(w)

    content := lipgloss.JoinVertical(lipgloss.Left,
        titleBar,
        infoPanel,
        snapPanel,
        logPanel,
        helpBar,
    )

    v := tea.NewView(content)
    v.AltScreen = true
    return v
}
```

Note: the existing v1 code uses `tea.WithAltScreen()` as a program option. In v2, this program option is removed; instead, set `v.AltScreen = true` in the `View()` return value. This also means `tea.EnterAltScreen` / `tea.ExitAltScreen` commands are no longer needed.

**`renderTitleBar(width int) string`:** Full-width colored bar with "SNAPPY v{version}" and subtitle. If `m.loading`, append spinner animation.

**`renderInfoPanel(width int) string`:** Bordered rounded box containing volume, refresh, last-refresh, Time Machine status, APFS volume, disk info, and auto-snapshot status (with Unicode indicators). Uses `sectionStyle.Width(contentWidth(width)).Render(...)`.

**`renderSnapshotPanel(width int) string`:** Bordered rounded box with section title "LOCAL SNAPSHOTS (n)" plus diff summary. The body is `m.snapView.View()`, which renders the scrollable viewport of all snapshot lines. No bookend/ellipsis logic needed; all snapshots are listed and the viewport handles scrolling. The section title is rendered above the viewport content, inside the bordered box.

**`renderLogPanel(width int) string`:** Bordered rounded box with section title "RECENT LOG". The body is `m.logView.View()`, rendering the scrollable viewport of colorized log entries (newest first). Both panels share extra vertical space equally (1:1 split). The focused panel (determined by `m.focusLog`) gets a highlighted border color.

**`renderHelpBar(width int) string`:** Renders `m.help.View(m.keys)` inside `helpBarStyle`. No border, sits as a footer.

**`colorizeLogEntry`:** Update to use the renamed `text*` style variables.

**`formatAutoStatus() string`:** Uses `indicatorOn`/`indicatorOff` Unicode symbols alongside styled "on"/"off" text.

**`formatSnapshotLine(i, count int) string`:** Uses `indicatorPurge`/`indicatorPinned`/`indicatorWarning` Unicode symbols for APFS flags.

### 5. `internal/tui/messages.go` (minimal changes)

Message types are mostly unchanged. The spinner's `spinner.TickMsg` is handled by the standard `spinner.Update` call; no custom message type is needed. Review any existing message types that reference v1 types (e.g., `tea.KeyMsg`) and update to v2 equivalents (`tea.KeyPressMsg`) if applicable.

### 6. `internal/tui/commands.go` (no changes)

Command factories are unchanged. The spinner is started by returning `m.spinner.Tick` alongside the existing commands via `tea.Batch`.

### 7. `internal/tui/model_test.go` (update for model changes)

**Update `testModel()`:** Initialize the new sub-model fields (keys, help, logView, spinner, hasDarkBG) to match the updated `NewModel`.

**Test assertions:** All existing `strings.Contains` checks should continue to pass because the content strings ("SNAPPY", "LOCAL SNAPSHOTS (0)", "press 's'", "[s]", snapshot dates, "purgeable", etc.) are preserved in the new rendering. Lipgloss wraps text in ANSI escape codes but does not alter the underlying text content.

**View() return type:** Tests that call `m.View()` now receive a `tea.View` struct instead of a `string`. Extract the rendered content for assertions. Bubble Tea v2 provides `tea.WithColorProfile(...)` and `tea.WithWindowSize(w, h)` program options for deterministic test output.

**If ANSI codes interfere with assertions:** Use `tea.WithColorProfile(colorprofile.Ascii)` when creating the test program, or use `lipgloss.NewWriter(io.Discard, colorprofile.Ascii)` to force ASCII-only output. Check how the existing tests handle the current Lipgloss-styled output (they already pass with `green.Render()`, `bold.Render()`, etc.), and apply the same pattern.

**Update bookend test:** `TestViewBookend` currently checks for `"... and 2 more ..."`. Since the bookend logic is removed (all snapshots render in the viewport), this test should change to verify that all 6 snapshots appear in the viewport content, or be replaced with a test that verifies the snapshot viewport contains the expected content.

**Add new tests:**

- `TestViewSpinnerDuringLoading` to verify the spinner appears when `m.loading = true`.
- `TestViewFullHeight` to verify the rendered output uses the full terminal height.

## Visual Layout

The TUI fills the full terminal. Fixed-height sections are the title bar (1 line), info panel (~7 lines with border), and help bar (1 line). The snapshot panel and log panel split remaining vertical space equally (1:1) and are both scrollable viewports. Tab switches scroll focus between them, indicated by a highlighted border on the focused panel.

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│              ● SNAPPY v1.0  Time Machine Local Snapshot Manager        ⠋   │  Title bar (1 line)
╭─────────────────────────────────────────────────────────────────────────────╮
│ Volume: /    Refresh: 60s    Last: 2026-03-01T15:00:00                     │  Info panel (fixed)
│ Time Machine: Configured                                                    │
│ APFS Volume: disk3s5    Other snapshots: 2                                  │
│ Disk: 460Gi total, 215Gi used, 242Gi available (48%)                       │
│ Auto-snapshot: ● on    every 60s    next in 45s    thin >10m to 300s       │
╰─────────────────────────────────────────────────────────────────────────────╯
╭─────────────────────────────────────────────────────────────────────────────╮
│ LOCAL SNAPSHOTS (12)  [+1 added, 0 removed]                                 │  Snapshot panel
│                                                                              │  (viewport, scrollable,
│  1.  2026-03-01-150000   (just now)     ABC-123  ◇ purgeable               │   shares flexible height
│  2.  2026-03-01-145000   (10m ago)      DEF-456  ◆ pinned  ⚠ limits shrink │   with log panel, 1:1)
│  3.  2026-03-01-143000   (30m ago)      GHI-789  ◇ purgeable               │
│  4.  2026-03-01-142000   (40m ago)      JKL-012  ◇ purgeable               │
│  ...                                                     (scroll with j/k) │
╰─────────────────────────────────────────────────────────────────────────────╯
╭─────────────────────────────────────────────────────────────────────────────╮
│ RECENT LOG                                                                   │  Log panel
│                                                                              │  (viewport, scrollable,
│ [15:00:05] Refresh: 12 snapshots, disk 460Gi total...                       │   shares flexible height
│ [15:00:00] Snapshot created: 2026-03-01-150000                              │   with snapshot panel, 1:1)
│ [14:59:55] Auto-snapshot: Creating auto-snapshot...                         │
│ [14:59:50] Thinned 1 snapshot(s) older than 10m to 5s cadence              │
│ [14:59:45] snappy v0.1.0 | volume=/ | refresh=60s                          │
╰─────────────────────────────────────────────────────────────────────────────╯
  s snapshot  r refresh  a auto-snap  j/k scroll  tab focus  q quit             Help bar (1 line)
```

Key differences from the current layout:

- **Full-height**: TUI fills the entire terminal; flexible sections grow with terminal height
- **Scrollable snapshots**: All snapshots listed (no bookend/ellipsis); scroll with j/k or arrow keys
- **Scrollable log**: Log entries shown in a scrollable viewport; auto-scrolls to bottom on new entries
- **1:1 split**: Extra vertical space shared equally between snapshot and log panels
- **Focus switching**: Tab toggles scroll focus; focused panel has a highlighted border
- Title is a full-width colored bar (not text between `====` lines)
- Each section has rounded borders (`╭╮╰╯│`) instead of ASCII separators
- Padding inside bordered sections adds breathing room
- Unicode indicators for status (●/○), snapshot flags (◇/◆), and warnings (⚠)
- Spinner animation (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏) shows during async operations
- Help bar uses canonical Charm-style `key description` format
- Layout adapts to terminal width and height

## Implementation Sequence

1. Upgrade dependencies: `go get charm.land/bubbletea/v2 charm.land/lipgloss/v2 charm.land/bubbles/v2` and then `go mod tidy` to remove old v1 paths
2. Update all import statements project-wide from `github.com/charmbracelet/...` to `charm.land/.../v2`
3. `internal/tui/styles.go` (complete rewrite: `LightDark` adaptive colors, new styles, constants)
4. `internal/tui/model.go` (add key bindings, help, viewport, spinner sub-models, `hasDarkBG` field)
5. `internal/tui/update.go` (handle `tea.BackgroundColorMsg`, `tea.KeyPressMsg`, forward sub-model messages, spinner lifecycle, viewport content updates with getter/setter API)
6. `internal/tui/view.go` (rewrite rendering: `View()` returns `tea.View`, Lipgloss composition, `v.AltScreen = true`)
7. `cmd/root.go` (remove `tea.WithAltScreen()` program option, now set declaratively in `View()`)
8. `internal/tui/model_test.go` (update testModel, adjust for `tea.View` return type, verify assertions, add spinner test)
9. Run `make test` and `make lint` to verify
10. Build and visually test: `make build && ./bin/snappy`

## v2 API Migration Reference

Quick reference for the most relevant v1-to-v2 changes in this plan:

| v1 API                                 | v2 API                                                    |
| -------------------------------------- | --------------------------------------------------------- |
| `View() string`                        | `View() tea.View`                                         |
| `return renderedString`                | `return tea.NewView(renderedString)`                      |
| `tea.WithAltScreen()` (program option) | `v.AltScreen = true` (on `tea.View`)                      |
| `tea.KeyMsg` (struct)                  | `tea.KeyPressMsg` (interface)                             |
| `msg.Type`, `msg.Runes`                | `msg.Code` (rune), `msg.Text` (string)                    |
| `msg.Alt`                              | `msg.Mod.Contains(tea.ModAlt)`                            |
| `lipgloss.AdaptiveColor{Light, Dark}`  | `lipgloss.LightDark(hasDarkBG)(lightColor, darkColor)`    |
| (no equivalent)                        | `tea.BackgroundColorMsg` with `msg.IsDark()`              |
| `viewport.New(w, h)` (constructor)     | `viewport.New()` then `.SetWidth(w)`, `.SetHeight(h)`     |
| `vp.Width = w` (direct field)          | `vp.SetWidth(w)` (setter)                                 |
| `vp.Height` (direct field)             | `vp.Height()` (getter)                                    |
| `help.New()` (auto-detects theme)      | `help.New()` + `help.DefaultDarkStyles()`/`LightStyles()` |
| `tea.Sequentially(...)`                | `tea.Sequence(...)`                                       |
| `lipgloss.Color(204)` (int)            | `lipgloss.Color("204")` (string only)                     |

## Verification

1. `make test`: all existing tests pass (content assertions like "SNAPPY", "LOCAL SNAPSHOTS", "[s]" still present; bookend test updated)
2. `make lint`: no linter violations
3. `make build && ./bin/snappy`: visually verify:
   - TUI fills the full terminal height and width
   - Snapshot list and log panel share flexible space equally (1:1)
   - Both panels grow when terminal is resized taller
   - Scrolling works (j/k or arrow keys) in the focused panel
   - Tab switches focus between snapshot list and log; focused panel has highlighted border
   - Log auto-scrolls to bottom on new entries
   - Rounded bordered panels render correctly
   - Spinner appears when pressing 's' (create snapshot) and 'r' (refresh)
   - Spinner stops when the operation completes
   - Help bar shows key bindings in Charm style (including scroll and tab)
   - Auto-snapshot toggle (press 'a') updates the indicator (●/○)
   - Unicode flags (◇/◆/⚠) display correctly for APFS snapshots
   - Colors look reasonable in both light and dark terminal themes
