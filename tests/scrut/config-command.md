# Config command

Tests for `snappy config` and `snappy config init` subcommands.

## Config show displays current configuration

```scrut
$ "${SNAPPY_BIN}" config
Config file: none

refresh: 1m0s
mount: /
log_dir: */.local/share/snappy (glob)
log_max_size: 5242880
log_max_files: 3
auto_enabled: true
auto_snapshot_interval: 1m0s
thin_age_threshold: 10m0s
thin_cadence: 5m0s
```

## Config show with env override

```scrut
$ SNAPPY_MOUNT="/Volumes/Test" "${SNAPPY_BIN}" config
Config file: none

refresh: 1m0s
mount: /Volumes/Test
log_dir: */.local/share/snappy (glob)
log_max_size: 5242880
log_max_files: 3
auto_enabled: true
auto_snapshot_interval: 1m0s
thin_age_threshold: 10m0s
thin_cadence: 5m0s
```

## Config init creates default file

```scrut
$ rm -f /tmp/snappy-scrut-config-test.yaml && "${SNAPPY_BIN}" config init --config /tmp/snappy-scrut-config-test.yaml
Config file created: /tmp/snappy-scrut-config-test.yaml
```

## Config init file contains comments

```scrut
$ grep -c '^#' /tmp/snappy-scrut-config-test.yaml
12
```

## Config init fails when file exists

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" config init --config /tmp/snappy-scrut-config-test.yaml
Error: config file already exists: /tmp/snappy-scrut-config-test.yaml
[1]
```

## Config init cleanup

```scrut
$ rm -f /tmp/snappy-scrut-config-test.yaml
```

## Config help

```scrut
$ "${SNAPPY_BIN}" config --help
Show or manage snappy configuration

Usage:
  snappy config [flags]
  snappy config [command]

Available Commands:
  init        Create a default config file

Flags:
  -h, --help   help for config

Global Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)

Use "snappy config [command] --help" for more information about a command.
```

## Config init help

```scrut
$ "${SNAPPY_BIN}" config init --help
Create a default config file

Usage:
  snappy config init [flags]

Flags:
  -h, --help   help for init

Global Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
```
