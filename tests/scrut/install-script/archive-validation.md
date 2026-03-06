# Archive validation

Tests for `validate_archive()` in `install.sh`.

## Safe archive passes

```scrut
$ tmp="$(mktemp -d)" && mkdir -p "${tmp}/contents" && printf 'binary' > "${tmp}/contents/snappy" && tar -czf "${tmp}/safe.tar.gz" -C "${tmp}/contents" snappy && source "${INSTALL_SH_BIN}" && validate_archive "${tmp}/safe.tar.gz"
```

## Archive with absolute paths fails

Uses `-P` (preserve absolute paths) to create an archive with entries
starting from the root.

```scrut
$ tmp="$(mktemp -d)" && printf 'binary' > "${tmp}/snappy" && tar -czPf "${tmp}/abs.tar.gz" "${tmp}/snappy" && source "${INSTALL_SH_BIN}" && validate_archive "${tmp}/abs.tar.gz" 2>&1
Error: archive contains unsafe paths, refusing to install.
[1]
```

## Archive with directory traversal fails

```scrut
$ tmp="$(mktemp -d)" && mkdir -p "${tmp}/a/b" && printf 'binary' > "${tmp}/a/b/snappy" && (cd "${tmp}/a/b" && tar -czf "${tmp}/traversal.tar.gz" ../../a/b/snappy) && source "${INSTALL_SH_BIN}" && validate_archive "${tmp}/traversal.tar.gz" 2>&1
Error: archive contains unsafe paths, refusing to install.
[1]
```
