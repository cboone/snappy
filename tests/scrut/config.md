# Config flag behavior

Tests for `--config` flag interactions with `--help` and `--version`, plus
warning behavior for nonexistent config files.

## Help with config flag (help first)

`--help` short-circuits before `RunE`, so a bad config path does not prevent
help from succeeding.

```scrut
$ "${SNAPPY_BIN}" --help --config /nonexistent/path/config.yaml
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [flags]

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy
```

## Help with config flag (config first)

Flag order does not matter for `--help` short-circuiting.

```scrut
$ "${SNAPPY_BIN}" --config /nonexistent/path/config.yaml --help
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [flags]

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy
```

## Version with config flag

`--version` also short-circuits before `RunE`.

```scrut
$ "${SNAPPY_BIN}" --version --config /nonexistent/path/config.yaml
snappy version * (glob)
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
