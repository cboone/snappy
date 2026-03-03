# Shutdown behavior

Tests the `cleanup()` trap handler.

## Cleanup produces SHUTDOWN log line

```scrut
$ source "${SNAPPY_EZ_BIN}" && cleanup
[????-??-?? ??:??:??] SHUTDOWN Shutting down. (glob)
```
