# Address minor detail issues (#37, #42, #43, #44, #53)

## Context

Five open issues track small improvements identified during PR reviews and
general codebase housekeeping. None are large features; each is a targeted fix,
refactor, or optimization. Addressing them together in one branch keeps the
scope manageable and avoids repeated review cycles.

## Issue #44: Deduplicate plist parsing in diskutil.go

**File:** `internal/platform/diskutil.go`

`GetVolumeName` (line 116) and `getDeviceIdentifier` (line 130) both run
`diskutil info -plist`, unmarshal into `diskutilInfoPlist`, and return one field.
The logic is duplicated.

### Changes

1. Add an unexported helper `getDiskutilInfo` that runs the command, parses the
   plist, and returns the struct.
2. Rewrite `GetVolumeName` and `getDeviceIdentifier` as one-liner wrappers that
   call the helper and return the relevant field.

---

## Issue #37: Handle write errors in logLine helper

**Files:** `cmd/run.go`, `cmd/run_test.go`

`logLine` (line 107) discards write errors (`_, _ = fmt.Fprintf(...)`). If the
output pipe breaks, the daemon loop keeps running silently.

### Changes

1. Change `logLine` to return `error`.
2. Change `runIteration` to return `error`. Check each `logLine` call and return
   on write failure.
3. In `runDaemon`, check errors from `logLine` (STARTUP, SHUTDOWN) and from
   `runIteration`. On error, exit the loop and return the error through Cobra.
4. Update `TestLogLine` to verify the return value. Add a sub-test with a
   failing writer (`errWriter`) to confirm error propagation.
5. Update `TestRunIteration*` tests to capture and check the returned error.

---

## Issue #42: Document click-to-select design rationale

**File:** `internal/tui/update.go`

The fragile arithmetic was already replaced by `snapRowAtVisualLine` (line 621),
which parses the rendered table view. The issue also requested documenting the
coupling and assumptions.

### Changes

1. Expand the doc comment on `snapRowAtVisualLine` to explain:
   - Why the rendered view is parsed (Bubbles table does not expose its viewport
     scroll offset).
   - The coupling to the DATE column being first with a fixed 19-char width.

---

## Issue #43: Skip 1-second UI tick when auto-snapshot is disabled and idle

**Files:** `internal/tui/model.go`, `internal/tui/update.go`,
`internal/tui/model_test.go`

The UI tick fires every second unconditionally, rebuilding all table rows even
when nothing meaningful has changed.

### Changes

1. **`Init()` (model.go:186):** Only include `uiTick()` in the batch when
   `m.auto.Enabled()` is true.
2. **`UITickMsg` handler (update.go:53):** Only reschedule `uiTick()` when
   `m.auto.Enabled() || m.loading`.
3. **Auto-snap toggle handler (update.go:147):** When toggling auto ON, return
   `uiTick()` as a command to restart the tick cycle.
4. **Tests (model_test.go):** Add:
   - `TestUITickStopsWhenAutoDisabledAndIdle`: auto off, not loading, expect nil cmd.
   - `TestUITickContinuesWhenAutoEnabled`: auto on, expect non-nil cmd.
   - `TestAutoToggleOnRestartsUITick`: toggle on, expect non-nil cmd.

---

## Issue #53: Remove mount point config

Remove the `MountPoint` field from `Config`. The mount point is always "/" on
macOS for Time Machine snapshots. Continue to show the mount point (hardcoded
to "/") in `snappy status` and `snappy list` output (both human and JSON), but
omit it from `snappy config` show/init output.

### Config package (`internal/config/config.go`)

1. Remove `MountPoint string` from `Config` struct (line 21).
2. Export the constant: `DefaultMount = "/"` (line 32).
3. Remove from `Load()`: `MountPoint: viper.GetString("mount")` (line 51).
4. Remove from `SetDefaults()`: `viper.SetDefault("mount", ...)` (line 74).
5. Remove `mount:` line from `defaultConfigTmpl` (line 102) and its template
   data field.
6. Remove `mount:` line from `formatConfigTmpl` (line 163) and its data field.
7. Decrement the comment count in `defaultConfigTmpl` if the associated comment
   is removed (check the `grep -c '^#'` scrut test).

### Config tests (`internal/config/config_test.go`)

1. Remove `MountPoint` check from `TestLoadDefaults` (line 70).
2. Remove the "mount override" test case from `TestLoadEnvOverrides` (line 110).
3. Remove `MountPoint` field from `TestFormatConfig` data struct (line 306) and
   `"mount: /"` from expected strings (line 329).
4. Remove `MountPoint` check from `TestLoadWithoutSetDefaults` (line 388).
5. Remove `"mount:"` from the keys list in `TestWriteDefaultConfig` (line 293).

### Command files

Replace every `cfg.MountPoint` with `config.DefaultMount`:

- `cmd/root.go` (lines 112, 114, 118, 120, 124)
- `cmd/helpers.go` (lines 37, 50)
- `cmd/list.go` (lines 80, 91)
- `cmd/status.go` (lines 39, 40, 42, 102, 121)
- `internal/tui/commands.go` (lines 22, 24, 65): also add
  `"github.com/cboone/snappy/internal/config"` import.

### Command tests

- `cmd/config_test.go`: Remove `"mount:"` from `TestConfigShow` keys list
  (line 34).
- `cmd/list_test.go`: No change needed (JSON still has `mount: "/"`).

### TUI test

- `internal/tui/model_test.go`: Remove `MountPoint: "/"` from `testConfig()`
  (line 41).

### Scrut tests

- `tests/scrut/config-command.md`:
  - Remove `mount: /` line from "Config show" expected output (line 12).
  - Remove entire "Config show with env override" test (lines 22-37) since
    `SNAPPY_MOUNT` no longer exists.
  - Update comment count in "Config init file contains comments" if the mount
    comment was removed (line 50: `grep -c '^#'` expected count drops by 2).
- `tests/scrut/environment.md`:
  - Remove "SNAPPY_MOUNT env var reaches TUI stage" test (lines 18-24).
  - Remove "SNAPPY_MOUNT env var produces no stdout" test (lines 26-31).
  - Remove `SNAPPY_MOUNT` from "Multiple SNAPPY env vars together" (line 60).
  - Remove `SNAPPY_MOUNT` from "Env var with help flag" (line 68).
  - Remove `SNAPPY_MOUNT` from "Env var with version flag" (line 96).
- `tests/scrut/config-files.md`:
  - Remove `SNAPPY_MOUNT` from "Config flag combined with env var" (line 40).
  - Update the test description (line 36) since it no longer tests mount.
- `tests/scrut/status-command.md`:
  - Keep "Status shows mount point" test (line 29) unchanged since human output
    still shows `Mount: /`.

---

## Commit strategy

One commit per issue in this order:

1. `refactor(platform): extract getDiskutilInfo to deduplicate plist parsing`
2. `fix(cmd): propagate write errors from logLine in run daemon`
3. `docs(tui): document click-to-select design rationale`
4. `perf(tui): skip UI tick when auto-snapshot is disabled and idle`
5. `refactor(config): remove mount point configuration`

---

## Verification

After each commit:

```bash
make test           # Go unit tests
make test-scrut     # Scrut CLI tests
make lint           # golangci-lint, markdownlint, actionlint
```

After all commits:

```bash
make test-all       # Full test suite
make build && ./snappy --help   # Smoke test
```
