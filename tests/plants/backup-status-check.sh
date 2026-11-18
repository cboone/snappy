#!/usr/bin/env bash
set -euo pipefail
printf 'Checking active-backup protection\n'
./backup-status.test -test.run '^TestThinningSkipReason/running$' -test.v
