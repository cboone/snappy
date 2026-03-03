# Plan: snappy-ez simplified snapshot script

## Context

Snappy's full TUI requires Go and a build step. Some users want the core
functionality (create snapshots, thin old ones, log state) without the overhead.
snappy-ez is a standalone bash script they can download and run immediately,
editing hardcoded constants to taste. No flags, no config files, no
dependencies beyond macOS and tmutil.

## Files to modify

1. **Create** `bin/snappy-ez` (new, executable)
2. **Edit** `.gitignore` (add negation so `bin/snappy-ez` is tracked despite
   the `/bin/` exclusion on line 23)
3. **Edit** `README.md` (add snappy-ez section between Installation and Usage)

## 1. `.gitignore` change

Add `!/bin/snappy-ez` immediately after the existing `/bin/` line (line 23).
The negation must come after the exclusion rule.

## 2. `bin/snappy-ez` script

### Structure

```text
Header comment (description, usage examples, customization guide, author, license)
set -euo pipefail
Constants (SNAPSHOT_INTERVAL, THIN_AGE_THRESHOLD, THIN_CADENCE)
Logging (log function, printf to stdout with full timestamp)
Dependency checks (require_macos, require_tmutil)
Date conversion (snapshot_date_to_epoch, using date -j -f)
Snapshot operations (create_snapshot, list_snapshots, thin_snapshots)
Main loop (while true: create, thin, list, sleep)
Entry point (main)
```

### Constants (hardcoded, no env vars, no flags)

| Constant             | Value | Meaning                                       |
| -------------------- | ----- | --------------------------------------------- |
| `SNAPSHOT_INTERVAL`  | `60`  | Seconds between snapshots                     |
| `THIN_AGE_THRESHOLD` | `600` | Snapshots younger than this are never thinned |
| `THIN_CADENCE`       | `300` | Min gap between kept old snapshots            |

Each constant gets an inline comment explaining what it does and how to change
it.

### Logging

Simple `printf` to stdout: `[YYYY-MM-DD HH:MM:SS] EVENT message`. Full
timestamps since the script may run for days. No file logging; users redirect
stdout themselves when running in the background.

### Dependency checks

Two functions following `install.sh` patterns: `require_macos` (checks
`uname -s` == `Darwin`) and `require_tmutil` (checks `command -v tmutil`).
Exit with clear error if either fails.

### Core functions

- **`snapshot_date_to_epoch`**: Converts `YYYY-MM-DD-HHMMSS` to epoch via
  `date -j -f`. Reused from proof-of-concept (line 554).
- **`create_snapshot`**: Runs `tmutil localsnapshot`, logs success or failure.
- **`list_snapshots`**: Runs `tmutil listlocalsnapshotdates /`, filters to
  valid date lines, logs count and each date.
- **`thin_snapshots`**: The thinning algorithm from proof-of-concept
  (lines 483-543):
  1. List snapshots (sorted ascending by tmutil).
  2. Walk oldest-first. Skip if age < `THIN_AGE_THRESHOLD`.
  3. Keep first old snapshot. Keep subsequent old snapshots only if gap from
     last kept >= `THIN_CADENCE`. Mark the rest for deletion.
  4. Delete marked snapshots with `tmutil deletelocalsnapshots <date>`.
  5. Log count of thinned snapshots.

### Main loop

```text
while true:
  create_snapshot
  thin_snapshots
  list_snapshots
  sleep SNAPSHOT_INTERVAL
```

No interactive controls, no screen clearing, no colors. Plain text output.

### Trap handler

`trap cleanup INT TERM` for graceful shutdown on Ctrl-C. Logs "Shutting down."
and exits 0.

### Shell conventions

- Shebang: `#!/usr/bin/env bash`
- `set -euo pipefail`
- `function name() {` style
- `readonly` for constants
- All variables quoted: `"${var}"`
- `[[ ... ]]` conditionals
- Function documentation comments (Arguments, Outputs, Returns)
- Section dividers: `# ===...===`
- 2-space indent, space redirects, binary next line (per `.editorconfig`)

### Inline documentation goals

- Header comment block explains what the script does, both usage modes
  (foreground and background), how to customize, and how thinning works
- Each constant has a multi-line comment explaining its purpose and how to
  change it
- The thinning algorithm has a step-by-step comment block
- Each function has a doc comment

## 3. `README.md` change

Add a new `## snappy-ez` section between Installation (ends at line 45) and
Usage (starts at line 47). Contents:

- Brief description (standalone script, no TUI, no Go required)
- Download command (`curl` from raw GitHub URL + `chmod +x`)
- Foreground usage (`./snappy-ez`, Ctrl-C to stop)
- Background usage (`./snappy-ez >> ~/snappy-ez.log 2>&1 &`, how to monitor
  and stop)
- Customize table (the three constants with defaults and descriptions)

## Verification

1. `shellcheck bin/snappy-ez` passes (with `enable=all` per `.shellcheckrc`)
2. `shfmt -d bin/snappy-ez` produces no diff
3. `make fmt-check` passes
4. `make lint-md` passes for README changes
5. Manual test on macOS: run `bin/snappy-ez`, confirm it creates a snapshot,
   logs state, and exits cleanly on Ctrl-C

## Reference files

- `docs/proof-of-concept/snappy`: Thinning algorithm, date parsing, logging
- `install.sh`: Shell conventions, function doc style, dependency checks
- `.editorconfig`: shfmt settings
- `.shellcheckrc`: ShellCheck config (`enable=all`)
