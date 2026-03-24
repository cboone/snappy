# Improve `snappy list` human-readable output

Issue: #96

## Context

The `snappy list` human-readable output is hard to scan: fields concatenated on one line with inconsistent spacing, no column alignment, no header row, and the `delta:N` inline label format. The TUI snapshot panel already uses aligned columns with headers (DATE, AGE, XID, DELTA, UUID, STATUS). This plan brings the CLI output to parity with that layout, plus adds graceful degradation when APFS data is unavailable and a `--no-header` flag for piping.

## Design decisions

**Table formatting approach:** Manual column width computation (not `text/tabwriter`), because `tabwriter` does not support per-column alignment (right for numbers, left for strings). One pass computes max widths, another renders. This matches what the TUI does.

**Formatting logic location:** Extract a small `columnTable` type in `cmd/table.go`. Keeps `writeListHuman()` focused on data extraction and makes alignment testable independently. Lightweight: ~60 lines.

**`--no-header` with zero snapshots:** Still prints the summary line (`"0 snapshot(s) on /"`). The flag only suppresses the column header row, not the summary.

## Changes

### 1. Create `cmd/table.go` - column table builder

Small unexported type:

```go
type columnAlign int // alignLeft, alignRight

type columnDef struct {
    title string
    align columnAlign
}

type columnTable struct {
    cols []columnDef
    rows [][]string
}
```

Methods:

- `addRow(cells ...string)` - appends a row
- `render(w io.Writer, showHeader bool) error` - computes max width per column (headers + cells), writes header (if enabled) then data rows. Left-aligned: `fmt.Sprintf("%-*s", w, val)`. Right-aligned: `fmt.Sprintf("%*s", w, val)`. Two-space column separator. Two-space row indentation.

### 2. Create `cmd/table_test.go`

Test cases: left/right alignment, width computation from widest cell, `showHeader=false`, empty table with header only.

### 3. Modify `cmd/list.go`

**Add `--no-header` flag** in `init()`:

```go
listCmd.Flags().Bool("no-header", false, "suppress column header row")
```

**Add helpers** near `formatRelativeAgo()`:

```go
func formatHumanDate(t time.Time) string {
    return t.Format("2006-01-02 15:04:05")
}

func formatStatus(s snapshot.Snapshot) string {
    // Returns "purgeable", "pinned", or "pinned, limits shrink"
}
```

**Rewrite `writeListHuman()`** signature to accept `noHeader bool`:

1. Print summary line (`"N snapshot(s) on /"`), return early if empty
2. Detect APFS data: scan for any snapshot with non-empty UUID
3. Build column definitions:
   - With APFS: `#` (right), DATE (left), AGE (left), XID (right), DELTA (right), UUID (left), STATUS (left)
   - Without APFS: `#` (right), DATE (left), AGE (left)
4. Iterate newest-first, extract cell values using `formatHumanDate()`, `formatRelativeAgo()`, `formatStatus()`
5. Render with `ct.render(w, !noHeader)`

Date format changes from `"2026-03-01-140100"` to `"2026-03-01 14:01:00"`.
Status format changes from `"pinned   limits shrink"` to `"pinned, limits shrink"`.
Delta changes from `"delta:50"` to `"50"` in its own column.

### 4. Update `cmd/list_test.go`

**Update existing tests:**

- `TestListHumanWithSnapshots`: Check for human date format `"2026-03-01 14:01:00"`, check header contains `DATE` and `AGE` but not `XID` or `UUID` (no APFS data)
- `TestListHumanWithAPFSDetails`: Check for `DELTA` header, `50` as column value (not `delta:50`), `pinned, limits shrink` with comma, `ABC-123` UUID

**Add new tests:**

- `TestListHumanNoHeader` - verifies summary line present but no `DATE`/`AGE` header labels
- `TestListHumanNoHeaderWithAPFS` - same with APFS data
- `TestListHumanColumnAlignment` - verifies consistent column positions across rows

### 5. Update `tests/scrut/list-command.md`

- Update help output to include `--no-header` flag
- Add test for `snappy list --no-header | head -1` still producing summary line

### 6. Clean up `cmd/mock_test.go`

Remove `nolint:unparam` annotation from `setFlag()` since it will now be called with both `"json"` and `"no-header"` names. (The annotation says "name is always json today" which will no longer be true.)

### 7. Lint and format

Run `make lint` and `make fmt`.

## Files to modify

| File | Action |
| ------- | ---------- |
| `cmd/table.go` | Create |
| `cmd/table_test.go` | Create |
| `cmd/list.go` | Modify |
| `cmd/list_test.go` | Modify |
| `cmd/mock_test.go` | Modify (remove nolint) |
| `tests/scrut/list-command.md` | Modify |

## Reusable code

- `snapshot.FormatRelativeTime()` from `internal/snapshot/snapshot.go` (already used)
- `formatRelativeAgo()` from `cmd/list.go` (already used, unchanged)
- TUI column definitions at `internal/tui/update.go:1108-1138` as reference for column names and widths
- TUI row building at `internal/tui/update.go:1040-1066` as reference for data extraction logic

## Verification

1. `make build` - compiles successfully
2. `make test` - all Go unit tests pass (updated assertions + new tests)
3. `make test-scrut` - scrut CLI tests pass (updated help output + new `--no-header` test)
4. `make lint` - no lint errors
5. Manual: `./bin/snappy list` shows aligned columns with header
6. Manual: `./bin/snappy list --no-header` shows aligned columns without header
7. Manual: `./bin/snappy list --json` output unchanged
8. Manual: on a system without APFS snapshots, columns degrade to `#`, `DATE`, `AGE` only
