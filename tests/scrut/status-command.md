# Status command

Tests for `snappy status` subcommand.

## Status help

```scrut
$ "${SNAPPY_BIN}" status --help
Show Time Machine and disk status

Usage:
  snappy status [flags]

Flags:
  -h, --help   help for status
      --json   output in JSON format

Global Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
```

## Status default invocation shows Time Machine line

```scrut
$ "${SNAPPY_BIN}" status | head -1
Time Machine: * (glob)
```

## Status shows mount point

```scrut
$ "${SNAPPY_BIN}" status | grep Mount
Mount: /
```

## Status JSON produces valid JSON

```scrut
$ "${SNAPPY_BIN}" status --json | head -1
{
```

## Status rejects extra arguments

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" status extra-arg
Error: unknown command "extra-arg" for "snappy status"
[1]
```
