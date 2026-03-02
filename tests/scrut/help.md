# Help output

Tests for snappy help flags and subcommands.

## Root help flag

```scrut
$ "${SNAPPY_BIN}" --help
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [flags]
  snappy [command]

Available Commands:
  help        Help about any command
  version     Print the version number of snappy

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy

Use "snappy [command] --help" for more information about a command.
```

## Short help flag

```scrut
$ "${SNAPPY_BIN}" -h
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [flags]
  snappy [command]

Available Commands:
  help        Help about any command
  version     Print the version number of snappy

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy

Use "snappy [command] --help" for more information about a command.
```

## Help subcommand

The `help` subcommand produces the same output as `--help`.

```scrut
$ "${SNAPPY_BIN}" help
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [flags]
  snappy [command]

Available Commands:
  help        Help about any command
  version     Print the version number of snappy

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy

Use "snappy [command] --help" for more information about a command.
```

## Help for version subcommand

`snappy help version` shows the version subcommand's help text.

```scrut
$ "${SNAPPY_BIN}" help version
Print the version number of snappy

Usage:
  snappy version [flags]

Flags:
  -h, --help   help for version

Global Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
```
