# Thinning ESTALE handling

Tests that `thin_snapshots()` handles exit code 70 (ESTALE) as a skip rather
than an error.

## Exit code 70 logs as a skipped pinned snapshot

```scrut
$ source "${SNAPPY_EZ_BIN}" && tmutil() { if [[ "${1}" == "deletelocalsnapshots" ]]; then return 70; elif [[ "${1}" == "listlocalsnapshotdates" ]]; then printf '%s\n' "Snapshot dates for all disks:" "2026-01-01-100000" "2026-01-01-100100"; else command tmutil "${@}"; fi; } && export -f tmutil && thin_snapshots
[????-??-?? ??:??:??] THIN Skipped pinned snapshot: 2026-01-01-100100 (stale handle) (glob)
[????-??-?? ??:??:??] THIN Thinned 1 snapshot(s) (glob)
```

## Successful deletion logs normally

```scrut
$ source "${SNAPPY_EZ_BIN}" && tmutil() { if [[ "${1}" == "deletelocalsnapshots" ]]; then return 0; elif [[ "${1}" == "listlocalsnapshotdates" ]]; then printf '%s\n' "Snapshot dates for all disks:" "2026-01-01-100000" "2026-01-01-100100"; else command tmutil "${@}"; fi; } && export -f tmutil && thin_snapshots
[????-??-?? ??:??:??] THIN Deleted old snapshot: 2026-01-01-100100 (glob)
[????-??-?? ??:??:??] THIN Thinned 1 snapshot(s) (glob)
```

## Non-zero non-70 exit code logs as an error

```scrut
$ source "${SNAPPY_EZ_BIN}" && tmutil() { if [[ "${1}" == "deletelocalsnapshots" ]]; then return 1; elif [[ "${1}" == "listlocalsnapshotdates" ]]; then printf '%s\n' "Snapshot dates for all disks:" "2026-01-01-100000" "2026-01-01-100100"; else command tmutil "${@}"; fi; } && export -f tmutil && thin_snapshots
[????-??-?? ??:??:??] ERROR Failed to delete snapshot: 2026-01-01-100100 (exit 1) (glob)
[????-??-?? ??:??:??] THIN Thinned 1 snapshot(s) (glob)
```
