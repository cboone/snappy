# Invalid CLI input

Tests for error handling when invalid flags are passed.

`SilenceUsage: true` ensures that only the error line appears on stderr,
not the full help text.

## Unknown long flag

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" --bogus
Error: unknown flag: --bogus
[1]
```

## Unknown short flag

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" -z
Error: unknown shorthand flag: 'z' in -z
[1]
```

## Config flag without value

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" --config
Error: flag needs an argument: --config
[1]
```

## Unknown flag after help flag

Cobra parses all flags before dispatching help, so `--bogus` still triggers
an error even though `--help` is present.

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" --help --bogus
Error: unknown flag: --bogus
[1]
```

## Double-dash separator

`--` stops flag parsing; `--help` is treated as a positional argument, so the
binary proceeds to the TUI stage and fails without a TTY. Unlike bare positional
arguments (e.g., `snappy foo`), arguments after `--` bypass Cobra's subcommand
matching, so they reach `RunE` even when subcommands are registered.

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" -- --help
Error: running TUI: * (glob)
[1]
```

## Empty config value

An empty string for `--config` is equivalent to omitting the flag, since
`initConfig()` treats `""` as unset.

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" --config ""
Error: running TUI: * (glob)
[1]
```

## Unknown flag produces no stdout

```scrut
$ "${SNAPPY_BIN}" --unknown-flag 2>/dev/null
[1]
```
