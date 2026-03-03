# List command

Tests for `snappy list` subcommand.

## List help

```scrut
$ "${SNAPPY_BIN}" list --help
List local snapshots with details

Usage:
  snappy list [flags]

Flags:
  -h, --help   help for list
      --json   output in JSON format

Global Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
```

## List default invocation shows snapshot count

```scrut
$ "${SNAPPY_BIN}" list | head -1
* snapshot(s) on / (glob)
```

## List JSON produces valid JSON

```scrut
$ "${SNAPPY_BIN}" list --json | head -1
{
```

## List rejects extra arguments

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" list extra-arg
Error: unknown command "extra-arg" for "snappy list"
[1]
```
