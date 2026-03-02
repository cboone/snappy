# Add help and version subcommands

## Context

Snappy currently only supports `--help`/`-h` and `--version`/`-v` as flags.
Adding `help` and `version` as subcommands improves discoverability and follows
the convention of most CLI tools (e.g., `snappy help`, `snappy version`). The
`help` subcommand will also support `snappy help <subcommand>` for future
subcommands.

## Approach

Cobra auto-generates a `help` subcommand when any subcommand is registered via
`AddCommand()`. So we only need to create a `version` subcommand, and Cobra
handles the rest. We also need to disable Cobra's auto-generated `completion`
subcommand, which would clutter the output.

## Behavioral changes

Adding subcommands changes Cobra's behavior in two notable ways:

1. **Help output expands.** The help text gains an `Available Commands` section,
   a `snappy [command]` usage line, and a footer hint. This affects every scrut
   test that captures help output.

2. **Positional arguments become unknown-command errors.** Without subcommands,
   Cobra uses `legacyArgs` validation which accepts arbitrary positional
   arguments. With subcommands, `snappy some-argument` returns
   `Error: unknown command "some-argument"` instead of reaching the TUI. This
   is better behavior (catches typos) but changes two tests in `startup.md`.

## Implementation

### Step 1: Create `cmd/version.go`

New file with a `version` subcommand that prints the same output as
`snappy --version`. Self-registers via `init()` calling
`rootCmd.AddCommand(versionCmd)`.

Key details:

- Uses `rootCmd.Version` (not the package-level `version` var) to stay in sync
  with the `--version` flag
- Uses `cobra.NoArgs` to reject `snappy version foo`
- Uses `Run` (not `RunE`) since printing cannot fail

### Step 2: Modify `cmd/root.go`

Add one line to `init()`:

```go
rootCmd.CompletionOptions.HiddenDefaultCmd = true
```

This keeps Cobra's auto-generated `completion` subcommand available (e.g.,
`snappy completion bash` works) but hides it from help output.

### Step 3: Update scrut tests

Seven of eight test files contain help output expectations that will change.
The approach: build, run `make test-scrut-update` to auto-capture exact output,
then review each diff for correctness.

Files that need help output updates (the `Available Commands` section and
`[command]` usage line get added):

| File                             | Tests affected     | Nature of change                                              |
| -------------------------------- | ------------------ | ------------------------------------------------------------- |
| `tests/scrut/help.md`            | 2 existing + 2 new | Update format, add `help` and `help version` subcommand tests |
| `tests/scrut/version.md`         | 1 new              | Add `version` subcommand test                                 |
| `tests/scrut/config.md`          | 2 of 6             | Help output tests gain subcommand section                     |
| `tests/scrut/flag-precedence.md` | 4 of 4             | All capture help output                                       |
| `tests/scrut/environment.md`     | 1 of 9             | "Env var with help flag" test                                 |
| `tests/scrut/errors.md`          | 0                  | No help output captured (errors use `SilenceUsage`)           |
| `tests/scrut/startup.md`         | 2 of 4             | Positional args become unknown-command errors                 |

Files with no changes needed:

- `tests/scrut/config-files.md` (no help output, only stderr error patterns)

### Step 4: Handle `startup.md` positional argument changes

The two positional-argument tests change from "reaches TUI stage" to
"unknown command error":

- "Extra positional argument accepted" becomes something like
  "Unknown subcommand rejected" with expected error
  `Error: unknown command "some-argument" for "snappy"`
- "Multiple extra positional arguments" similarly changes with error for
  `"arg1"`

The "Double-dash separator" test in `errors.md` (`snappy -- --help`) also
changes: `--help` after `--` becomes a positional argument, which Cobra now
treats as an unknown command instead of passing to RunE.

## Verification

1. `make build` compiles successfully
2. Manual smoke tests:
   - `./bin/snappy version` prints version
   - `./bin/snappy help` prints help with Available Commands
   - `./bin/snappy help version` prints version subcommand help
   - `./bin/snappy --help` and `./bin/snappy --version` still work
   - `./bin/snappy` still launches the TUI
3. `make test-scrut` passes (after updating expectations)
4. `make test` passes (Go unit tests unaffected)
5. `make lint` passes

## Files

- `cmd/version.go` (new)
- `cmd/root.go` (one line added to `init()`)
- `tests/scrut/help.md`
- `tests/scrut/version.md`
- `tests/scrut/config.md`
- `tests/scrut/flag-precedence.md`
- `tests/scrut/environment.md`
- `tests/scrut/errors.md`
- `tests/scrut/startup.md`
