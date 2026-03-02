# Version output

Tests for snappy version flags and subcommands.

## Version flag

```scrut
$ "${SNAPPY_BIN}" --version
snappy version * (glob)
```

## Short version flag

```scrut
$ "${SNAPPY_BIN}" -v
snappy version * (glob)
```

## Version subcommand

The `version` subcommand produces the same output as `--version`.

```scrut
$ "${SNAPPY_BIN}" version
snappy version * (glob)
```
