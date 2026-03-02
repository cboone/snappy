# Flag precedence

Documents flag interaction behavior when multiple mode-switching flags are
combined. Locks down Cobra's precedence so version upgrades do not silently
change behavior.

## Help flag wins over version flag

```scrut
$ "${SNAPPY_BIN}" --help --version
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [flags]

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy
```

## Version flag before help flag

Flag order does not matter: `--help` still takes precedence.

```scrut
$ "${SNAPPY_BIN}" --version --help
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [flags]

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy
```

## Short help and version flags

```scrut
$ "${SNAPPY_BIN}" -v -h
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [flags]

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy
```

## All three flags together

`--help` wins even when `--version` and `--config` are also present.

```scrut
$ "${SNAPPY_BIN}" --help --version --config /dev/null
Automatically increase your Time Machine snapshot frequency

Usage:
  snappy [flags]

Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
  -h, --help            help for snappy
  -v, --version         version for snappy
```
