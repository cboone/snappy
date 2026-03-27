## Branch Review: feature/add-more-service-controls-to-tui

Base: main (merge base: 897dd8f)
Commits: 10
Files changed: 9 (2 added, 7 modified, 0 deleted, 0 renamed)
Reviewed through: 99f72c1

### Summary

This branch adds service install and uninstall controls to the TUI, completing the service management story so users can set up and tear down the background launchd service without leaving the TUI. The `i` key acts as a context-aware toggle: installing the service when none exists and uninstalling when one is present. The implementation includes a full plan document, 12 new test cases, and a minor refactor extracting `handleQuit` from `handleKey`.

### Changes by Area

**TUI Service Interface** (`internal/tui/service.go`)
Extended `ServiceController` with `Install`, `Uninstall`, and `ResolveBinaryPath` methods. Added corresponding implementations on `LaunchdController` that delegate to the `service` package.

**TUI Message and Command Layer** (`internal/tui/messages.go`, `internal/tui/commands.go`)
Added `ServiceInstallResultMsg` and `ServiceUninstallResultMsg` message types, along with `doServiceInstall` and `doServiceUninstall` command factories that invoke the controller methods asynchronously.

**TUI Model** (`internal/tui/model.go`)
Added `ServiceInstall` key binding (disabled by default, enabled when a `ServiceController` is present). Added `configFile` field to `Model` and `ConfigFile` to `ModelParams`. Added `updateServiceInstallHelpText()` helper for dynamic "install"/"uninstall" label. Updated `ShortHelp()` and `FullHelp()` to include the new binding.

**TUI Update Logic** (`internal/tui/update.go`)
Added `handleServiceInstallToggle()` with install/uninstall branching, `handleServiceInstallResult()`, and `handleServiceUninstallResult()`. Install path resolves binary, disables local auto-snap, builds `PlistConfig`, and triggers follow-up status/refresh/tick. Uninstall clears all service state and triggers delayed status recheck. Extracted `handleQuit()` from `handleKey()` to reduce cyclomatic complexity.

**CLI Config Passthrough** (`cmd/root.go`)
Added `resolveConfigFile()` helper with tilde expansion and `filepath.Abs` logic. Passes the resolved path as `ConfigFile` in `ModelParams`.

**Tests** (`internal/tui/model_test.go`)
Extended `mockServiceController` with `installFn`, `uninstallFn`, `resolveFn`. Added 12 test cases covering: install/uninstall key behavior, disabled-without-controller guard, toggling-in-progress guard, success/error result handling, auto-snap handoff on install, help text updates, binary resolve failure, and config file passthrough.

**Documentation** (`docs/plans/todo/2026-03-27-tui-service-install-uninstall.md`)
Added a detailed plan document covering context, design, keybinding, state transitions, file changes, edge cases, and verification steps.

**Configuration** (`.claude/settings.json`)
Updated permission allowlist (housekeeping).

### File Inventory

**New files (2):**
- `docs/plans/todo/2026-03-27-tui-service-install-uninstall.md`
- `internal/tui/service.go` (extended, but already existed; diff shows modifications)

**Modified files (7):**
- `.claude/settings.json`
- `cmd/root.go`
- `internal/tui/commands.go`
- `internal/tui/messages.go`
- `internal/tui/model.go`
- `internal/tui/model_test.go`
- `internal/tui/update.go`

### Notable Changes

- **Interface extension**: `ServiceController` gains three new methods, which is a breaking change for any external implementations (though currently only `LaunchdController` and test mocks implement it).
- **Config file passthrough**: The resolved config file path is now threaded from CLI flags through `ModelParams` to the TUI model, enabling correct plist generation at install time.
- **Auto-snap handoff**: The install path disables local auto-snap and releases the flock, cleanly handing off snapshot responsibility to the launchd service.

### Plan Compliance

**Plan**: `docs/plans/todo/2026-03-27-tui-service-install-uninstall.md`

**Compliance verdict**: Full compliance. Every file change, state transition, edge case, and test case specified in the plan is implemented faithfully. The implementation matches both the letter and the spirit of the plan.

**Overall progress**: 7/7 file change items done (100%)

**Done items:**

1. **`internal/tui/service.go`** - Interface extended with `Install`, `Uninstall`, `ResolveBinaryPath`. `LaunchdController` implements all three, delegating to the `service` package. Matches plan exactly.

2. **`internal/tui/messages.go`** - `ServiceInstallResultMsg` and `ServiceUninstallResultMsg` added with `Err error` field. Matches plan exactly.

3. **`internal/tui/commands.go`** - `doServiceInstall(ctrl, cfg)` and `doServiceUninstall(ctrl, label)` added. Signatures and return types match plan.

4. **`internal/tui/model.go`** - `ServiceInstall` key binding added to `keyMap` (disabled by default), included in `ShortHelp()` (between AutoSnap and OpenLog) and `FullHelp()` row 1. `configFile` added to `Model`, `ConfigFile` added to `ModelParams`. `NewModel()` stores `configFile`, enables binding when `serviceCtrl != nil`, calls `updateServiceInstallHelpText()`. Helper implemented. All matches plan.

5. **`internal/tui/update.go`** - `Update()` dispatch extended. `handleKey()` routes `i` to `handleServiceInstallToggle()`. Install/uninstall result handlers implemented. `handleServiceStatusResult()` calls `updateServiceInstallHelpText()`. All matches plan.

6. **`cmd/root.go`** - `resolveConfigFile()` added with tilde expansion and `filepath.Abs`. Passed as `ConfigFile` in `ModelParams`. Matches plan.

7. **`internal/tui/model_test.go`** - All 12 specified test cases implemented:
   - `TestServiceInstallKeyWhenNotInstalled`
   - `TestServiceInstallKeyWhenInstalled`
   - `TestServiceInstallKeyDisabledWithoutController`
   - `TestServiceInstallKeyIgnoredWhileToggling`
   - `TestServiceInstallResultSuccess`
   - `TestServiceInstallResultError`
   - `TestServiceUninstallResultSuccess`
   - `TestServiceUninstallResultError`
   - `TestServiceInstallDisablesLocalAutoSnap`
   - `TestServiceInstallHelpTextUpdates`
   - `TestServiceInstallBinaryResolveFails`
   - `TestServiceInstallPassesConfigFile`

**State transitions verified:**

- Install path steps 1-6: All implemented correctly in `handleServiceInstallToggle()` and `handleServiceInstallResult()`.
- Uninstall path steps 1-3: All implemented correctly in `handleServiceInstallToggle()` and `handleServiceUninstallResult()`.

**Edge cases verified:**

- Binary path resolution failure: Logs error, clears `serviceToggling`, returns no command.
- Auto-snapshot in flight: Uses `lockReleasePending` mechanism.
- External install/uninstall: `updateServiceInstallHelpText()` called in `handleServiceStatusResult()`.
- Post-uninstall: `serviceEverInstalled` cleared in `handleServiceUninstallResult()`.

**Deviations**: None. The implementation follows the plan precisely.

**Fidelity concerns**: None. The implementation matches the plan's intent throughout.

**Note**: The plan file is still in `docs/plans/todo/`. It should be moved to `docs/plans/done/` since the work is complete.

### Code Quality Assessment

**Overall quality**: This code is ready to merge. The implementation is clean, thorough, and follows established codebase patterns precisely.

**Strengths:**

- **Pattern consistency**: The install/uninstall handlers follow the exact same structure as the existing start/stop handlers (`handleServiceToggleResult`). Message types, command factories, and update dispatch all mirror existing conventions.
- **Thorough test coverage**: 12 tests covering happy paths, error paths, guard conditions, state transitions, and integration points (config passthrough, auto-snap handoff). Tests exercise the command return values, not just model state.
- **Clean state management**: The install success path correctly sets all related flags (`serviceInstalled`, `serviceRunning`, `serviceEverInstalled`, `daemonActive`) and triggers the right follow-up actions (status recheck, data refresh, UI tick). The uninstall path correctly clears all state including `serviceEverInstalled` and `daemonRefreshCount`.
- **Good commit hygiene**: Small, focused commits with clear conventional commit messages. Each commit is logically self-contained (interface first, then messages, then commands, then handlers, then tests, then the wiring commit).
- **Edge case handling**: Binary resolve failure is handled gracefully. Lock release handles both in-flight and idle auto-snapshot states. The `serviceToggling` guard prevents double-presses.
- **The handleQuit refactor** in its own commit is a clean improvement that reduces cyclomatic complexity without behavioral change.

**Issues to address:**

- **Duplicated config resolution logic**: `resolveConfigFile()` in `cmd/root.go` duplicates the tilde-expansion and `filepath.Abs` logic from `cmd/service.go`'s `runServiceInstall`. This is a minor DRY concern; both call sites could share a helper. Low severity since the logic is simple and unlikely to diverge, but worth noting.

**Suggestions:**

- Consider moving the plan from `docs/plans/todo/` to `docs/plans/done/` as part of the merge commit or a follow-up.
- The `resolveConfigFile()` tilde expansion could be extracted to a shared utility if a third call site appears, but two instances is acceptable for now.
