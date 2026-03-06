# Flag precedence

Documents flag interaction behavior when multiple mode-switching flags are
combined. Locks down Cobra's precedence so version upgrades do not silently
change behavior.

## Help flag wins over version flag

```scrut
$ "${SNAPPY_BIN}" --help --version
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [command] [flags]

Available Commands:
  config      Show or manage snappy configuration
  create      Create a new local Time Machine snapshot
  help        Help about any command
  list        List local snapshots with details
  run         Run the auto-snapshot loop (foreground daemon)
  service     Manage the snappy background service (launchd)
  status      Show Time Machine and disk status
  thin        Thin old snapshots based on configured cadence
  version     Print the version number of snappy

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy

Use "snappy [command] --help" for more information about a command.
```

## Version flag before help flag

Flag order does not matter: `--help` still takes precedence.

```scrut
$ "${SNAPPY_BIN}" --version --help
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [command] [flags]

Available Commands:
  config      Show or manage snappy configuration
  create      Create a new local Time Machine snapshot
  help        Help about any command
  list        List local snapshots with details
  run         Run the auto-snapshot loop (foreground daemon)
  service     Manage the snappy background service (launchd)
  status      Show Time Machine and disk status
  thin        Thin old snapshots based on configured cadence
  version     Print the version number of snappy

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy

Use "snappy [command] --help" for more information about a command.
```

## Short help and version flags

```scrut
$ "${SNAPPY_BIN}" -v -h
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [command] [flags]

Available Commands:
  config      Show or manage snappy configuration
  create      Create a new local Time Machine snapshot
  help        Help about any command
  list        List local snapshots with details
  run         Run the auto-snapshot loop (foreground daemon)
  service     Manage the snappy background service (launchd)
  status      Show Time Machine and disk status
  thin        Thin old snapshots based on configured cadence
  version     Print the version number of snappy

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy

Use "snappy [command] --help" for more information about a command.
```

## All three flags together

`--help` wins even when `--version` and `--config` are also present.

```scrut
$ "${SNAPPY_BIN}" --help --version --config /dev/null
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [command] [flags]

Available Commands:
  config      Show or manage snappy configuration
  create      Create a new local Time Machine snapshot
  help        Help about any command
  list        List local snapshots with details
  run         Run the auto-snapshot loop (foreground daemon)
  service     Manage the snappy background service (launchd)
  status      Show Time Machine and disk status
  thin        Thin old snapshots based on configured cadence
  version     Print the version number of snappy

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy

Use "snappy [command] --help" for more information about a command.
```
