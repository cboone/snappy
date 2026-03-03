# Startup behavior

Tests the startup log line by sourcing snappy-ez and overriding `run_loop` to
prevent the infinite loop.

## STARTUP message includes parameters with default values

```scrut
$ source "${SNAPPY_EZ_BIN}" && run_loop() { :; } && main
[????-??-?? ??:??:??] STARTUP snappy-ez started (interval=60s, thin_age=600s, thin_cadence=300s) (glob)
```
