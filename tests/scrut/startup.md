# Startup behavior

Tests for the binary's behavior when it attempts to run without `--help`,
`--version`, or flag errors. On macOS without a TTY, the binary gets past
the `tmutil` check but fails launching the TUI.

The `Error: running TUI: *` pattern validates that the binary reached the TUI
launch stage (past tmutil check, config loading, logger init, and APFS volume
discovery).

## Bare invocation error

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}"
Error: running TUI: * (glob)
[1]
```

## Bare invocation produces no stdout

```scrut
$ "${SNAPPY_BIN}" 2>/dev/null
[1]
```

## Unknown subcommand rejected

With subcommands registered, Cobra rejects unrecognized positional arguments as
unknown commands instead of passing them to `RunE`.

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" some-argument
Error: unknown command "some-argument" for "snappy"
[1]
```

## Multiple unknown positional arguments

Only the first unrecognized argument is reported.

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" arg1 arg2 arg3
Error: unknown command "arg1" for "snappy"
[1]
```
