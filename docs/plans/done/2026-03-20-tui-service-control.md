# 2026-03-20 TUI Service Control via Smart Toggle

## Context

The TUI currently detects a running daemon (via lock probing) but cannot control
the launchd service. Pressing 'a' when a daemon is active just logs a warning.
The CLI already has `snappy service install|uninstall|start|stop|status|log`.
This plan adds a "smart toggle" to the 'a' key so the TUI can start/stop the
service when it is installed, falling back to TUI auto-snap toggle when it is
not.

## Behavior

When the launchd service **is installed**:

- 'a' toggles the service (start if stopped, stop if running)
- Help bar shows "start service" or "stop service" contextually
- Info panel shows "Service: (running/stopped)" instead of "Auto-snapshot:"

When the launchd service is **not installed**:

- 'a' works as before (toggle TUI auto-snaps)
- No behavioral change

Service installation status is checked at startup and on each refresh tick
(~60s), so changes made via CLI are picked up automatically.

## Files to Modify

### 1. `internal/logger/logger.go`

Add `CatService Category = "SERVICE"` and register it in `knownCategories`.

### 2. `internal/tui/service.go` (new file)

Testability interface and real implementation:

```go
type ServiceController interface {
    Status(label string) (*service.Info, error)
    Start(label string) error
    Stop(label string) error
}

type LaunchdController struct{}
// Methods delegate to service.Status, service.Start, service.Stop
```

### 3. `internal/tui/model.go`

New fields on `Model`:

```go
serviceCtrl      ServiceController
serviceInstalled bool
serviceRunning   bool
serviceLabel     string
serviceToggling  bool   // debounce: true while start/stop in flight
```

New fields on `ModelParams`:

```go
ServiceCtrl      ServiceController
ServiceInstalled bool
ServiceRunning   bool
```

In `NewModel`:

- Store service fields from params
- Set `serviceLabel = service.DefaultLabel`
- Call `updateAutoSnapHelpText()` to set initial help text

New method `updateAutoSnapHelpText()`:

- If service installed + running: `m.keys.AutoSnap.SetHelp("a", "stop service")`
- If service installed + stopped: `m.keys.AutoSnap.SetHelp("a", "start service")`
- If not installed: `m.keys.AutoSnap.SetHelp("a", "auto-snap")` (default)

### 4. `internal/tui/messages.go`

```go
type ServiceStatusResultMsg struct {
    Info *service.Info
    Err  error
}

type ServiceToggleResultMsg struct {
    Action string // "start" or "stop"
    Err    error
}
```

### 5. `internal/tui/commands.go`

```go
func doServiceStatus(ctrl ServiceController, label string) tea.Cmd
func doServiceStart(ctrl ServiceController, label string) tea.Cmd
func doServiceStop(ctrl ServiceController, label string) tea.Cmd
```

Each wraps the corresponding `ctrl` method call, returning the result message.

### 6. `internal/tui/update.go`

**Add cases to `Update` switch:**

- `ServiceStatusResultMsg` -> `handleServiceStatusResult`
- `ServiceToggleResultMsg` -> `handleServiceToggleResult`

**`handleServiceStatusResult`:**

- Update `serviceInstalled`, `serviceRunning` from `msg.Info`
- Log state transitions (installed/uninstalled, started/stopped)
- Call `updateAutoSnapHelpText()`

**`handleServiceToggleResult`:**

- Clear `serviceToggling`
- On success: optimistically set `serviceRunning`, log result
- On error: log error, trigger a `doServiceStatus` to get real state
- Always follow up with `doServiceStatus` to confirm + get PID

**Replace `handleAutoSnapToggle`:**

```go
func (m Model) handleAutoSnapToggle() (tea.Model, tea.Cmd) {
    if m.serviceInstalled && m.serviceCtrl != nil {
        return m.handleServiceToggle()
    }
    // ... existing TUI auto-snap toggle (unchanged) ...
}
```

**New `handleServiceToggle`:**

- If `serviceToggling`: return (debounce)
- If running: set toggling, log "Stopping service...", return `doServiceStop`
- If stopped: set toggling, log "Starting service...", return `doServiceStart`

**Modify `handleTick`:**

- Add `doServiceStatus(m.serviceCtrl, m.serviceLabel)` to the command batch
  (piggybacks on the existing refresh tick for periodic checks)

**Add to `logEntryStyle`:**

- `CatService` -> `s.textCyan` (same color as auto-snap entries)

### 7. `internal/tui/view.go`

**Modify `formatAutoStatus`:**

- When `serviceInstalled`: show "Service: (running/stopped)" with config params
- Keep existing `daemonActive` branch for non-service daemons (e.g., another TUI)
- Keep existing on/off branch for TUI auto-snaps

**Modify `buildDotIndicator`:**

- Include `serviceInstalled && serviceRunning` in the "active" condition

### 8. `cmd/root.go`

In `runTUI`, before creating the model:

```go
svcCtrl := tui.LaunchdController{}
var svcInstalled, svcRunning bool
if svcInfo, err := service.Status(service.DefaultLabel); err == nil {
    svcInstalled = svcInfo.Installed
    svcRunning = svcInfo.Running
}
if svcRunning {
    daemonActive = true
}
```

Pass to `ModelParams`:

```go
ServiceCtrl:      svcCtrl,
ServiceInstalled: svcInstalled,
ServiceRunning:   svcRunning,
```

### 9. `internal/tui/model_test.go`

Add a `mockServiceController` with configurable `statusFn`, `startFn`, `stopFn`.

New test cases:

- `TestAutoToggleControlsServiceWhenInstalled`: 'a' stops running service
- `TestAutoToggleStartsServiceWhenStopped`: 'a' starts stopped service
- `TestAutoToggleFallsBackWhenServiceNotInstalled`: existing behavior preserved
- `TestAutoToggleIgnoredWhileServiceToggling`: debounce check
- `TestServiceToggleErrorLogsAndRechecks`: error handling
- `TestServiceStatusResultUpdatesState`: state transitions
- `TestHelpTextChangesWithServiceState`: dynamic help text
- `TestViewAutoStatusServiceRunning`: view renders "Service: running"
- `TestViewAutoStatusServiceStopped`: view renders "Service: stopped"
- `TestRefreshTickTriggersServiceStatusCheck`: periodic check fires
- Existing auto-toggle tests pass unchanged (no `ServiceCtrl` set)

## State Interaction

`serviceInstalled` and `serviceRunning` are orthogonal to `daemonActive`:

- `serviceInstalled` determines which code path `handleAutoSnapToggle` takes
- `daemonActive` (lock-based, fast) drives TUI auto-snap suppression
- `serviceRunning` (launchctl-based, async) drives display and toggle direction

After stopping the service, the process exits, releasing its flock.
`syncDaemonState` detects this on the next tick, clearing `daemonActive`. This
is the "corrective" half of the optimistic + corrective model.

## Verification

1. `make build` compiles cleanly
2. `make test` passes all existing + new unit tests
3. `make test-scrut` passes CLI snapshot tests
4. `make lint` passes
5. Manual testing:
   - With service installed and running: launch TUI, verify "Service: running"
     shown, press 'a', verify service stops, press 'a' again, verify it starts
   - With service not installed: launch TUI, verify 'a' toggles TUI auto-snaps
   - Install service via CLI while TUI is open: verify TUI picks up the change
     within ~60s and 'a' switches to service control mode
