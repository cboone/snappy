# Platform detection

Tests for `require_macos()` and `detect_arch()` in `install.sh`.

## require_macos passes on macOS

```scrut
$ source "${INSTALL_SH_BIN}" && require_macos
```

## detect_arch returns a known architecture

The function uses `printf` without a trailing newline.

```scrut
$ source "${INSTALL_SH_BIN}" && detect_arch && echo
amd64|arm64 (regex)
```
