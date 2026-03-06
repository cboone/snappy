# Environment variables

Tests for `SNAPPY_*` environment variable integration via
`viper.SetEnvPrefix("SNAPPY")` and `viper.AutomaticEnv()`.

## SNAPPY_LOG_DIR env var affects logger setup

Setting `SNAPPY_LOG_DIR` to a path under `/dev/null` forces logger directory
creation to fail. This warning only appears if the env var is read and applied.

```scrut {output_stream: stderr}
$ SNAPPY_LOG_DIR="/dev/null/snappy" "${SNAPPY_BIN}"
Warning: cannot create log directory /dev/null/snappy: * (glob)
Error: running TUI: * (glob)
[1]
```

## SNAPPY_MOUNT env var reaches TUI stage

```scrut {output_stream: stderr}
$ SNAPPY_MOUNT="/Volumes/Test" "${SNAPPY_BIN}"
Error: running TUI: * (glob)
[1]
```

## SNAPPY_MOUNT env var produces no stdout

```scrut
$ SNAPPY_MOUNT="/Volumes/Test" "${SNAPPY_BIN}" 2>/dev/null
[1]
```

## SNAPPY_AUTO_ENABLED=false reaches TUI stage

```scrut {output_stream: stderr}
$ SNAPPY_AUTO_ENABLED=false "${SNAPPY_BIN}"
Error: running TUI: * (glob)
[1]
```

## SNAPPY_REFRESH with numeric value

```scrut {output_stream: stderr}
$ SNAPPY_REFRESH=30 "${SNAPPY_BIN}"
Error: running TUI: * (glob)
[1]
```

## SNAPPY_REFRESH with Go duration string

```scrut {output_stream: stderr}
$ SNAPPY_REFRESH="2m" "${SNAPPY_BIN}"
Error: running TUI: * (glob)
[1]
```

## Multiple SNAPPY env vars together

```scrut {output_stream: stderr}
$ SNAPPY_MOUNT="/Volumes/Test" SNAPPY_REFRESH=30 SNAPPY_AUTO_ENABLED=false "${SNAPPY_BIN}"
Error: running TUI: * (glob)
[1]
```

## Env var with help flag

```scrut
$ SNAPPY_MOUNT="/Volumes/Test" "${SNAPPY_BIN}" --help
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [command] [flags]

Available Commands:
  config      Show or manage snappy configuration
  create      Create a new local Time Machine snapshot
  help        Help about any command
  list        List local snapshots with details
  run         Run the auto-snapshot loop (foreground daemon)
  status      Show Time Machine and disk status
  thin        Thin old snapshots based on configured cadence
  version     Print the version number of snappy

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy

Use "snappy [command] --help" for more information about a command.
```

## Env var with version flag

```scrut
$ SNAPPY_MOUNT="/Volumes/Test" "${SNAPPY_BIN}" --version
snappy version * (glob)
```
