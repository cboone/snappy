#!/usr/bin/env bash
set -euo pipefail
printf 'Compiling backup-status regression test\n'
go test -c -o backup-status.test ./internal/platform
