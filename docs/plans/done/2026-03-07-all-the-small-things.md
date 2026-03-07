# All the Small Things: Issues #59, #62-#82

## Context

This plan addresses 14 open GitHub issues spanning bug fixes, refactors,
optimizations, test coverage, and enhancements. The branch name reflects the
nature of the work: many small, independent improvements. Each issue is
addressed, tested, and committed separately.

Two issues require no code changes:

- **#62** (snapRowAtVisualLine refactor): already resolved by PR #81 which
  removed the function entirely. Close with a comment.
- **#64** (revisit snapshot sizes with TM enabled): investigation-only task
  requiring a Time Machine backup destination. Leave open for future research.

---

## Issue Order

Issues are ordered to minimize merge conflicts and respect dependencies.
Foundational refactors come first, then bug fixes, then features, then tests.

---

## 1. Close #62 - snapRowAtVisualLine already removed

`snapRowAtVisualLine` was removed in PR #81 (decouple mouse wheel scroll from
selection). The test `TestMouseClickSnapshotSelectsTopVisibleRowWhenTableIsOffset`
was renamed to `TestMouseClickSnapshotSelectsTopVisibleRowWhenScrolled`. No code
changes needed.

**Action:** `gh issue close 62 --comment "..."`

---

## 2. Skip #64 - snapshot size investigation

Leave open for future research. Requires Time Machine enabled on a backup
destination, which is outside the scope of this branch.

---

## 3. #66 - Replace NewModel parameter list with options struct

**Problem:** `NewModel` takes 9 positional parameters, 5 of which are bare
strings. Easy to swap arguments without a compile error.

**Files:**
- `internal/tui/model.go` - add `ModelParams` struct, update `NewModel` signature
- `cmd/root.go` - update call site (line ~196)
- `internal/tui/model_test.go` - update `testModel()` helper (line ~50)

**Implementation:**

```go
type ModelParams struct {
    APFSVolume    string
    APFSContainer string
    TMStatus      string
    VolumeName    string
    Version       string
    DaemonActive  bool
}

func NewModel(cfg *config.Config, runner platform.CommandRunner, log *logger.Logger, params ModelParams) Model
```

Move all six fields (5 strings + bool) into the struct. Keep `cfg`, `runner`,
and `log` as direct parameters since they are distinct types that can't be
accidentally swapped.

**Tests:** Existing tests continue to pass after updating `testModel()`.

---

## 4. #75 - Fix logDiffChanges FOUND count including user-created snapshot

**Problem:** On first refresh, all `diff.Added` snapshots are counted as
"Found N existing" without filtering out snapshots the user just created via
the 'c' key.

**File:** `internal/tui/update.go`, `logDiffChanges()` (line ~530)

**Fix:** Before logging the FOUND count, subtract entries present in
`m.recentCreated`:

```go
if !m.hadFirstRefresh && len(diff.Added) > 0 {
    foundCount := 0
    for _, s := range diff.Added {
        if _, ok := m.recentCreated[s.Date]; !ok {
            foundCount++
        }
    }
    if foundCount > 0 {
        m.log.Log(logger.LevelInfo, logger.CatFound, fmt.Sprintf(
            "Found %d existing snapshots", foundCount))
    }
}
```

**Tests:** Add a unit test in `model_test.go` that:
1. Creates a model, adds a date to `recentCreated`
2. Sends a `RefreshResultMsg` containing that date plus others
3. Asserts the log says "Found N existing" with the reduced count

---

## 5. #65 - Fix log cursor drift when ring buffer is at capacity

**Problem:** When the ring buffer is full (50 entries), new entries replace old
ones without changing `len(entries)`. The cursor adjustment in
`updateLogViewContent()` only fires when `newCount > m.logCount`, so the
cursor silently drifts.

**Files:**
- `internal/logger/logger.go` - add `Seq uint64` to `Entry`, add `seq uint64`
  counter to `Logger`, increment in `Log()`
- `internal/tui/model.go` - add `logLastSeq uint64` field to `Model`
- `internal/tui/update.go` - in `updateLogViewContent()`, detect new entries
  by comparing the newest entry's `Seq` with `m.logLastSeq`

**Implementation in `updateLogViewContent()`:**

```go
entries := m.log.Entries()
newCount := len(entries)

// Detect genuinely new entries using sequence numbers.
var newestSeq uint64
if newCount > 0 {
    newestSeq = entries[newCount-1].Seq
}

if m.logCursor > 0 {
    if newCount > m.logCount {
        // Buffer grew: shift cursor by the number of new entries.
        m.logCursor += newCount - m.logCount
    } else if newCount == m.logCount && newestSeq > m.logLastSeq {
        // Buffer at capacity: each new entry shifts existing ones down by 1.
        m.logCursor += int(newestSeq - m.logLastSeq)
    }
}
m.logLastSeq = newestSeq
m.logCount = newCount
if m.logCursor >= m.logCount {
    m.logCursor = max(m.logCount-1, 0)
}
```

**Tests:** Add a test in `model_test.go` that:
1. Fills the ring buffer to capacity (50 entries)
2. Moves the cursor to a non-zero position
3. Adds more entries (triggering the at-capacity path)
4. Asserts the cursor still points at the same logical entry

---

## 6. #82 - Preserve log viewport position when new entries arrive

**Problem:** When the user scrolls the log viewport away from the top, new
entries prepend content and the viewport jumps.

**File:** `internal/tui/update.go`, `updateLogViewContent()` (line ~763)

**Fix:** After rebuilding content, if the viewport was scrolled (YOffset > 0),
adjust the offset by the number of newly added visual lines:

```go
// Before SetContent:
prevOffset := m.logView.YOffset
prevTotalLines := m.logTotalLines

// ... rebuild content, set m.logTotalLines ...

m.logView.SetContent(content)

// After SetContent: adjust viewport if scrolled
if prevOffset > 0 && m.logTotalLines > prevTotalLines {
    newOffset := prevOffset + (m.logTotalLines - prevTotalLines)
    maxOffset := m.logTotalLines - m.logView.Height
    if newOffset > maxOffset {
        newOffset = maxOffset
    }
    if newOffset > 0 {
        m.logView.SetYOffset(newOffset)
    }
}
```

**Tests:** Add a test that verifies viewport offset stability when new log
entries arrive while scrolled.

---

## 7. #59 - Optimize UITickMsg to only update AGE column

**Problem:** `handleUITick()` calls `updateSnapViewContent()` which rebuilds
columns and all rows. Only the AGE column values actually change between ticks.

**File:** `internal/tui/update.go`

**Implementation:** Extract `updateSnapAges()` that reuses existing columns
and only rebuilds row data:

```go
func (m *Model) updateSnapAges() {
    rows := m.snapTable.Rows()
    if len(rows) == 0 {
        return
    }
    now := m.now()
    count := len(m.snapshots)
    for ri, row := range rows {
        si := count - 1 - ri // newest-first mapping
        if si >= 0 && si < count {
            row[1] = snapshot.FormatRelativeTime(m.snapshots[si].Time, now)
            rows[ri] = row
        }
    }
    m.snapTable.SetRows(rows)
    m.updateSnapRenderCache()
}
```

Update `handleUITick()` to call `updateSnapAges()` instead of
`updateSnapViewContent()`. The full rebuild remains in `handleRefreshResult()`
and `handleWindowSize()`.

**Tests:** Add a test that verifies only the AGE column changes after a tick,
and that column widths remain stable.

---

## 8. #72 - Log tidemark fetch failures in TUI refresh

**Problem:** `doRefresh()` silently drops `GetContainerTidemark` errors when
`apfsContainer` is set.

**Files:**
- `internal/tui/messages.go` - add `TidemarkErr error` to `RefreshResultMsg`
- `internal/tui/commands.go` - populate `TidemarkErr` in `doRefresh()`
- `internal/tui/update.go` - in `handleRefreshResult()`, log when
  `msg.TidemarkErr != nil`

**Implementation in `doRefresh()`:**

```go
var tidemark int64
var tidemarkErr error
if apfsContainer != "" {
    tidemark, tidemarkErr = platform.GetContainerTidemark(ctx, runner, apfsContainer)
}
// Include tidemarkErr in the returned msg
```

In `handleRefreshResult()`, log the error:

```go
if msg.TidemarkErr != nil {
    m.log.Log(logger.LevelWarn, logger.CatRefresh,
        "Tidemark fetch failed: "+msg.TidemarkErr.Error())
}
```

**Tests:** Add a test with a mock runner that returns an error for the
tidemark command, verify the warning appears in the log.

---

## 9. #76 - Eliminate IsHeld/Acquire race with shared lock

**Problem:** `IsHeld` probes with `LOCK_EX`, briefly holding an exclusive lock.
A concurrent `Acquire` during that window gets a spurious `ErrLocked`.

**File:** `internal/service/lock.go`, `IsHeld()` (line 68)

**Fix:** Change `IsHeld` to use `LOCK_SH` (shared lock) instead of `LOCK_EX`:

```go
err = syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
```

Rationale: A shared lock won't conflict with other shared locks (multiple
`IsHeld` calls), and will still detect an exclusive lock held by `Acquire`
(returns `EWOULDBLOCK`). The residual race (IsHeld's shared lock blocking
Acquire's exclusive lock) has the same nanosecond window but is far less
likely since IsHeld is called infrequently (every few seconds by the TUI).

**Tests:** Existing `lock_test.go` tests should pass. Add a test that
verifies `IsHeld` returns true while a lock is held and false when released.

---

## 10. #73 - Acquire lock around TUI auto-snapshots

**Problem:** `syncDaemonState()` checks `IsHeld` then proceeds with
auto-snapshots, but the lock state could change between the check and the
actual snapshot execution.

**Files:**
- `internal/tui/commands.go` - in `doAutoSnapshot()` (or wherever
  auto-snapshots are triggered), attempt to acquire the lock before creating
  the snapshot
- `internal/tui/update.go` - handle `ErrLocked` in the auto-snapshot result

**Implementation:** In the auto-snapshot command function, wrap the snapshot
creation with a lock acquire/release:

```go
lock, err := service.Acquire(lockPath)
if err != nil {
    if errors.Is(err, service.ErrLocked) {
        return AutoSnapshotResultMsg{Skipped: true}
    }
    return AutoSnapshotResultMsg{Err: err}
}
defer lock.Release()
// ... create snapshot ...
```

This requires passing `lockPath` into the auto-snapshot command. The model
already has `m.cfg.LogDir` from which `DefaultLockPath` can be derived.

**Tests:** Add a test that verifies the auto-snapshot is skipped when the lock
is held.

---

## 11. #74 - Use plist serializer for GeneratePlist

**Problem:** `GeneratePlist()` uses `text/template` which doesn't XML-escape
values. Fields with `&` could produce invalid plist XML.

**File:** `internal/service/launchd.go`

**Implementation:** Replace the template with `howett.net/plist.Marshal()`:

```go
type generatedPlist struct {
    Label            string   `plist:"Label"`
    ProgramArguments []string `plist:"ProgramArguments"`
    RunAtLoad        bool     `plist:"RunAtLoad"`
    KeepAlive        bool     `plist:"KeepAlive"`
    ProcessType      string   `plist:"ProcessType"`
    ThrottleInterval int      `plist:"ThrottleInterval"`
    StandardOutPath  string   `plist:"StandardOutPath"`
    StandardErrorPath string  `plist:"StandardErrorPath"`
}

func GeneratePlist(cfg PlistConfig) ([]byte, error) {
    logPath := filepath.Join(cfg.LogDir, "snappy-service.log")
    args := []string{cfg.BinaryPath, "run"}
    if cfg.ConfigFile != "" {
        args = append(args, "--config", cfg.ConfigFile)
    }

    pl := generatedPlist{
        Label:             cfg.Label,
        ProgramArguments:  args,
        RunAtLoad:         true,
        KeepAlive:         true,
        ProcessType:       "Background",
        ThrottleInterval:  10,
        StandardOutPath:   logPath,
        StandardErrorPath: logPath,
    }
    return plist.MarshalIndent(pl, plist.XMLFormat, "\t")
}
```

Remove the `plistTemplate` variable and `plistData` struct. The output format
will be proper XML with correct escaping.

**Tests:** Update `launchd_test.go` to verify the output is valid plist XML
and that values containing `&` are properly escaped.

---

## 12. #77 - Derive service log path from installed plist

**Problem:** `runServiceStatus` and `runServiceLog` derive the log path from
`config.Load()`, but the installed plist may have been installed with a
different `LogDir`.

**Files:**
- `internal/service/launchd.go` - add `StandardOutPath string` to
  `launchdPlist` struct; add `ReadLogPathFromPlist(path string) string` function
- `cmd/service.go` - in `runServiceStatus()` and `runServiceLog()`, try to
  read the log path from the installed plist first, fall back to config

**Implementation in launchd.go:**

```go
type launchdPlist struct {
    ProgramArguments []string `plist:"ProgramArguments"`
    StandardOutPath  string   `plist:"StandardOutPath"`
}

func ReadLogPathFromPlist(path string) string {
    data, err := os.ReadFile(path)
    if err != nil {
        return ""
    }
    var pl launchdPlist
    if _, err := plist.Unmarshal(data, &pl); err != nil {
        return ""
    }
    return pl.StandardOutPath
}
```

In `runServiceStatus()` (line ~240-241):

```go
logPath := service.ReadLogPathFromPlist(st.PlistPath)
if logPath == "" {
    cfg := config.Load()
    logPath = service.LogPath(cfg.LogDir)
}
_, _ = fmt.Fprintf(w, "Log:     %s\n", logPath)
```

Similar change in `runServiceLog()`.

**Tests:** Add tests in `service_test.go` and `launchd_test.go` that verify
the plist-derived path is used when available and the config fallback when not.

---

## 13. #69 - Test coverage: tidemark display and XID delta formatting

**File:** `internal/tui/model_test.go`

**Tests to add:**
1. `TestRefreshResultTidemarkFormatted` - send `RefreshResultMsg` with
   `Tidemark: 50000000000`, assert `m.tidemark` contains formatted bytes string
2. `TestRefreshResultTidemarkEmpty` - send with `Tidemark: 0`, assert
   `m.tidemark` is empty
3. `TestInfoPanelIncludesTidemark` - assert rendered view contains "Tidemark:"
   when tidemark is set
4. `TestInfoPanelOmitsTidemark` - assert rendered view omits "Tidemark:" when
   empty
5. `TestXIDDeltaInSnapTable` - verify XID delta column is populated correctly
   for snapshots with UUID data

**File:** `cmd/list_test.go`

6. `TestFormatRelativeAgoFuture` - assert `formatRelativeAgo` returns "future"
   for a future timestamp

---

## 14. #71 - Test coverage: list command XID delta and APFS details

**File:** `cmd/list_test.go`

**Implementation:** Create test cases with mock runners that return valid
`diskutil info` and `diskutil apfs listSnapshots` plist responses.

**Tests to add:**
1. `TestListJSONWithAPFSDetails` - mock returns APFS snapshot details; assert
   JSON output includes `uuid`, `xid_delta`, `purgeable`, `limits_shrink`
2. `TestListJSONXIDDeltaOmittedForFirst` - assert first snapshot has no
   `xid_delta`
3. `TestListHumanWithAPFSDetails` - assert human output includes `delta:N`
   suffix, purgeable/pinned flags, and limits-shrink warning

This requires adding mock responses for both `tmutil listlocalsnapshotdates`
and `diskutil` commands. Reuse the existing `mockRunner` pattern from
`helpers_test.go`.

---

## Verification

After each commit:

```bash
make test          # Go unit tests pass
make lint          # No lint errors
make build         # Binary builds
```

After all commits:

```bash
make test-all      # Full test suite (unit + scrut + scrut-ez)
```
