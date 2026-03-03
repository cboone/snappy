# Run command

Tests for `snappy run` subcommand.

## Run help

```scrut
$ "${SNAPPY_BIN}" run --help
Run the auto-snapshot loop (foreground daemon)

Usage:
  snappy run [flags]

Flags:
  -h, --help   help for run

Global Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
```

## Run rejects --json flag

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" run --json
Error: unknown flag: --json
[1]
```
