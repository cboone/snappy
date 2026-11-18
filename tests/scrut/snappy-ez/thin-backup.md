# Thinning during backups

## Active backup

```scrut
$ source "${SNAPPY_EZ_BIN}" && deletions=0 && tmutil() { case "$1" in listlocalsnapshotdates) printf '%s\n' 2026-01-01-100000 2026-01-01-100100 ;; status) printf '    Running = 1;\n' ;; deletelocalsnapshots) deletions=$((deletions + 1)) ;; *) return 99 ;; esac; } && thin_snapshots && echo "deletions: ${deletions}"
[????-??-?? ??:??:??] THIN Thinning skipped: Time Machine backup is running (glob)
deletions: 0
```

## Unknown backup status

```scrut
$ source "${SNAPPY_EZ_BIN}" && deletions=0 && tmutil() { case "$1" in listlocalsnapshotdates) printf '%s\n' 2026-01-01-100000 2026-01-01-100100 ;; status) echo 'tmutil: permission denied' >&2; return 77 ;; deletelocalsnapshots) deletions=$((deletions + 1)) ;; *) return 99 ;; esac; } && thin_snapshots && echo "deletions: ${deletions}"
[????-??-?? ??:??:??] THIN Thinning skipped: cannot determine Time Machine backup status (exit 77): tmutil: permission denied (glob)
deletions: 0
```

## Malformed backup status

```scrut
$ source "${SNAPPY_EZ_BIN}" && deletions=0 && tmutil() { case "$1" in listlocalsnapshotdates) printf '%s\n' 2026-01-01-100000 2026-01-01-100100 ;; status) printf 'NotRunning = 0;\n' ;; deletelocalsnapshots) deletions=$((deletions + 1)) ;; *) return 99 ;; esac; } && thin_snapshots && echo "deletions: ${deletions}"
[????-??-?? ??:??:??] THIN Thinning skipped: cannot determine Time Machine backup status (invalid Running field) (glob)
deletions: 0
```

## Backup starts during thinning

The deletion stub counts calls and starts the backup after the first deletion. All commands use
shell functions, so the test cannot delete real snapshots.

```scrut
$ source "${SNAPPY_EZ_BIN}" && deletions=0 && running=0 && tmutil() { case "$1" in listlocalsnapshotdates) printf '%s\n' 2026-01-01-100000 2026-01-01-100100 2026-01-01-100200 ;; status) printf '    Running = %s;\n' "${running}" ;; deletelocalsnapshots) deletions=$((deletions + 1)); running=1 ;; *) return 99 ;; esac; } && thin_snapshots && echo "deletions: ${deletions}"
[????-??-?? ??:??:??] THIN Deleted old snapshot: 2026-01-01-100100 (glob)
[????-??-?? ??:??:??] THIN Thinning skipped: Time Machine backup is running (glob)
deletions: 1
```
