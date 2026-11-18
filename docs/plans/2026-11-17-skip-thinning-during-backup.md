# Skip thinning during Time Machine backups

## Scope and decision

Fix cboone/snappy issue 119 on `fix/skip-thinning-during-backup`, based on
`origin/main` at `581bb36bc634ba1bf6cd2f10f51f2d7ad7c3f835`. Keep all work local.
Do not change the installed binary or user configuration.

Check `tmutil status` immediately before each thinning deletion, after listing
candidates. Stop the pass when `Running = 1`, the command fails, or the Running
field is missing, malformed, or duplicated. Log the skip reason. Retry on the
next pass without pinning undeleted snapshots. Keep snapshot creation, cadence,
CLI JSON, and other existing behavior unchanged.

This uses the supported status command and needs neither persistent ownership
tracking nor private preference keys. Rechecking each deletion covers a backup
starting during a pass. Status checking and deletion are separate system calls;
this does not add synchronization with backupd.

## Mechanism audit and validation

- CLI and service: `cmd/helpers.go` guards each deletion; callers report skips.
- TUI: `internal/tui/commands.go` guards each deletion; the result handler logs
  skips without adding failed dates or leaving thinning active.
- Bash: `bin/snappy-ez` guards each deletion and logs before returning.
- `internal/platform/tmutil.go` supplies the shared Go status guard.
- `docs/proof-of-concept/snappy` is historical reference code, outside this fix.
- Tests cover active and idle backups, status errors, malformed output, backups
  starting between deletions, retry after idle, logging, and unchanged JSON.
- The README, man page, and changelog describe the new pause behavior.

## Defect plant

`thinning-plant.json` declares one plant: disable the active-backup branch in the
shared Go guard. The new running-status regression must fail. The checked runner
uses private source copies, then restores the source and reruns the test.
The committed patch and check scripts reproduce the experiment.

## Results

- `make test-all`: exit 0. Go unit tests passed; 66 CLI Scrut cases and
  17 snappy-ez Scrut cases passed, with zero failed or skipped cases.
- `make lint-all`: exit 0, covering Go, Markdown, Actions, and the man page.
  Mandoc reported an outdated system index for four existing macOS references.
- `make vet`, `make fmt`, and `make format-check`: each exit 0.
- `bash -n bin/snappy-ez` and ShellCheck on the changed Bash script and plant
  scripts: each exit 0.
- The first unit-test run exposed a test-only logger fixture with no retained
  entries; enabling retention corrected it. The first lint run required named
  helper return values; that correction passed the subsequent full checks.
- The additional Git whitespace check flags space-before-tab in standard
  unified-diff context lines inside the deliberate plant patch. Checking the
  production sources and user docs separately exits 0.
- One local code review at Nit checked the deletion paths, skip propagation,
  retry behavior, fixtures, and docs. No findings remained: clean at Nit.
- Implementation and tests: signed commit
  `0e1aaab3e59ef04d74d33f727a13654f6f962f00`.
- No push, PR, GitHub metadata write, installed-binary change, or user
  configuration change was performed.

<!-- checked-plants:start -->

## Verified plants and controls

plant disable-active-backup-protection: named assertion reached; red exit 1; restored; green exit 0.

Evidence: `/Users/ctm/Development/snappy-119/docs/plans/2026-11-17-skip-thinning-plant-results.json`.

<!-- checked-plants:end -->
