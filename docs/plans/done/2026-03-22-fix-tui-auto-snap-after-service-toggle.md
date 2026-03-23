# 2026-03-22 Fix TUI Auto-snap Starting After Service Toggle

## Context

When the launchd service is running, the user starts the TUI, stops the service via
'a', then presses 'a' again to restart it, the TUI's own auto-snapshotting starts
instead of the service restarting.

The root cause: `handleAutoSnapToggle()` guards service delegation on
`m.serviceInstalled`, which is updated asynchronously by `handleServiceStatusResult`.
After `Stop()` (disable + kill), a status check can transiently report
`Installed=false` (macOS launchctl timing), clearing `m.serviceInstalled`. The next
'a' press bypasses the service guard (line 244 of update.go) and falls through to the
TUI auto-snap enabling path (line 268), which acquires the free lock and enables TUI
auto-snapshotting.

## Fix: Debounce + Defense-in-Depth

Two complementary changes prevent the fallthrough.

### Change 1: Debounce `serviceInstalled` transitions to false

Require two consecutive status checks reporting `Installed=false` before accepting the
transition. A single transient reading triggers an immediate re-check instead.

**New fields on Model** (`internal/tui/model.go`, near line 118):

```go
serviceConsecutiveUninstalled int  // consecutive status checks reporting not-installed
serviceEverInstalled         bool // sticky: true once service has been seen installed
```

**`NewModel`** (`model.go`, near line 224): initialize `serviceEverInstalled` from
`params.ServiceInstalled`.

**`handleServiceStatusResult`** (`update.go`, lines 338-359): replace the direct
assignment of `m.serviceInstalled = msg.Info.Installed` with:

```go
if msg.Info.Installed {
    m.serviceInstalled = true
    m.serviceRunning = msg.Info.Running
    m.serviceConsecutiveUninstalled = 0
    m.serviceEverInstalled = true
} else {
    m.serviceConsecutiveUninstalled++
    m.serviceRunning = false
    if !wasInstalled || m.serviceConsecutiveUninstalled >= 2 {
        m.serviceInstalled = false
        m.serviceEverInstalled = false
    }
    // First time: keep serviceInstalled=true, trigger immediate re-check
}
```

After the existing log/help-text updates, when this is the first `Installed=false`
after previously being installed, return an immediate `doServiceStatus` re-check
command instead of nil.

### Change 2: Defense-in-depth guard in `handleAutoSnapToggle`

In `handleAutoSnapToggle` (`update.go`), before the TUI auto-snap enabling path
(line 268), add:

```go
if m.serviceEverInstalled && m.serviceCtrl != nil {
    m.log.Log(logger.LevelInfo, logger.CatService, "Service state unclear; rechecking...")
    m.updateLogViewContent()
    return m, doServiceStatus(m.serviceCtrl, m.serviceLabel)
}
```

This prevents TUI auto-snapping from starting when the service was recently installed
but `serviceInstalled` is transiently false. In normal operation, the debounce prevents
reaching this code; it exists as a safety net.

## Files to Modify

1. **`internal/tui/model.go`**
   - Add `serviceConsecutiveUninstalled int` and `serviceEverInstalled bool` fields
   - Initialize `serviceEverInstalled` in `NewModel` from `params.ServiceInstalled`

2. **`internal/tui/update.go`**
   - `handleServiceStatusResult` (line 331): debounce `serviceInstalled` transitions,
     trigger immediate re-check on first `Installed=false`, set/clear `serviceEverInstalled`
   - `handleAutoSnapToggle` (line 268): defense-in-depth guard before TUI auto-snap enabling

3. **`internal/tui/model_test.go`**
   - Update `TestServiceStatusResultDetectsUninstall` (line 3031): now requires two
     consecutive `Installed=false` messages to confirm uninstall
   - Add `TestServiceStatusTransientUninstallDebounced`: single `Installed=false` keeps
     `serviceInstalled=true`, returns re-check command
   - Add `TestAutoToggleBlockedWhenServiceRecentlyInstalled`: `serviceEverInstalled=true`
     with `serviceInstalled=false` blocks TUI auto-snap, triggers status re-check
   - Add `TestAutoToggleAllowedAfterConfirmedUninstall`: after confirmed uninstall
     (both flags false), TUI auto-snap works normally

## Verification

1. `make test` -- all existing and new unit tests pass
2. `make lint` -- no lint issues
3. Manual test:
   - Service installed + running: start TUI, press 'a' (service stops), press 'a'
     (service restarts, NOT TUI auto-snap)
   - Service not installed: 'a' toggles TUI auto-snap as before
   - `snappy service uninstall` in another terminal while TUI is running: TUI picks
     up the uninstall within two status checks (~2 min) and 'a' falls back to TUI
     auto-snap
