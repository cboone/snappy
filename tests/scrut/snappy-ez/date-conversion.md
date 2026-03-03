# Date conversion

Tests the `snapshot_date_to_epoch()` function.

## Valid snapshot date produces a non-zero epoch

```scrut
$ source "${SNAPPY_EZ_BIN}" && result=$(snapshot_date_to_epoch "2026-03-02-143000") && test "${result}" -gt 0 && echo "ok"
ok
```

## Invalid input returns zero

```scrut
$ source "${SNAPPY_EZ_BIN}" && snapshot_date_to_epoch "not-a-date"
0
```

## Empty string returns zero

```scrut
$ source "${SNAPPY_EZ_BIN}" && snapshot_date_to_epoch ""
0
```
