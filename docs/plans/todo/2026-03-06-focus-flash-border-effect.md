# Focus Flash Border Effect

## Context

The TUI currently switches panel border colors instantly when focus changes (subtle gray to white/black). This plan adds a gentle "gem-like flash" animation: a bright beam of light sweeps across the horizontal borders of a panel when it gains focus, then settles to the normal focused border color. The effect is exploratory, to see how it looks and feels.

## Design

### Effect Description

When a panel gains focus, a bright highlight spot sweeps left-to-right across the top border and right-to-left across the bottom border over ~500ms (10 frames at 50ms). Each horizontal border character gets an individually computed color based on its distance from the moving beam center. Side borders use the normal focus color throughout.

### Color Palette

- **Dark mode**: beam sweeps gold (`#FFD700`) across a white base
- **Light mode**: beam sweeps royal blue (`#4169E1`) across a black base
- Gaussian-like falloff around the beam center (~25% of border width)
- Beam starts slightly off-screen left and exits off-screen right, ensuring smooth entry/exit

### Animation Lifecycle

1. Focus changes via Tab, mouse click, or mouse wheel
2. `setFocusPanel()` starts flash (resets state, returns `flashTick()` command)
3. Each `FlashTickMsg` advances frame counter and schedules the next tick
4. `View()` renders per-character gradient on top/bottom borders during flash
5. After frame 9, `flash.active` becomes false; rendering returns to normal `sectionFocus` style

### Edge Cases

- **Already-focused panel clicked**: `setFocusPanel` returns nil (no flash)
- **Rapid Tab switching**: resets flash to frame 0 for the new panel
- **Resize during flash**: `flashBorderColors()` takes width as parameter, recomputes
- **Theme change during flash**: `handleBackgroundColor` rebuilds styles; next frame picks up new colors
- **Initial load**: no flash (setFocusPanel not called during NewModel)

## File Changes

### New: `internal/tui/flash.go`

Core animation logic:

- `flashState` struct: `active bool`, `panel int`, `frame int`, `totalFrames int`
- `flashBorderColors(width, frame, totalFrames int, base, highlight colorful.Color) []lipgloss.Color`: computes per-character colors for a horizontal border line using Gaussian falloff around a sweeping beam position
- `borderTitleFlash(rendered, title string, colors []lipgloss.Color) string`: like `borderTitle()` but colors each `─` and corner character individually
- `borderBottomFlash(rendered string, colors []lipgloss.Color) string`: replaces the last line of rendered output with per-character colored bottom border
- Helper to convert lipgloss.Color to colorful.Color for blending

### Modified: `internal/tui/messages.go`

- Add `FlashTickMsg struct{}`

### Modified: `internal/tui/commands.go`

- Add `flashTick()` returning `tea.Tick(50*time.Millisecond, ...)`

### Modified: `internal/tui/styles.go`

- Add `flashHighlight colorful.Color` and `flashBase colorful.Color` fields to `modelStyles`
- Compute from hex colors in `newModelStyles()` based on `hasDarkBG`
- Import `go-colorful` (already an indirect dependency)

### Modified: `internal/tui/model.go`

- Add `flash flashState` field to `Model`

### Modified: `internal/tui/update.go`

- Add `FlashTickMsg` case to `Update()` switch, calling `handleFlashTick()`
- `handleFlashTick()`: advance frame, stop at totalFrames, schedule next tick
- **Refactor `setFocusPanel()` to return `tea.Cmd`**: starts flash when panel changes, returns `flashTick()`. Returns nil if panel unchanged.
- Update all callers to capture and return the command:
  - `handleKey` Tab case (line 203): `cmd := m.setFocusPanel(...); return m, cmd`
  - `handleMouseClick` (lines 222, 234, 242): capture cmd in local var, return it
  - `handleMouseWheel` (lines 249, 259): capture and return cmd

### Modified: `internal/tui/view.go`

- Add `flashColors(width int) (topColors, bottomColors []lipgloss.Color)` method: computes gradient for current frame, sweeping left-to-right for top and right-to-left for bottom
- In `renderInfoPanel`, `renderSnapshotPanel`, `renderLogPanel`: when `m.flash.active && m.flash.panel == thisPanel`, call `borderTitleFlash` + `borderBottomFlash` instead of `borderTitle`

## Reusable Code

- `borderTitle()` in `view.go:171-198`: base structure for the flash variant
- `lipgloss.RoundedBorder()` for border character constants (TopLeft, Top, TopRight, BottomLeft, Bottom, BottomRight)
- `flashTick()` follows the exact pattern of existing `uiTick()` and `refreshTick()` in `commands.go`
- `go-colorful` (`github.com/lucasb-eyer/go-colorful`): already in go.sum as indirect dep, use `BlendLab()` for perceptually smooth color interpolation

## Verification

1. `make build` compiles successfully
2. `make test` passes (setFocusPanel refactor may affect existing tests)
3. `make lint` passes
4. Run the TUI and press Tab repeatedly: observe the flash sweep on each panel's borders
5. Click panels with the mouse: same flash effect
6. Rapid Tab switching: flash restarts cleanly on the new panel
7. Resize terminal during a flash: no visual glitches
8. Test in both light and dark terminal themes
