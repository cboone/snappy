## Branch Review: feature/add-help

Base: main (merge base: 226df88)
Commits: 1
Files changed: 4 (1 added, 3 modified, 0 deleted, 0 renamed)
Reviewed through: c644057

### Summary

This branch adds an in-TUI help toggle activated by the `?` key, addressing issue #56. When pressed, the help bar expands from a single-line compact view (primary actions only) to a full multi-column display showing all keybindings, including previously hidden navigation keys. The snapshot and log panels dynamically resize to accommodate the expanded help. A layout refactor extracts `recalcLayout()` and `helpBarHeight()` to eliminate hardcoded height assumptions, and an `handleOpenLog()` method is split out of `handleKey()` for cyclomatic complexity.

### Changes by Area

**TUI / Keybindings** (`internal/tui/model.go`)

Added `Help key.Binding` field to `keyMap`, wired into `defaultKeyMap()` with `?` as the trigger key. `ShortHelp()` now appends the Help binding. `FullHelp()` adds it to the navigation group, making both groups 5 items tall for a balanced two-column layout.

**TUI / Layout and Update Logic** (`internal/tui/update.go`)

- Extracted `recalcLayout()` from `handleWindowSize()`, replacing the hardcoded help bar height of 1 with `helpBarHeight()` which returns 5 when help is expanded.
- Added `handleHelpToggle()`: toggles `help.ShowAll`, updates the help description between "help" and "close help", and calls `recalcLayout()`.
- Extracted `handleOpenLog()` from inline code in `handleKey()` for cyclomatic complexity reduction.

**Tests** (`internal/tui/model_test.go`)

Added 4 test functions: `TestHelpToggle` (toggle on/off and description changes), `TestHelpToggleRelayoutsPanels` (panels shrink when help expands), `TestViewShowsHelpHint` (`?` appears in default view), `TestFullHelpShowsAllBindings` (navigation bindings visible when expanded).

**Documentation** (`docs/plans/todo/2026-03-24-add-in-tui-help.md`)

Plan document describing the feature design and implementation approach.

### File Inventory

- **New files (1):** `docs/plans/todo/2026-03-24-add-in-tui-help.md`
- **Modified files (3):** `internal/tui/model.go`, `internal/tui/update.go`, `internal/tui/model_test.go`
- **Deleted files (0):** none
- **Renamed files (0):** none

### Notable Changes

- The plan file is in `docs/plans/todo/` despite the work being complete. It should likely be moved to `docs/plans/done/` or equivalent.

### Plan Compliance

**Plan:** `docs/plans/todo/2026-03-24-add-in-tui-help.md`

**Compliance verdict:** Excellent compliance. The implementation follows the plan faithfully in both letter and spirit. Every plan item is fully implemented, and the code matches the plan's specified approaches precisely.

**Overall progress:** 4/4 items done (100%)

**Done items:**

1. **Add `Help` binding to `keyMap`** (model.go): Done. `Help` field added after `ShiftTab` as specified. `defaultKeyMap()` uses the exact binding definition from the plan. `ShortHelp()` appends `k.Help` as last item. `FullHelp()` appends `k.Help` to the navigation group, creating two groups of 5 for a clean rectangular layout. All sub-items complete.

2. **Extract layout helper and add help height logic** (update.go): Done. `fullHelpHeight = 5` constant defined. `helpBarHeight()` method returns `fullHelpHeight` when `ShowAll` is true, 1 otherwise. Layout logic extracted into `recalcLayout()` using `m.helpBarHeight()` instead of the hardcoded 1. `handleWindowSize()` simplified to set dimensions and call `recalcLayout()`. All sub-items complete.

3. **Add `?` key handler** (update.go): Done. New case in `handleKey` matches `m.keys.Help`, delegates to `handleHelpToggle()`. The toggle logic matches the plan: flips `ShowAll`, updates help text between "help"/"close help", calls `recalcLayout()`, returns nil cmd. Placement is after `ShiftTab` and before `ScrollUp`/`ScrollDown` as specified.

4. **Add tests** (model_test.go): Done. All 4 test functions implemented as described:
   - `TestHelpToggle`: Verifies `ShowAll` toggle and description changes.
   - `TestHelpToggleRelayoutsPanels`: Sends `WindowSizeMsg`, verifies `snapVisibleRows` decreases.
   - `TestViewShowsHelpHint`: Checks `?` appears in rendered view.
   - `TestFullHelpShowsAllBindings`: Verifies navigation descriptions appear in full help.

**Deviations:**

1. **`handleOpenLog()` extraction (minor scope addition):** The plan did not mention extracting `handleOpenLog()` from inline code in `handleKey()`. The commit message explains this was done to stay within cyclomatic complexity limits. This is a reasonable, justified deviation: it keeps the linter happy without changing behavior.

2. **Handler delegation pattern:** The plan showed the help toggle logic inline in the `handleKey` switch case, but the implementation delegates to `handleHelpToggle()`. This follows the existing pattern in the codebase (`handleAutoSnapToggle()`, `handleScroll()`) and is consistent with the `handleOpenLog()` extraction. A reasonable approach deviation.

**Fidelity concerns:** None. The implementation closely matches the plan's design intent.

### Code Quality Assessment

**Overall quality:** This code is ready to merge. It is clean, well-structured, and follows the existing codebase patterns precisely.

**Strengths:**

- **Consistent with codebase patterns.** The `handleHelpToggle()` method follows the same signature and structure as `handleAutoSnapToggle()` and other handler methods. The pointer receiver on `recalcLayout()` and `helpBarHeight()` is appropriate since they mutate or read mutable state.
- **Clean refactoring.** The `recalcLayout()` extraction is well-scoped. It removes duplication that would have existed between `handleWindowSize` and `handleHelpToggle`, and the `helpBarHeight()` helper keeps the height logic in one place.
- **Good test coverage.** Four focused tests cover the core behaviors: toggle state, layout recalculation, UI rendering of the hint, and full help content. The tests use the existing `testModel()` and `viewContent()` helpers, maintaining consistency with the rest of the test file.
- **Minimal footprint.** No new files beyond the plan document. No changes needed to `view.go`, `styles.go`, `messages.go`, or `commands.go`, just as the plan predicted.
- **Descriptive commit message.** The commit message explains both the feature and the ancillary refactoring, with the issue reference.

**Issues to address:**

1. **Plan file location.** The plan is in `docs/plans/todo/` but the work is complete and merged. It should be moved to `docs/plans/done/` (or whichever directory this project uses for completed plans) to reflect its status.

**Suggestions:**

- None. The implementation is focused and complete.
