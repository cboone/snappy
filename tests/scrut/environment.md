# Environment variables

Tests for `SNAPPY_*` environment variable integration via
`viper.SetEnvPrefix("SNAPPY")` and `viper.AutomaticEnv()`.

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
  snappy [flags]

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy
```

## Env var with version flag

```scrut
$ SNAPPY_MOUNT="/Volumes/Test" "${SNAPPY_BIN}" --version
snappy version * (glob)
```
