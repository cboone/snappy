# TUI Refresh When Daemon Active

## Context

When the snappy TUI is open and another process (another TUI instance or the
background service) is running autosnapshot, the TUI should update every second
to keep ages fresh and detect new snapshots promptly.

Currently it does not because the 1-second UITick is gated on
`m.auto.Enabled() || m.loading` (update.go:96). When a daemon is detected, the
TUI disables its own auto-snapshots (`autoEnabled = false`), so UITick either
never starts or stops at its next fire. The display then only refreshes every
60 seconds via RefreshTick, making the UI feel frozen.

## Changes

### 1. Start UITick when daemon is active at startup

**File:** `internal/tui/model.go` (line 247)

Change the Init() condition from:

```go
if m.auto.Enabled() {
```

to:

```go
if m.auto.Enabled() || m.daemonActive {
```

### 2. Keep UITick running when daemon is active

**File:** `internal/tui/update.go` (line 96, `handleUITick`)

Change the reschedule condition from:

```go
if m.auto.Enabled() || m.loading {
```

to:

```go
if m.auto.Enabled() || m.daemonActive || m.loading {
```

### 3. Start UITick when daemon is detected mid-session

**File:** `internal/tui/update.go` (line 435, `handleTick`)

Capture `daemonActive` before `syncDaemonState` and start UITick if it
transitioned to true:

```go
func (m Model) handleTick() (tea.Model, tea.Cmd) {
    now := m.now()
    wasDaemonActive := m.daemonActive
    m.syncDaemonState(now)

    var cmds []tea.Cmd

    if m.daemonActive && !wasDaemonActive {
        cmds = append(cmds, uiTick())
    }

    // ... rest unchanged ...
```

When the daemon disappears, UITick stops naturally at its next fire because
`handleUITick` checks `m.daemonActive`.

### 4. Trigger faster data refresh while daemon is active

UITick only updates age strings; it does not fetch new data. To detect
externally-created snapshots sooner than the 60-second RefreshTick, trigger a
data refresh every 5 UITicks (5 seconds) when `daemonActive` is true.

**File:** `internal/tui/model.go` -- add field to Model struct (near line 110):

```go
daemonRefreshCount int
```

**File:** `internal/tui/update.go` -- expand `handleUITick`:

```go
func (m Model) handleUITick() (tea.Model, tea.Cmd) {
    m.updateSnapAges()

    var cmds []tea.Cmd

    if m.daemonActive {
        m.daemonRefreshCount++
        if m.daemonRefreshCount >= 5 && !m.refreshing {
            m.daemonRefreshCount = 0
            m.refreshing = true
            cmds = append(cmds, doRefresh(m.runner, m.apfsVolume, m.apfsContainer))
        }
    }

    if m.auto.Enabled() || m.daemonActive || m.loading {
        cmds = append(cmds, uiTick())
    }

    if len(cmds) == 0 {
        return m, nil
    }
    return m, tea.Batch(cmds...)
}
```

**File:** `internal/tui/update.go` -- reset counter in `syncDaemonState`
(line 490, the `!lockHeld && m.daemonActive` case):

```go
m.daemonRefreshCount = 0
```

The existing `m.refreshing` guard prevents concurrent refreshes from the 60s
RefreshTick and the 5s daemon refresh.

## Files to modify

| File | Change |
| --- | --- |
| `internal/tui/model.go` | Add `daemonRefreshCount` field; update `Init()` condition |
| `internal/tui/update.go` | Update `handleUITick()`, `handleTick()`, `syncDaemonState()` |

## Tests

**File:** `internal/tui/model_test.go`

- **TestUITickContinuesWhenDaemonActive**: Set `daemonActive = true`, auto
  disabled, not loading. Send UITickMsg. Expect non-nil cmd (uiTick
  rescheduled).
- **TestUITickStopsWhenDaemonDeactivates**: Set `daemonActive = false`, auto
  disabled, not loading. Send UITickMsg. Expect nil cmd.
- **TestDaemonDetectionStartsUITick**: Simulate `handleTick` where
  `syncDaemonState` transitions `daemonActive` from false to true. Expect
  uiTick in returned cmds.
- **TestDaemonModeTriggersPeriodicRefresh**: With `daemonActive = true`, send 5
  UITickMsgs. Expect a refresh cmd on the 5th tick but not on ticks 1-4.
- Verify existing `TestUITickStopsWhenAutoDisabledAndIdle` still passes
  (it should, since `daemonActive` defaults to false in test models).

## Verification

1. `make test` -- all existing and new unit tests pass
2. `make build` -- builds cleanly
3. Manual: start `snappy run` in one terminal, open TUI in another. Confirm:
   - AGE column updates every second
   - New snapshots appear within ~5 seconds of creation
   - Status shows green dot with "service" label
   - When `snappy run` is stopped, TUI logs the deactivation and UITick stops
     (ages freeze until next manual action or RefreshTick)
