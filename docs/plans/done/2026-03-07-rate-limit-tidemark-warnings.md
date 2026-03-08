# Rate-limit repeated tidemark WARN logs

Issue: #86

## Context

`applyTidemark` in `internal/tui/update.go` logs a WARN on every refresh cycle
(default 60 seconds) when tidemark retrieval fails. A persistent failure (e.g.,
unsupported volume or container) fills the ring buffer with duplicate warnings,
reducing visibility of other log messages.

The codebase already uses a "log on change" pattern in several places:

- `logRefreshSummary()` tracks `m.lastRefreshSummary` and only logs when the
  summary string changes.
- `applyAPFSInfo()` tracks `m.lastOtherSnapCount` and only logs when the count
  changes.

The fix applies the same pattern to tidemark errors.

## Changes

### 1. Add a `lastTidemarkErr` field to `Model`

File: `internal/tui/model.go`

Add a `lastTidemarkErr string` field to the `Model` struct, grouped with the
other "last seen" state fields (`lastOtherSnapCount`, `lastRefreshSummary`).

No initialization needed (zero value `""` is correct for "no previous error").

### 2. Deduplicate tidemark warnings in `applyTidemark`

File: `internal/tui/update.go`

Update `applyTidemark` to only log when the error message differs from the
previously seen error:

```go
func (m *Model) applyTidemark(msg RefreshResultMsg) {
    errMsg := ""
    if msg.TidemarkErr != nil {
        errMsg = msg.TidemarkErr.Error()
    }
    if errMsg != "" && errMsg != m.lastTidemarkErr {
        m.log.Log(logger.LevelWarn, logger.CatRefresh,
            "Tidemark fetch failed: "+errMsg)
    }
    m.lastTidemarkErr = errMsg

    if msg.Tidemark > 0 {
        m.tidemark = platform.FormatBytes(msg.Tidemark)
    } else {
        m.tidemark = ""
    }
}
```

Behavior:

- First failure: logged at WARN (errMsg differs from initial `""`)
- Repeated identical failure: suppressed (errMsg == lastTidemarkErr)
- Different failure reason: logged again (new errMsg)
- Error clears then recurs: logged again (lastTidemarkErr was reset to `""`)

### 3. Update the existing test

File: `internal/tui/model_test.go`

`TestTidemarkFetchFailureLogged` (line 2025) already verifies the first warning
is logged. Extend it to confirm that a second refresh with the same error does
NOT produce a duplicate warning entry.

### 4. Add a test for changed error messages

File: `internal/tui/model_test.go`

Add a test that verifies: when the tidemark error message changes between
refreshes, the new error IS logged.

## Verification

1. `make test` -- unit tests pass, including the updated/new tidemark tests
2. `make lint` -- no lint issues introduced
3. `make build` -- binary compiles cleanly
