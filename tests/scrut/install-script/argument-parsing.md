# Argument parsing

Tests for `parse_args()` in `install.sh`.

## No arguments leaves VERSION empty

```scrut
$ source "${INSTALL_SH_BIN}" && parse_args && echo "VERSION='${VERSION}'"
VERSION=''
```

## --version sets VERSION

```scrut
$ source "${INSTALL_SH_BIN}" && parse_args --version v1.2.3 && echo "VERSION='${VERSION}'"
VERSION='v1.2.3'
```

## --version without value shows error

```scrut
$ source "${INSTALL_SH_BIN}" && parse_args --version 2>&1
Error: --version requires a non-empty argument.
Usage: * --version VERSION (glob)
[1]
```

## Unknown argument shows error

```scrut
$ source "${INSTALL_SH_BIN}" && parse_args --bogus 2>&1
Unknown argument: --bogus
[1]
```
