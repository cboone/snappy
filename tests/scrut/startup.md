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

## Extra positional argument accepted

Cobra's default argument handling (`legacyArgs`) accepts arbitrary positional
arguments when there are no subcommands.

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" some-argument
Error: running TUI: * (glob)
[1]
```

## Multiple extra positional arguments

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" arg1 arg2 arg3
Error: running TUI: * (glob)
[1]
```
