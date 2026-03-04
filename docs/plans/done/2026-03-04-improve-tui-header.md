# 2026-03-04: Improve TUI Header

## Context

The TUI header currently shows `● SNAPPY vv0.5.0-33-ge07bd1c  Time Machine Local Snapshot Manager`. Three problems:

1. **Double "v"**: `git describe --tags` returns `v0.5.0-...` and `renderTitleBar` prepends another `v`
2. **Static indicator**: The `●` dot is always `indicatorOn`, never changes
3. **Wasted space**: The tagline "Time Machine Local Snapshot Manager" provides no dynamic information

The goal is a cleaner, more informative header: just the app name, a meaningful status dot, and the current operation when busy.

## Design

The new header will look like:

```text
● SNAPPY                        (auto on, idle - green dot)
● SNAPPY  Refreshing ⣾          (auto on, busy - yellow dot)
○ SNAPPY                        (auto off, idle - green dot)
○ SNAPPY  Thinning ⣾            (auto off, busy - yellow dot)
```

- **Dot shape**: `●` when auto-snapshot is enabled, `○` when disabled (reuses existing `indicatorOn`/`indicatorOff` constants)
- **Dot color**: green when idle, yellow when busy (reuses existing `textGreen`/`textYellow` styles from `styles.go`)
- **Operation label**: "Refreshing", "Snapshotting", or "Thinning" with spinner, shown only when `m.loading` is true
- **Version and tagline**: removed entirely from the header

## Changes

### 1. `internal/tui/view.go` - `renderTitleBar()`

Replace the current implementation:

```go
func (m Model) renderTitleBar(width int) string {
    title := fmt.Sprintf("%s SNAPPY v%s  Time Machine Local Snapshot Manager", indicatorOn, m.version)
    if m.loading {
        title += "  " + m.spinner.View()
    }
    return m.styles.titleBar.Width(width).Render(title)
}
```

With:

```go
func (m Model) renderTitleBar(width int) string {
    dot := indicatorOff
    if m.auto.Enabled() {
        dot = indicatorOn
    }
    if m.loading {
        dot = m.styles.textYellow.Render(dot)
    } else {
        dot = m.styles.textGreen.Render(dot)
    }

    title := dot + " SNAPPY"

    if m.snapshotting {
        title += "  Snapshotting " + m.spinner.View()
    } else if m.thinning {
        title += "  Thinning " + m.spinner.View()
    } else if m.loading {
        title += "  Refreshing " + m.spinner.View()
    }

    return m.styles.titleBar.Width(width).Render(title)
}
```

Notes:

- `m.loading` is always true when any of `snapshotting`/`refreshing`/`thinning` is true, so it serves as the composite "busy" check for dot color
- Snapshotting and thinning are checked before the generic `m.loading` fallback so the label is specific
- The spinner is already styled via `spinnerStyle` and works inside the title bar (existing behavior)

### 2. `internal/tui/model_test.go` - update title bar assertions

Two tests reference the title bar:

- **`TestViewRendersAllSections`** (line 69): asserts `strings.Contains(v, "SNAPPY")` - still passes, no change needed
- **`TestViewSpinnerDuringLoading`** (line 483): asserts `strings.Contains(withLoading, "SNAPPY")` - still passes, no change needed

Both tests check for `"SNAPPY"` which remains in the header. No test changes required.

### 3. Bonus: `cmd/root.go` line 118 - fix double "v" in startup log

The log line `fmt.Sprintf("snappy v%s | ...")` also double-prefixes the version. Change `v%s` to `%s`.

## Verification

1. `make build && bin/snappy` - visually confirm header:
   - Shows `● SNAPPY` or `○ SNAPPY` (no version, no tagline)
   - Dot is green when idle
   - Press `s` to snapshot: dot turns yellow, "Snapshotting" label + spinner appear
   - Press `r` to refresh: dot turns yellow, "Refreshing" label + spinner appear
   - Press `a` to toggle auto: dot shape changes between `●` and `○`
2. `make test` - all existing tests pass
3. `make lint` - no lint issues
