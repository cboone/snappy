# Log output format

Tests the `log()` function output format: `[YYYY-MM-DD HH:MM:SS] EVENT message`.

## Log line format with STARTUP event

```scrut
$ source "${SNAPPY_EZ_BIN}" && log "STARTUP" "hello world"
[????-??-?? ??:??:??] STARTUP hello world (glob)
```

## Log line format with ERROR event

```scrut
$ source "${SNAPPY_EZ_BIN}" && log "ERROR" "something failed"
[????-??-?? ??:??:??] ERROR something failed (glob)
```

## Log message content preserved exactly

```scrut
$ source "${SNAPPY_EZ_BIN}" && log "SNAPSHOT" "Created: 2026-03-02-143000"
[????-??-?? ??:??:??] SNAPSHOT Created: 2026-03-02-143000 (glob)
```
