# Dependency checks

Tests `require_macos()` and `require_tmutil()` on a macOS host.

## require_macos exits 0 on macOS

```scrut
$ source "${SNAPPY_EZ_BIN}" && require_macos
```

## require_tmutil exits 0 when tmutil is present

```scrut
$ source "${SNAPPY_EZ_BIN}" && require_tmutil
```
