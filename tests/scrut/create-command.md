# Create command

Tests for `snappy create` subcommand.

## Create help

```scrut
$ "${SNAPPY_BIN}" create --help
Create a new local Time Machine snapshot

Usage:
  snappy create [flags]

Flags:
  -h, --help   help for create
      --json   output in JSON format

Global Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
```

## Create rejects extra arguments

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" create extra-arg
Error: unknown command "extra-arg" for "snappy create"
[1]
```
