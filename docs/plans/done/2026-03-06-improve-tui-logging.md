# Improve TUI Logging

## Context

The TUI log panel has several issues: confusing messages, redundant entries, an opaque error, and a conflation of log severity with event category. The current `EventType` enum mixes log levels (INFO, ERROR) with event categories (CREATED, ADDED, AUTO, etc.) into a single dimension. This plan separates them into two dimensions and addresses all other logging improvements.

## Part 0: Separate log Level from event Category

This is the foundational change. Everything else builds on it.

### New types

Replace the single `EventType` with two orthogonal enums:

```go
// Level is the severity of a log entry.
type Level string

const (
    LevelInfo  Level = "INFO"
    LevelWarn  Level = "WARN"
    LevelError Level = "ERROR"
)

// Category identifies what kind of event occurred.
type Category string

const (
    CatStartup  Category = "STARTUP"
    CatRefresh  Category = "REFRESH"
    CatSnapshot Category = "SNAPSHOT"
    CatCreated  Category = "CREATED"
    CatAdded    Category = "ADDED"
    CatRemoved  Category = "REMOVED"
    CatAuto     Category = "AUTO"
    CatThinned  Category = "THINNED"
    CatFound    Category = "FOUND"     // new: first-run discovery
    CatShutdown Category = "SHUTDOWN"  // new: quit
    CatOpen     Category = "OPEN"      // new: open log dir
)
```

### Entry struct

```go
Entry struct {
    Timestamp time.Time
    Level     Level
    Category  Category
    Message   string
    Formatted string  // "[HH:MM:SS] LEVEL CATEGORY message"
}
```

### Log method signature

Change from `Log(eventType EventType, message string)` to `Log(level Level, category Category, message string)`.

### Display format

```text
[14:32:01] INFO  STARTUP  snappy v0.6.0 | volume=/ | refresh=60s
```

Format string: `[%s] %-5s %-8s %s` (time, level padded to 5, category padded to 8, message). Total prefix width increases from 21 to 26 characters.

### Styling

Color is driven by **Level**, not Category:

- `LevelError` -> `textRed`
- `LevelWarn` -> `textYellow`
- `LevelInfo` -> default (no style)

Category can optionally add a secondary style hint (e.g., dim for FOUND, cyan for AUTO), but Level takes precedence for the overall color.

### Complete message classification

Every existing `log.Log()` call remapped to Level + Category:

| Location      | Old EventType | New Level | New Category | Message                                    |
| ------------- | ------------- | --------- | ------------ | ------------------------------------------ |
| root.go:114   | Startup       | **WARN**  | STARTUP      | Warning: failed to discover APFS volume... |
| root.go:123   | Startup       | INFO      | STARTUP      | snappy version/volume/refresh              |
| root.go:126   | Startup       | INFO      | STARTUP      | apfs-volume=...                            |
| root.go:128   | Startup       | INFO      | STARTUP      | auto-snapshot config                       |
| update.go:133 | Info          | INFO      | SNAPSHOT     | Creating snapshot...                       |
| update.go:152 | Info          | INFO      | AUTO         | Auto-snapshots enabled...                  |
| update.go:159 | Info          | INFO      | AUTO         | Auto-snapshots disabled                    |
| update.go:165 | Info          | INFO      | SHUTDOWN     | Shutting down                              |
| update.go:290 | Auto          | INFO      | AUTO         | Creating auto-snapshot...                  |
| update.go:317 | Info          | INFO      | REFRESH      | Non-TM snapshots on...: N                  |
| update.go:330 | Error         | ERROR     | REFRESH      | APFS details unavailable...                |
| update.go:334 | Error         | ERROR     | REFRESH      | Failed to list snapshots...                |
| update.go:351 | Added         | INFO      | ADDED        | Snapshot appeared: DATE                    |
| update.go:354 | Removed       | INFO      | REMOVED      | Snapshot disappeared: DATE                 |
| update.go:358 | Info          | INFO      | REFRESH      | Refresh: N snapshots, disk...              |
| update.go:409 | Error         | ERROR     | SNAPSHOT     | Failed to create snapshot...               |
| update.go:411 | Created       | INFO      | CREATED      | Snapshot created: DATE                     |
| update.go:413 | Created       | INFO      | CREATED      | Snapshot created                           |
| update.go:432 | Thinned       | INFO      | THINNED      | Thinned N snapshot(s)...                   |
| update.go:445 | Error         | **WARN**  | THINNED      | ESTALE: stale handle, skipped (see Part 4) |
| update.go:445 | Error         | ERROR     | THINNED      | Other thinning errors (non-ESTALE)         |
| (new)         | --            | INFO      | FOUND        | Found N existing snapshots                 |
| (new)         | --            | INFO      | AUTO         | Thinning N snapshot(s)...                  |
| (new)         | --            | INFO      | OPEN         | Opening log directory...                   |

### Files to modify

**`internal/logger/logger.go`**:

- Replace `EventType` with `Level` and `Category` types and constants
- Update `Entry` struct: replace `Type EventType` with `Level Level` and `Category Category`
- Update `Log()` signature to `Log(level Level, cat Category, message string)`
- Update format string to `[%s] %-5s %-8s %s`
- Remove old `EventType` constants

**`internal/tui/update.go`**:

- Update every `m.log.Log(...)` call to the new three-argument signature per the table above
- Update `logEntryStyle()` to style based on `entry.Level` instead of `entry.Type`:
  - `LevelError` -> `s.textRed`
  - `LevelWarn` -> `s.textYellow`
  - `LevelInfo` -> check Category for secondary hints (CatAuto -> `s.textCyan`, CatStartup -> `s.textMagenta`, CatFound -> dim/subtle, others -> default)
- Update `updateLogViewContent()` prefix width from 21 to 27

**`cmd/root.go`**:

- Update 4 startup log calls to new signature

**`internal/tui/model_test.go`**:

- Update all test assertions that check `entry.Type` to check `entry.Level` and `entry.Category`
- Update all `m.log.Log(...)` calls in test helpers

**`internal/logger/logger_test.go`**:

- Update all `Log()` calls and `Entry` field assertions

## Part 1: FOUND summary for first-run discovery

Replace N individual ADDED lines on first refresh with a single FOUND summary.

**`internal/tui/model.go`**:

- Add `hadFirstRefresh bool` field to `Model`

**`internal/tui/update.go`** (handleRefreshResult, ~line 347-356):

- When `!m.hadFirstRefresh` and `len(diff.Added) > 0`:
  - Log `logger.LevelInfo, logger.CatFound, fmt.Sprintf("Found %d existing snapshots", len(diff.Added))`
  - Skip the per-snapshot ADDED loop
- Set `m.hadFirstRefresh = true` after first refresh regardless
- When `m.hadFirstRefresh`, continue logging individual ADDED lines as before

## Part 2: Suppress redundant ADDED after snapshot creation

**`internal/tui/model.go`**:

- Add `recentCreated map[string]struct{}` field, initialized in `NewModel`

**`internal/tui/update.go`** (handleSnapshotCreated, ~line 410-411):

- When `msg.Date != ""`, add `m.recentCreated[msg.Date] = struct{}{}`

**`internal/tui/update.go`** (handleRefreshResult, ~line 350-352):

- Skip logging ADDED for dates in `m.recentCreated`
- Clear `m.recentCreated` after processing the diff

## Part 3: Suppress redundant REMOVED after thinning; add thinning-start log

**`internal/tui/model.go`**:

- Add `recentThinned map[string]struct{}` field, initialized in `NewModel`

**`internal/tui/messages.go`** (ThinResultMsg):

- Add `ThinnedDates []string` field

**`internal/tui/commands.go`** (doThinSnapshots):

- Track successfully deleted dates, include as `ThinnedDates` in the result

**`internal/tui/update.go`** (handleThinResult):

- Populate `m.recentThinned` from `msg.ThinnedDates`

**`internal/tui/update.go`** (handleRefreshResult):

- Skip logging REMOVED for dates in `m.recentThinned`
- Clear `m.recentThinned` after processing the diff

**`internal/tui/update.go`** (maybeThin):

- Before dispatching `doThinSnapshots`, log:
  `m.log.Log(logger.LevelInfo, logger.CatAuto, fmt.Sprintf("Thinning %d snapshot(s)...", len(filtered)))`

## Part 4: User-friendly ESTALE error messages

**`internal/tui/commands.go`** (doThinSnapshots, ~line 97-99):

- Import `errors` and `os/exec`
- Check if error wraps `*exec.ExitError` with exit code 70
- If ESTALE: use `fmt.Sprintf("%s (stale handle, skipped)", date)` in detail string
- Otherwise: use existing `fmt.Sprintf("%s (%v)", date, err)` format

**`internal/tui/commands.go`** (doThinSnapshots):

- Add `estaleCount int` to track how many failures were ESTALE vs other errors
- Return both counts in `ThinResultMsg` (add `EstaleCount int` field to `ThinResultMsg` in messages.go)

**`internal/tui/update.go`** (handleThinResult, ~line 440-445):

- If all failures are ESTALE (`msg.EstaleCount == len(msg.FailedDates)`): log as WARN, not ERROR
  `m.log.Log(logger.LevelWarn, logger.CatThinned, fmt.Sprintf(...))`
- If any non-ESTALE failures exist: log as ERROR
  `m.log.Log(logger.LevelError, logger.CatThinned, fmt.Sprintf(...))`
- Still add all failed dates to `m.thinPinned` regardless

## Part 5: Load previous log entries on startup

**`internal/logger/logger.go`**:

- Add `LoadTail()` method:
  - Return early if `l.file == nil` or `l.filePath == ""`
  - Read the log file, take the last `l.maxSize` lines
  - Parse each line using the new format `[HH:MM:SS] LEVEL CATEGORY message`
  - Reconstruct `Entry` structs and populate the ring buffer
  - Lines that don't parse are silently skipped
  - Use today's date for the date component (file only stores HH:MM:SS)
- Add helper `parseLogLine(line string) (Entry, bool)`
- Also handle the old single-field format `[HH:MM:SS] TYPE     message` gracefully during the transition (parse TYPE as Category, default Level to INFO)

**`cmd/root.go`**:

- After `logger.New(...)` and before the first startup log, call `log.LoadTail()`

## Part 6: Key binding to open log directory

**`internal/tui/model.go`**:

- Add `OpenLog key.Binding` to `keyMap` struct
- Add `key.NewBinding(key.WithKeys("l", "L"), key.WithHelp("l", "open log"))` to `defaultKeyMap()`
- Add to `ShortHelp()` and `FullHelp()`

**`internal/tui/update.go`** (handleKey):

- Add case for `m.keys.OpenLog`:
  `m.log.Log(logger.LevelInfo, logger.CatOpen, "Opening log directory...")`
  Return `doOpenLogDir(m.cfg.LogDir)`

**`internal/tui/commands.go`**:

- Add `doOpenLogDir(dir string) tea.Cmd` using `exec.Command("open", dir).Start()`
- Fire-and-forget: returns nil message

## Part 7: Only log refresh summary when data changes

**`internal/tui/model.go`**:

- Add `lastRefreshSummary string` field

**`internal/tui/update.go`** (handleRefreshResult):

- Build the summary string, compare with `m.lastRefreshSummary`
- Only log if changed; update `m.lastRefreshSummary`

## Files Modified

| File                             | Parts               |
| -------------------------------- | ------------------- |
| `internal/logger/logger.go`      | 0, 5                |
| `internal/logger/logger_test.go` | 0, 5                |
| `internal/tui/model.go`          | 0, 1, 2, 3, 6, 7    |
| `internal/tui/update.go`         | 0, 1, 2, 3, 4, 6, 7 |
| `internal/tui/commands.go`       | 3, 4, 6             |
| `internal/tui/messages.go`       | 3, 4                |
| `cmd/root.go`                    | 0, 5                |
| `internal/tui/model_test.go`     | 0, 1, 2, 3, 4, 7    |

## Commit Strategy

One commit per part, in order. Each commit should pass `make test` and `make lint`.

0. `refactor(logger): separate log Level from event Category`
1. `feat(tui): summarize first-run snapshot discovery as single FOUND entry`
2. `fix(tui): suppress redundant ADDED log after snapshot creation`
3. `fix(tui): suppress redundant REMOVED after thinning, log thinning start`
4. `fix(tui): classify ESTALE as warning with user-friendly message`
5. `feat(logger): seed ring buffer from existing log file on startup`
6. `feat(tui): add 'l' key binding to open log directory`
7. `fix(tui): only log refresh summary when data changes`

## Verification

- `make test` and `make lint` pass after each commit
- Manual testing:
  - **Part 0**: Run snappy, verify all log lines show `LEVEL CATEGORY message` format
  - **Part 1**: Start with existing snapshots, verify single "INFO FOUND Found 47 existing snapshots" line
  - **Part 2**: Create a snapshot, verify no duplicate ADDED line
  - **Part 3**: Wait for thin cycle, verify "INFO AUTO Thinning N snapshot(s)..." appears, no duplicate REMOVED lines
  - **Part 4**: If ESTALE occurs, verify "WARN THINNED ..." (yellow) instead of "ERROR THINNED ..." (red)
  - **Part 5**: Quit and restart, verify previous entries appear in log panel
  - **Part 6**: Press `l`, verify Finder opens to log directory
  - **Part 7**: Wait two refresh cycles with no changes, verify only one Refresh line logged
