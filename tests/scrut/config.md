# Config flag behavior

Tests for `--config` flag interactions with `--help` and `--version`, plus
warning behavior for nonexistent config files.

## Help with config flag (help first)

When `--help` precedes `--config <value>`, Cobra validates flag names (catching
unknown flags like `--bogus`) but short-circuits before consuming flag value
arguments. The orphaned path becomes an unknown-command error. Use equals syntax
(`--config=<path>`) or place `--config` before `--help` to avoid this.

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" --help --config /nonexistent/path/config.yaml
Error: unknown command "/nonexistent/path/config.yaml" for "snappy"
[1]
```

## Help with config flag (config first)

When `--config` precedes `--help`, the flag value is consumed before `--help`
triggers.

```scrut
$ "${SNAPPY_BIN}" --config /nonexistent/path/config.yaml --help
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [command] [flags]

Available Commands:
  config      Show or manage snappy configuration
  create      Create a new local Time Machine snapshot
  help        Help about any command
  list        List local snapshots with details
  run         Run the auto-snapshot loop as a foreground service
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

## Version with config flag (version first)

Same traversal issue as `--help`: `--version` before `--config <value>` leaves
the path orphaned as an unknown command.

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" --version --config /nonexistent/path/config.yaml
Error: unknown command "/nonexistent/path/config.yaml" for "snappy"
[1]
```

## Config flag with equals syntax

`--config=value` works the same as `--config value`.

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" --config=/nonexistent/path/config.yaml
Warning: config file error: * (glob)
Error: running TUI: * (glob)
[1]
```

## Version with config equals syntax

```scrut
$ "${SNAPPY_BIN}" --version --config=/nonexistent/path/config.yaml
snappy version * (glob)
```

## Nonexistent config file warning

When `--config` points to a file that does not exist, `initConfig()` prints a
warning to stderr, then execution continues to `RunE` which fails at TUI
launch without a TTY.

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" --config /nonexistent/path/config.yaml
Warning: config file error: * (glob)
Error: running TUI: * (glob)
[1]
```
