# Config file scenarios

Tests for config file content and path edge cases beyond the nonexistent path
(covered in `config.md`).

## Config pointing to a directory

When `--config` points to a directory instead of a file, `viper.ReadInConfig()`
fails and a warning is printed.

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" --config /tmp
Warning: config file error: * (glob)
Error: running TUI: * (glob)
[1]
```

## Empty config file via /dev/null

```scrut {output_stream: stderr}
$ "${SNAPPY_BIN}" --config /dev/null
Warning: config file error: * (glob)
Error: running TUI: * (glob)
[1]
```

## Empty config file produces no stdout

```scrut
$ "${SNAPPY_BIN}" --config /dev/null 2>/dev/null
[1]
```

## Config flag combined with env var

Validates the full config pipeline: env var provides `mount`, `/dev/null`
provides an empty config file.

```scrut {output_stream: stderr}
$ SNAPPY_MOUNT="/Volumes/Test" "${SNAPPY_BIN}" --config /dev/null
Warning: config file error: * (glob)
Error: running TUI: * (glob)
[1]
```
