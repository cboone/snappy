# GitHub Copilot Instructions for Snappy

For full project conventions, see AGENTS.md in the repository root.

## PR Review

When reviewing pull requests, do not flag the following patterns as issues.
Each is an intentional project convention:

- **APFS data cleared on error**: When APFS detail queries fail, derived fields (e.g., `otherSnapCount`) are intentionally reset to zero values rather than preserving stale data from a previous successful fetch. The APFS error is logged separately so the user is informed about the failure.
- **CI job additions vs. switches**: When a new CI job is added (e.g., `test-scrut`), do not assume it replaces or duplicates a pre-existing job. Check the base branch to verify whether a job with that name already existed before flagging potential duplicates.
- **Install script served from `main` branch**: The `curl | bash` install pattern in the README intentionally references the `main` branch, not a pinned tag. This is the standard industry pattern (used by nvm, Homebrew, rustup, etc.) so users always get the latest installer logic. The downloaded binary is pinned to a specific versioned release.
- **Info panel shows 3 fixed lines**: The TUI info panel is intentionally limited to 3 lines (Volume/Disk, Time Machine/Refresh, Auto-snapshot) for a predictable layout. APFS volume details and non-TM snapshot counts are logged in the log panel instead. The model tracks `apfsVolume` and `lastOtherSnapCount` for log messages, not for info panel display. Do not flag this as a missing feature or regression.
- **Scrut tests source install.sh without strict mode**: The install script's `set -euo pipefail` is intentionally placed inside the source guard (`if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then`). When scrut tests `source install.sh` and call functions directly, strict mode is not active. This is by design: individual functions use explicit error handling (return codes and stderr output) rather than relying on `set -e`. Do not suggest wrapping test commands in `bash -euo pipefail -c '...'`.
