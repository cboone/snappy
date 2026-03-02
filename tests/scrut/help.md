# Help output

Tests for snappy help commands.

## Root help

```scrut
$ "${SNAPPY_BIN}" --help
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [flags]

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy
```

## Short help flag

```scrut
$ "${SNAPPY_BIN}" -h
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [flags]

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy
```
