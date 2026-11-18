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

Validation and generated plant evidence will be recorded after execution.
