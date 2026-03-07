# Focus Flash Border Effect

## Context

The TUI currently switches panel border colors instantly when focus changes (subtle gray to white/black). This plan adds a gentle "gem-like flash" animation: a bright beam of light sweeps across the horizontal borders of a panel when it gains focus, then settles to the normal focused border color. The effect is exploratory, to see how it looks and feels.

## Design

### Effect Description

When a panel gains focus, a bright highlight beam sweeps diagonally from the top-left corner to the bottom-right corner over ~500ms (10 frames at 50ms). All four borders participate: each border character gets an individually computed color based on its distance from the moving diagonal beam front. The beam appears as a bright band crossing the border on a slant.

### Diagonal Sweep Model

Each border character at position (x, y) is assigned a normalized diagonal value:

```text
d = x_norm + y_norm
```

Where `x_norm = x / (width - 1)` and `y_norm = y / (height - 1)`. This gives `d` in the range [0, 2], with the top-left corner at d=0 and bottom-right corner at d=2. The beam center sweeps from d=-overshoot to d=2+overshoot across frames.

At any given frame:

- **Top border** (y=0): d = x_norm, so the beam appears at a certain x position
- **Bottom border** (y=H-1): d = x_norm + 1, so the beam is shifted ~width positions to the left
- **Left side** (x=0): d = y_norm, beam travels top to bottom
- **Right side** (x=W-1): d = 1 + y_norm, beam travels top to bottom but delayed

This creates a convincing diagonal sweep from corner to corner.

### Color Palette

- **Dark mode**: beam sweeps gold (`#FFD700`) across a white base
- **Light mode**: beam sweeps royal blue (`#4169E1`) across a black base
- Gaussian-like falloff around the beam center in d-space
- Beam starts slightly off-screen (d < 0) and exits off-screen (d > 2), ensuring smooth entry/exit

### Rendering Approach During Flash

During a flash, the focused panel bypasses Lipgloss border rendering entirely:

1. Content is rendered with a borderless style (padding only, same content width)
2. All four borders are built manually with per-character ANSI coloring
3. Left/right side `│` characters are colored per-line based on their y position's diagonal value
4. Top/bottom `─` characters are colored per-column based on their x position's diagonal value
5. This avoids fighting with Lipgloss internals and gives full control over every border character

Width calculation: with `Padding(0, 1).Width(cw + 2)` (no border), the content area is `cw` chars wide, same as with the bordered style. Adding manual `│` on each side gives total width `cw + 4`, matching the normal rendering.

### Animation Lifecycle

1. Focus changes via Tab, mouse click, or mouse wheel
2. `setFocusPanel()` starts flash (resets state, returns `flashTick()` command)
3. Each `FlashTickMsg` advances frame counter and schedules the next tick
4. `View()` renders all four borders with per-character diagonal gradient during flash
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
- `flashDiagonalValue(x, y, width, height int) float64`: computes the normalized diagonal value `d = x_norm + y_norm` for a border character at (x, y)
- `flashCharColor(d, beamCenter float64, base, highlight colorful.Color) colorful.Color`: computes a single character's color based on its diagonal distance from the beam center, using Gaussian falloff and `BlendLab()`
- `flashBeamCenter(frame, totalFrames int) float64`: computes the beam center position in d-space for the current frame, sweeping from -0.15 to 2.15
- `renderFlashBorders(body, title string, width, height int, flash flashState, styles modelStyles) string`: renders content with all four borders colored per-character for the diagonal sweep. Bypasses Lipgloss border rendering entirely during flash:
  1. Renders content with padding only (no border) using a borderless style
  2. Builds the top border with per-character coloring and embedded title
  3. Builds each content line with individually colored left and right `│` characters
  4. Builds the bottom border with per-character coloring
  5. Assembles the final output
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

- In `renderInfoPanel`, `renderSnapshotPanel`, `renderLogPanel`: when `m.flash.active && m.flash.panel == thisPanel`, call `renderFlashBorders()` (which handles all four borders with diagonal sweep) instead of the normal `style.Render()` + `borderTitle()` path
- The non-flash rendering path is unchanged

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
