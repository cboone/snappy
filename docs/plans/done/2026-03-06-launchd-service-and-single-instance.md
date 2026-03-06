# launchd Background Service and Single-Instance Enforcement

Addresses: [#8](https://github.com/cboone/snappy/issues/8) (set up launchd), [#24](https://github.com/cboone/snappy/issues/24) (single autosnapshot worker)

## Context

Snappy's auto-snapshot loop currently only runs when the TUI is open. If the user
closes the terminal, snapshots stop. This defeats the purpose of "automatically
increase your Time Machine snapshot frequency." The `snappy run` foreground daemon
exists but has no way to start at login, no integration with the shared logger,
and no protection against multiple instances running simultaneously.

This plan adds:

- A launchd LaunchAgent so `snappy run` starts at login and persists in the background
- File-based locking so only one auto-snapshot routine runs at a time
- `snappy service` commands for full lifecycle management
- TUI awareness of the running daemon
- Shared logger integration for `snappy run`
- Homebrew caveats and install.sh instructions

## Design Decisions

**Lock mechanism**: `syscall.Flock` with `LOCK_EX | LOCK_NB` on
`~/.local/share/snappy/snappy.lock`. Automatically released by the OS on process
exit/crash. No stale lock problem, unlike PID files.

**LaunchAgent** (not LaunchDaemon): User-level agent in `~/Library/LaunchAgents/`.
Runs as the current user, has access to the home directory. Label: `com.cboone.snappy`.

**Service commands**: Full set: install, uninstall, start, stop, status, log.

**Homebrew**: Standard caveats printed after install, directing users to
`snappy service install`. No auto-start. The GoReleaser `homebrew_casks` config
gets `caveats` and `uninstall.launchctl` fields added.

**TUI coexistence**: When the daemon holds the lock, the TUI disables auto-snapshots
and shows "daemon active" in the info panel. Manual snapshot creation ('s') still
works. The auto-snap toggle ('a') logs a message instead of toggling.

## Implementation

### Step 1: Lock file package

**New file: `internal/service/lock.go`**

```go
package service

type LockFile struct {
    path string
    file *os.File
}

func DefaultLockPath(logDir string) string  // returns logDir + "/snappy.lock"
func Acquire(path string) (*LockFile, error) // LOCK_EX | LOCK_NB, returns ErrLocked if held
func (l *LockFile) Release() error
func IsHeld(path string) bool               // probe without acquiring

var ErrLocked = errors.New("another snappy auto-snapshot process is already running")
```

`Acquire` creates the directory if needed (`os.MkdirAll`), opens the file with
`O_CREATE|O_RDWR`, and calls `syscall.Flock`. On `EWOULDBLOCK`, returns `ErrLocked`.

`IsHeld` opens the file, tries `LOCK_EX|LOCK_NB`, and if it succeeds, immediately
releases and returns false. If it gets `EWOULDBLOCK`, returns true.

**New file: `internal/service/lock_test.go`**

- Acquire succeeds on first call
- Second acquire on same path returns `ErrLocked`
- `IsHeld` returns true when locked, false when released
- Lock auto-releases when file is closed
- Works when lock directory doesn't exist yet

### Step 2: launchd management package

**New file: `internal/service/launchd.go`**

```go
package service

const DefaultLabel = "com.cboone.snappy"

type PlistConfig struct {
    Label      string
    BinaryPath string // resolved absolute path to snappy binary
    LogDir     string // for StandardOutPath/StandardErrorPath
    ConfigFile string // optional --config flag value
}

func PlistPath(label string) (string, error)      // ~/Library/LaunchAgents/<label>.plist
func GeneratePlist(cfg PlistConfig) ([]byte, error) // render XML from template
func Install(cfg PlistConfig) error                // write plist + launchctl bootstrap
func Uninstall(label string) error                 // launchctl bootout + remove file
func Start(label string) error                     // launchctl kickstart
func Stop(label string) error                      // launchctl kill SIGTERM
func ResolveBinaryPath() (string, error)           // os.Executable + EvalSymlinks

type ServiceStatus struct {
    Installed  bool
    Running    bool
    PID        int
    BinaryPath string
    PlistPath  string
    Label      string
}
func Status(label string) (*ServiceStatus, error)  // parse launchctl print output
```

Uses modern `launchctl` API: `bootstrap gui/<uid>`, `bootout gui/<uid>/<label>`,
`kickstart gui/<uid>/<label>`, `kill SIGTERM gui/<uid>/<label>`.

Plist template (key properties):

- `RunAtLoad: true`
- `KeepAlive: true` (restart on crash)
- `ProcessType: Background` (power-friendly scheduling)
- `ThrottleInterval: 10` (prevent rapid restart loops)
- `StandardOutPath` / `StandardErrorPath` -> `<LogDir>/snappy-service.log`
- `ProgramArguments: [<binary>, run]` (plus `--config <path>` if set)

**New file: `internal/service/launchd_test.go`**

- `GeneratePlist` produces valid XML with correct fields
- `PlistPath` resolves to `~/Library/LaunchAgents/`
- `ResolveBinaryPath` resolves symlinks
- Template handles optional `--config` flag

### Step 3: Integrate lock into `snappy run`

**Modified: `cmd/run.go`**

At the start of `runDaemon`:

1. Load config
2. Ensure log directory exists
3. Acquire lock; exit with clear message if `ErrLocked`
4. Create a `logger.Logger` (file logging, no ring buffer needed)
5. Replace `logLine(w, ...)` calls with a helper that writes to both stdout and
   the shared logger, so daemon entries appear in `snappy.log`
6. Defer `lock.Release()` and `log.Close()`

The logger already handles directory creation and file rotation. The daemon's
entries will be interleaved with TUI entries in the same log file (safe via
`O_APPEND` atomicity for small writes).

**Reuse**: `internal/logger` (existing), `internal/service.Acquire` (step 1),
`cmd/helpers.go:requireTmutil`, `cmd/helpers.go:newRunner`, `cmd/helpers.go:loadSnapshots`,
`cmd/helpers.go:deleteSnapshots`.

### Step 4: Service management commands

**New file: `cmd/service.go`**

```text
snappy service             -> shows status (same as snappy service status)
snappy service install     -> generate plist, load agent, print confirmation
snappy service uninstall   -> unload agent, remove plist, print confirmation
snappy service start       -> launchctl kickstart
snappy service stop        -> launchctl kill SIGTERM
snappy service status      -> show installed/running/PID/binary path/plist path
snappy service log         -> tail -f the service log file (exec into tail)
```

`service install` flow:

1. Resolve binary path via `service.ResolveBinaryPath()`
2. Load config to get LogDir (and optional config file path)
3. Check if already installed; if so, uninstall first (clean re-install)
4. Generate plist, write to `~/Library/LaunchAgents/`
5. `launchctl bootstrap gui/<uid> <plist>`
6. Print: "Installed and started. snappy will run in the background."

`service uninstall` flow:

1. `launchctl bootout gui/<uid>/com.cboone.snappy`
2. Remove plist file
3. Print confirmation

`service status` output example:

```text
Service:   com.cboone.snappy
Status:    running (PID 12345)
Binary:    /opt/homebrew/bin/snappy
Plist:     ~/Library/LaunchAgents/com.cboone.snappy.plist
Log:       ~/.local/share/snappy/snappy-service.log
```

**New file: `cmd/service_test.go`**

Tests using the existing `mockRunner` pattern from `cmd/mock_test.go`.

### Step 5: TUI daemon awareness

**Modified: `cmd/root.go`**

After loading config, before creating the TUI model:

```go
lockPath := service.DefaultLockPath(cfg.LogDir)
daemonActive := service.IsHeld(lockPath)
if daemonActive {
    log.Log(logger.Startup, "Background service detected; TUI auto-snapshots disabled")
}
```

Pass `daemonActive` to `tui.NewModel`.

**Modified: `internal/tui/model.go`**

- Add `daemonActive bool` field to `Model`
- Update `NewModel` signature: add `daemonActive bool` parameter
- When `daemonActive` is true, create `AutoManager` with `enabled: false`

**Modified: `internal/tui/view.go`**

In `formatAutoStatus()`, add a third branch:

```go
if m.daemonActive {
    return label("Auto-snapshot:") + " " + indicatorOn + " " +
        m.styles.statusOn.Render("daemon") + ...
}
```

Show the daemon's config (interval, thin settings) so the user sees what's configured.

**Modified: `internal/tui/update.go`**

In `handleKey`, when `key.Matches(msg, m.keys.AutoSnap)` and `m.daemonActive`:

```go
m.log.Log(logger.Info, "Auto-snapshots managed by background service (snappy service stop to take over)")
```

Return without toggling.

### Step 6: install.sh instructions

**Modified: `install.sh`**

After the binary is installed, print:

```bash
printf '\nTo run snappy automatically in the background:\n'
printf '  %s/%s service install\n\n' "${INSTALL_DIR}" "${BINARY}"
```

Non-interactive (safe for `curl | bash`).

### Step 7: GoReleaser and Homebrew

**Modified: `.goreleaser.yml`**

Add `caveats` to the `homebrew_casks` section:

```yaml
homebrew_casks:
  - # ... existing config ...
    caveats: |
      To run snappy automatically in the background:
        snappy service install

      To stop the background service:
        snappy service uninstall
    uninstall:
      launchctl:
        - "com.cboone.snappy"
```

Note: verify that `homebrew_casks` supports `caveats` and `uninstall.launchctl`
in the version of GoReleaser being used. If not, this can be added to the
formula manually in the homebrew-tap repo as a follow-up.

### Step 8: Tests

**New scrut tests: `tests/scrut/service.md`**

- `snappy service --help` shows subcommands
- `snappy service status` runs without error (shows "not installed" if no plist)
- `snappy service install --help` shows usage

Updated existing scrut tests:

- Help output will include the new `service` command in the command list

### Step 9: Documentation

**Modified: `README.md`**

- Update Quick Start to mention `snappy service install`
- Add "Background Service" section documenting all `snappy service` subcommands
- Update install.sh section to mention service setup
- Update the structure diagram in CLAUDE.md

**Modified: `CLAUDE.md`**

Add `internal/service/` to the structure listing.

## Files Summary

| File                               | Action | Purpose                                |
| ---------------------------------- | ------ | -------------------------------------- |
| `internal/service/lock.go`         | New    | flock-based single-instance locking    |
| `internal/service/lock_test.go`    | New    | Lock mechanism tests                   |
| `internal/service/launchd.go`      | New    | Plist generation, launchctl management |
| `internal/service/launchd_test.go` | New    | Plist and service management tests     |
| `cmd/service.go`                   | New    | `snappy service` command group         |
| `cmd/service_test.go`              | New    | Service command tests                  |
| `cmd/run.go`                       | Modify | Add lock acquisition, shared logger    |
| `cmd/root.go`                      | Modify | Detect daemon, pass to TUI             |
| `internal/tui/model.go`            | Modify | Add `daemonActive` field               |
| `internal/tui/view.go`             | Modify | Show daemon status in info panel       |
| `internal/tui/update.go`           | Modify | Handle 'a' key when daemon active      |
| `install.sh`                       | Modify | Print service setup instructions       |
| `.goreleaser.yml`                  | Modify | Add caveats and uninstall.launchctl    |
| `README.md`                        | Modify | Document background service            |
| `CLAUDE.md`                        | Modify | Update structure listing               |
| `tests/scrut/service.md`           | New    | Scrut tests for service commands       |

## Edge Cases

- **Binary moved after install**: `snappy service status` detects mismatch and
  warns. Fix: `snappy service install` again.
- **Two `snappy run` instances**: Second one exits immediately with
  "another snappy auto-snapshot process is already running."
- **TUI + daemon**: TUI detects lock, disables auto-snapshots, shows daemon indicator.
  Manual snapshot ('s') still works.
- **Stale lock after crash**: `flock` is kernel-managed; automatically released
  on process exit. No cleanup needed.
- **Log directory missing**: Both logger and lock `Acquire` call `os.MkdirAll`.
- **Config changes while daemon runs**: Daemon must be restarted to pick up changes
  (`snappy service stop && snappy service start`). Note this in docs.
- **Homebrew upgrade**: Binary is replaced in-place; next daemon restart uses new
  version. `KeepAlive` in the plist handles the restart.

## Verification

1. `make build` compiles
2. `make test` passes (unit tests including new lock/service tests)
3. `make test-scrut` passes (updated help output, new service tests)
4. `make lint` clean
5. Manual testing:
   - `snappy service install` creates plist, starts daemon
   - `snappy service status` shows running + PID
   - `snappy run` (manual) fails with "already running"
   - `snappy` (TUI) shows "daemon" indicator, 'a' key shows message
   - `snappy service stop` stops daemon
   - `snappy run` (manual) succeeds, acquires lock
   - `snappy` (TUI) while manual `run` is active shows daemon indicator
   - `snappy service uninstall` removes plist
   - Reboot: agent starts automatically (RunAtLoad)
   - Agent crash: launchd restarts (KeepAlive + ThrottleInterval)
