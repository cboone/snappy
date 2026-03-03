# Thin command

Tests for `snappy thin` subcommand.

## Thin help

```scrut
$ "${SNAPPY_BIN}" thin --help
Thin old snapshots based on configured cadence

Usage:
  snappy thin [flags]

Flags:
  -h, --help   help for thin
      --json   output in JSON format

Global Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
```

## Thin rejects extra arguments

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" thin extra-arg
Error: unknown command "extra-arg" for "snappy thin"
[1]
```
