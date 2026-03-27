# TUI Service Install/Uninstall Controls

## Context

The TUI can start and stop an already-installed launchd service (via the `a` key), but cannot install or uninstall it. Those operations are only available through the CLI (`snappy service install` / `snappy service uninstall`). This means users must leave the TUI to set up or tear down the background service. Adding install/uninstall to the TUI completes the service management story so everything is accessible from a single interface.

## Design

### Keybinding: `i` as context-aware toggle

- When the service is **not installed**: `i` installs and starts the service.
- When the service **is installed**: `i` uninstalls (stops and removes) the service.
- Help text updates dynamically: `"i install"` or `"i uninstall"`.
- The binding starts disabled and is only enabled when `serviceCtrl != nil`.

This mirrors how `a` already overloads between auto-snap toggle and service start/stop. A single contextual key keeps the keymap compact. No confirmation prompt: the codebase pattern is immediate action with log feedback. The operation is easily reversible (press `i` again).

### State transitions

**Install path:**

1. Resolve binary path via `ServiceController.ResolveBinaryPath()`.
2. If local auto-snap is active, disable it and release the lock (service takes over).
3. Build `PlistConfig` from label, binary path, config LogDir, and config file path.
4. Call `ServiceController.Install(cfg)`.
5. On success: set `serviceInstalled=true`, `serviceRunning=true`, `daemonActive=true`. Trigger status recheck and data refresh. Start UI tick.
6. On error: log, trigger status recheck, clear `serviceToggling`.

**Uninstall path:**

1. Call `ServiceController.Uninstall(label)`.
2. On success: clear `serviceInstalled`, `serviceRunning`, `daemonActive`, `serviceEverInstalled`. Trigger delayed status recheck.
3. On error: log, trigger status recheck, clear `serviceToggling`.

The existing `serviceToggling` flag prevents double-presses during install/uninstall, just as it does for start/stop.

## File Changes

### 1. `internal/tui/service.go`

Extend the `ServiceController` interface:

```go
type ServiceController interface {
    Status(label string) (*service.Info, error)
    Start(label string) error
    Stop(label string) error
    Install(cfg service.PlistConfig) error
    Uninstall(label string) error
    ResolveBinaryPath() (string, error)
}
```

Add the three new methods to `LaunchdController`, delegating to the `service` package.

### 2. `internal/tui/messages.go`

Add two new message types:

```go
type ServiceInstallResultMsg struct{ Err error }
type ServiceUninstallResultMsg struct{ Err error }
```

### 3. `internal/tui/commands.go`

Add two new command factories:

- `doServiceInstall(ctrl, cfg)` -- calls `ctrl.Install(cfg)`, returns `ServiceInstallResultMsg`.
- `doServiceUninstall(ctrl, label)` -- calls `ctrl.Uninstall(label)`, returns `ServiceUninstallResultMsg`.

### 4. `internal/tui/model.go`

- Add `ServiceInstall key.Binding` to the `keyMap` struct, defaulting to disabled.
- Add it to `ShortHelp()` (between AutoSnap and OpenLog) and `FullHelp()` row 1.
- Add `configFile string` to `Model` and `ConfigFile string` to `ModelParams`.
- In `NewModel()`: store `configFile`, enable `ServiceInstall` when `serviceCtrl != nil`, call new `updateServiceInstallHelpText()`.
- Add `updateServiceInstallHelpText()` helper (sets "install" or "uninstall" based on `serviceInstalled`).

### 5. `internal/tui/update.go`

- Extend `Update()` dispatch to handle `ServiceInstallResultMsg` and `ServiceUninstallResultMsg`.
- Add `key.Matches(msg, m.keys.ServiceInstall)` case in `handleKey()`.
- Add `handleServiceInstallToggle()`: branches on `serviceInstalled` to install or uninstall.
- Add `handleServiceInstallResult(msg)`: on success, set installed/running state, update help texts, trigger status recheck + data refresh + UI tick.
- Add `handleServiceUninstallResult(msg)`: on success, clear installed/running/daemon state, update help texts, trigger delayed status recheck.
- In `handleServiceStatusResult()`: add `m.updateServiceInstallHelpText()` call so the `i` key stays in sync with externally-detected state changes.

### 6. `cmd/root.go`

- In `runTUI()`, resolve the config file path (same expansion logic as `cmd/service.go`'s `runServiceInstall`: handle `~` prefix, `filepath.Abs`).
- Pass it as `ConfigFile` in `ModelParams`.

### 7. `internal/tui/model_test.go`

- Extend `mockServiceController` with `installFn`, `uninstallFn`, `resolveFn` fields and corresponding methods.
- Add tests:
  - `TestServiceInstallKeyWhenNotInstalled` -- triggers install, sets toggling, logs message.
  - `TestServiceInstallKeyWhenInstalled` -- triggers uninstall.
  - `TestServiceInstallKeyDisabledWithoutController` -- no-op when `serviceCtrl` is nil.
  - `TestServiceInstallKeyIgnoredWhileToggling` -- no-op when `serviceToggling`.
  - `TestServiceInstallResultSuccess` -- sets installed/running, clears toggling, updates help.
  - `TestServiceInstallResultError` -- logs error, triggers status recheck.
  - `TestServiceUninstallResultSuccess` -- clears state.
  - `TestServiceUninstallResultError` -- logs error, triggers status recheck.
  - `TestServiceInstallDisablesLocalAutoSnap` -- disables auto-snap, releases lock.
  - `TestServiceInstallHelpTextUpdates` -- help text changes between "install"/"uninstall".
  - `TestServiceInstallBinaryResolveFails` -- aborts with log, clears toggling.
  - `TestServiceInstallPassesConfigFile` -- verifies correct PlistConfig.

## Edge Cases

- **Binary path resolution failure**: Logs error, clears `serviceToggling`, does not dispatch install.
- **Auto-snapshot in flight during install**: Lock release is deferred via existing `lockReleasePending` mechanism.
- **External install/uninstall detected**: Periodic status polling already detects these; adding `updateServiceInstallHelpText()` to `handleServiceStatusResult()` keeps the `i` key label in sync.
- **Post-uninstall**: `serviceEverInstalled` is cleared so the defense-in-depth check in `handleAutoSnapToggle()` does not block re-enabling local auto-snap.

## Verification

1. `make build` -- confirms compilation.
2. `make test` -- runs unit tests including new service install/uninstall tests.
3. `make lint` -- checks for style issues.
4. Manual test in TUI:
   - Launch TUI without service installed: verify `i install` appears in help bar.
   - Press `i`: verify service installs, log shows "Installing service..." then "Service installed and started", help switches to `i uninstall`, info panel shows service running.
   - Press `i` again: verify service uninstalls, log shows "Uninstalling service..." then "Service uninstalled", help switches to `i install`.
   - Press `a` after uninstall: verify local auto-snap re-enables.
   - Press `i` while toggling in progress: verify no-op.
