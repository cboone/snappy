# Thinning during backups

## Active backup

```scrut
$ source "${SNAPPY_EZ_BIN}" && tmutil() { case "$1" in listlocalsnapshotdates) printf '%s\n' 2026-01-01-100000 2026-01-01-100100 ;; status) printf '    Running = 1;\n' ;; deletelocalsnapshots) echo 'UNSAFE DELETION' ;; *) return 99 ;; esac; } && thin_snapshots
[????-??-?? ??:??:??] THIN Thinning skipped: Time Machine backup is running (glob)
```

## Unknown backup status

```scrut
$ source "${SNAPPY_EZ_BIN}" && tmutil() { case "$1" in listlocalsnapshotdates) printf '%s\n' 2026-01-01-100000 2026-01-01-100100 ;; status) return 1 ;; deletelocalsnapshots) echo 'UNSAFE DELETION' ;; *) return 99 ;; esac; } && thin_snapshots
[????-??-?? ??:??:??] THIN Thinning skipped: cannot determine Time Machine backup status (glob)
```

## Malformed backup status

```scrut
$ source "${SNAPPY_EZ_BIN}" && tmutil() { case "$1" in listlocalsnapshotdates) printf '%s\n' 2026-01-01-100000 2026-01-01-100100 ;; status) printf 'NotRunning = 0;\n' ;; deletelocalsnapshots) echo 'UNSAFE DELETION' ;; *) return 99 ;; esac; } && thin_snapshots
[????-??-?? ??:??:??] THIN Thinning skipped: cannot determine Time Machine backup status (invalid Running field) (glob)
```

## Backup starts during thinning

The deletion stub starts the backup after the first deletion. All commands use
shell functions, so the test cannot delete real snapshots.

```scrut
$ source "${SNAPPY_EZ_BIN}" && running=0 && tmutil() { case "$1" in listlocalsnapshotdates) printf '%s\n' 2026-01-01-100000 2026-01-01-100100 2026-01-01-100200 ;; status) printf '    Running = %s;\n' "${running}" ;; deletelocalsnapshots) if [[ "${running}" == 1 ]]; then echo 'UNSAFE DELETION'; fi; running=1 ;; *) return 99 ;; esac; } && thin_snapshots
[????-??-?? ??:??:??] THIN Deleted old snapshot: 2026-01-01-100100 (glob)
[????-??-?? ??:??:??] THIN Thinning skipped: Time Machine backup is running (glob)
```
