# Checksum verification

Tests for `verify_checksum()` in `install.sh`.

## Matching checksum passes

```scrut
$ tmp="$(mktemp -d)" && printf 'hello' > "${tmp}/file.tar.gz" && hash="$(shasum -a 256 "${tmp}/file.tar.gz" | awk '{ print $1 }')" && printf '%s  file.tar.gz\n' "${hash}" > "${tmp}/checksums.txt" && source "${INSTALL_SH_BIN}" && verify_checksum "${tmp}/file.tar.gz" "file.tar.gz" "${tmp}/checksums.txt"
Checksum verified.
```

## Mismatched checksum fails

```scrut
$ tmp="$(mktemp -d)" && printf 'hello' > "${tmp}/file.tar.gz" && printf '%s  file.tar.gz\n' "0000000000000000000000000000000000000000000000000000000000000000" > "${tmp}/checksums.txt" && source "${INSTALL_SH_BIN}" && verify_checksum "${tmp}/file.tar.gz" "file.tar.gz" "${tmp}/checksums.txt" 2>&1
Checksum mismatch: expected 0000000000000000000000000000000000000000000000000000000000000000, got * (glob)
[1]
```

## Missing entry in checksums file fails

```scrut
$ tmp="$(mktemp -d)" && printf 'hello' > "${tmp}/file.tar.gz" && printf '%s  other.tar.gz\n' "abc123" > "${tmp}/checksums.txt" && source "${INSTALL_SH_BIN}" && verify_checksum "${tmp}/file.tar.gz" "file.tar.gz" "${tmp}/checksums.txt" 2>&1
Error: checksum entry for file.tar.gz not found in */checksums.txt; aborting installation. (glob)
[1]
```
