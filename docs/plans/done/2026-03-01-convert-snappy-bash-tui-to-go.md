# Convert Snappy bash TUI to Go with Charmbracelet

## Context

Snappy is a macOS utility that creates frequent Time Machine local snapshots and
displays them in an interactive terminal UI. The current implementation is an
908-line bash script (`bin/snappy`) that uses `read -t`, `tput`, and `printf` for
its TUI. The Go project scaffold (Cobra, Viper, bubbletea, lipgloss, bubbles) is
already in place.

This plan converts the bash script into idiomatic Go using the Charmbracelet
bubbletea framework. The goal is a faithful conversion that matches the bash
version's behavior and visual output, with improved testability and
maintainability.

## Pre-work

Move the bash reference implementation before any Go code lands:

- Move `bin/snappy` to `docs/proof-of-concept/snappy`
- Update the delegate-retention plan to note that thinning is kept (current behavior is correct)

## Package structure

```text
main.go                              Entry point (exists)
cmd/
  root.go                            Cobra root command (modify to launch TUI)
internal/
  config/config.go                   Viper-backed config struct
  platform/
    exec.go                          CommandRunner interface (for testability)
    tmutil.go                        tmutil command wrappers
    diskutil.go                      diskutil/APFS wrappers + plist parsing
    disk.go                          df disk info wrapper
  snapshot/
    snapshot.go                      Snapshot type, date parsing, relative time
    diff.go                          Snapshot list diffing (added/removed)
    auto.go                          Auto-snapshot timer and thinning logic
  logger/logger.go                   Ring buffer + file logger
  tui/
    model.go                         Bubbletea model + Init
    update.go                        Update function (message dispatch)
    view.go                          View function (render header/list/log/controls)
    styles.go                        Lipgloss style definitions
    messages.go                      Custom tea.Msg types
    commands.go                      tea.Cmd factories for async operations
```

## Phase 1: Config and domain types

Create the foundational types that the rest of the codebase depends on.

### `internal/config/config.go`

```go
type Config struct {
    RefreshInterval      time.Duration  // SNAPPY_REFRESH, default 60s
    MountPoint           string         // SNAPPY_MOUNT, default "/"
    LogDir               string         // SNAPPY_LOG_DIR, default ~/.local/share/snappy
    AutoEnabled          bool           // SNAPPY_AUTO_ENABLED, default true
    AutoSnapshotInterval time.Duration  // 60s (constant)
    ThinAgeThreshold     time.Duration  // 600s / 10m
    ThinCadence          time.Duration  // 300s / 5m
}
```

- `Load()` reads from Viper (env vars via `SNAPPY_` prefix + config file)
- Add `viper.SetEnvPrefix("SNAPPY")` and defaults in `cmd/root.go`'s `initConfig()`

### `internal/snapshot/snapshot.go`

```go
type Snapshot struct {
    Date         string    // "YYYY-MM-DD-HHMMSS"
    Time         time.Time // parsed from Date
    UUID         string
    Purgeable    bool
    LimitsShrink bool
}
```

- `ParseDate(s string) (time.Time, error)` - parse `YYYY-MM-DD-HHMMSS`
- `FormatRelativeTime(t time.Time) string` - produce "3m ago", "2h ago", etc.

### `internal/snapshot/diff.go`

- `ComputeDiff(prev, curr []Snapshot) (added, removed []Snapshot)` - map-based set comparison

### Tests

Unit tests for date parsing, relative time formatting, snapshot diffing.

## Phase 2: Platform layer

Wrap macOS commands behind a testable interface. Add `howett.net/plist` dependency.

### `internal/platform/exec.go`

```go
type CommandRunner interface {
    Run(ctx context.Context, name string, args ...string) ([]byte, error)
}
```

Production implementation uses `os/exec.CommandContext`. Tests inject mocks with
canned output, enabling CI to run on Linux.

### `internal/platform/tmutil.go`

| Function                            | Wraps                           | Returns             |
| ----------------------------------- | ------------------------------- | ------------------- |
| `CheckStatus(ctx, runner)`          | `tmutil destinationinfo`        | status string       |
| `ListSnapshots(ctx, runner, mount)` | `tmutil listlocalsnapshotdates` | sorted date strings |
| `CreateSnapshot(ctx, runner)`       | `tmutil localsnapshot`          | created date string |
| `DeleteSnapshot(ctx, runner, date)` | `tmutil deletelocalsnapshots`   | error               |

### `internal/platform/diskutil.go`

| Function                                  | Wraps                                                         | Returns                        |
| ----------------------------------------- | ------------------------------------------------------------- | ------------------------------ |
| `FindAPFSVolume(ctx, runner, mount)`      | `diskutil info -plist` + `diskutil apfs listSnapshots -plist` | device string                  |
| `GetSnapshotDetails(ctx, runner, volume)` | `diskutil apfs listSnapshots -plist`                          | snapshot details + other count |

Parse plist XML with `howett.net/plist` instead of shelling out to `plutil` in a
loop. Handle both boolean and string ("YES"/"NO") plist field types for
`Purgeable` and `LimitingContainerShrink`.

### `internal/platform/disk.go`

- `GetDiskInfo(ctx, runner, mount)` - runs `df -h`, returns formatted string

### Tests

Mock-based tests for each function using canned command output.

## Phase 3: Logger

### `internal/logger/logger.go`

```go
type Logger struct {
    entries []Entry
    maxSize int       // 50
    file    *os.File  // nil if file logging disabled
    mu      sync.Mutex
}
```

- `New(logDir string, maxEntries int) (*Logger, error)` - creates dir, opens file, degrades gracefully
- `Log(eventType, message)` - appends to ring buffer + writes to file (thread-safe)
- `Entries() []Entry` - returns copy for the view to read
- `Close()` - closes file handle

Event types: `STARTUP`, `INFO`, `CREATED`, `ADDED`, `REMOVED`, `AUTO`, `ERROR`, `THINNED`

Format: `[HH:MM:SS] TYPE     message` (matching bash output)

## Phase 4: Auto-snapshot and thinning logic

### `internal/snapshot/auto.go`

```go
type AutoManager struct {
    enabled      bool
    interval     time.Duration
    thinAge      time.Duration
    thinCadence  time.Duration
    lastAutoTime time.Time
}
```

- `ShouldSnapshot(now) bool` - check if interval elapsed
- `RecordSnapshot(now)` - advance timer (regardless of success, matching bash)
- `Toggle() bool` - flip enabled, reset timer on enable
- `NextIn(now) time.Duration` - countdown for display
- `ComputeThinTargets(snapshots, now) []string` - walk oldest-first, apply age threshold + cadence, return dates to delete

### Tests

Cover: interval timing, toggle reset, thinning with various snapshot distributions,
edge cases (zero snapshots, all young, all old).

## Phase 5: Bubbletea TUI

This is the integration phase that wires everything together.

### `internal/tui/styles.go`

Lipgloss styles matching the bash color scheme (ANSI 8-color: green, yellow, red,
cyan, magenta, bold, dim). Separator strings (80 `=` or `-` chars).

### `internal/tui/messages.go`

Custom message types:

- `RefreshTickMsg` - periodic refresh timer
- `RefreshResultMsg` - carries snapshot list, TM status, APFS info, disk info
- `SnapshotCreatedMsg` - result of manual/auto snapshot creation
- `ThinResultMsg` - result of thinning operation

### `internal/tui/commands.go`

`tea.Cmd` factories wrapping async platform calls:

- `doRefresh(runner, cfg)` - list snapshots, get APFS details, get disk info
- `doCreateSnapshot(runner)` - create a snapshot
- `doThinSnapshots(runner, targets)` - delete thinning targets
- `refreshTick(interval)` - schedule next refresh

All macOS commands execute inside `tea.Cmd` (non-blocking to the UI).

### `internal/tui/model.go`

```go
type Model struct {
    cfg            *config.Config
    runner         platform.CommandRunner
    log            *logger.Logger
    auto           *snapshot.AutoManager

    snapshots      []snapshot.Snapshot
    prevSnapshots  []snapshot.Snapshot
    diffAdded      int
    diffRemoved    int

    tmStatus       string
    apfsVolume     string
    otherSnapCount int
    diskInfo       string
    lastRefresh    time.Time

    width, height  int
    quitting       bool
}
```

`Init()` returns a batch of `doRefresh` + `refreshTick`.

### `internal/tui/update.go`

Message dispatch:

| Message                   | Action                                                                 |
| ------------------------- | ---------------------------------------------------------------------- |
| `tea.KeyMsg` `s`          | Log "Creating snapshot...", return `doCreateSnapshot`                  |
| `tea.KeyMsg` `r`          | Return `doRefresh`                                                     |
| `tea.KeyMsg` `a`          | Toggle auto via `AutoManager.Toggle()`, log event                      |
| `tea.KeyMsg` `q`/`ctrl+c` | Set quitting, return `tea.Quit`                                        |
| `tea.WindowSizeMsg`       | Update width/height                                                    |
| `RefreshTickMsg`          | Check `auto.ShouldSnapshot()`, maybe create, then refresh, re-arm tick |
| `RefreshResultMsg`        | Update state, compute diff, log additions/removals                     |
| `SnapshotCreatedMsg`      | Log event, trigger refresh                                             |
| `ThinResultMsg`           | Log event, trigger refresh                                             |

### `internal/tui/view.go`

Four sections matching the bash layout:

1. **Header**: title bar, volume/refresh/last-refresh, TM status, APFS volume, disk info, auto-snapshot status with countdown
2. **Snapshot list**: bookend display (2 newest + ellipsis + 2 oldest when >4), numbered newest=1, with UUID/purgeable/pinned/limits-shrink flags
3. **Recent log**: last 8 entries newest-first, color-coded by event type
4. **Controls**: `[s] Snapshot   [r] Refresh   [a] Auto-snap   [q] Quit`

### `cmd/root.go` modifications

Set root command's `RunE` to:

1. Load config via `config.Load()`
2. Check `tmutil` exists in PATH
3. Create logger
4. Discover APFS volume + check TM status (one-time startup calls)
5. Create bubbletea model
6. `tea.NewProgram(model, tea.WithAltScreen()).Run()`
7. Close logger on exit

### Key bubbletea patterns

- `tea.WithAltScreen()` for full-screen mode (replaces bash cursor hide/show + cleanup trap)
- All platform calls in `tea.Cmd` functions (never block in `Update`)
- `refreshTick` re-arms itself each time it fires (periodic loop)
- `tea.WindowSizeMsg` replaces SIGWINCH trap

## Phase 6: Polish and tests

1. **Platform mock tests**: full refresh flow with realistic canned tmutil/diskutil output
2. **TUI model tests**: drive model with `tea.Msg` values, verify view output
3. **Edge cases**: zero snapshots, exactly 4, 5+ (bookend), APFS not found, tmutil missing, log dir not writable, terminal resize
4. **Build tags**: gate macOS-only integration tests with `//go:build darwin`
5. **Documentation**: update AGENTS.md structure section and README.md usage section

## Verification

After each phase, confirm:

- `make build` succeeds
- `make test` passes
- `make vet` and `make fmt` are clean

After Phase 5 (full TUI):

- `./bin/snappy` launches the TUI on macOS
- Visual comparison with `docs/proof-of-concept/snappy` (run side by side)
- Press each key (s, r, a, q) and verify behavior matches
- Let it run for a few refresh cycles, confirm auto-snapshots fire and thinning works
- Confirm log file appears at `~/.local/share/snappy/snappy.log`
- Resize terminal, confirm UI redraws correctly
