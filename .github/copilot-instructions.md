# GitHub Copilot Instructions for Snappy

For full project conventions, see AGENTS.md in the repository root.

## PR Review

When reviewing pull requests, do not flag the following patterns as issues.
Each is an intentional project convention:

- **APFS data cleared on error**: When APFS detail queries fail, derived fields (e.g., `otherSnapCount`) are intentionally reset to zero values rather than preserving stale data from a previous successful fetch. The APFS error is logged separately so the user is informed about the failure.
- **CI job additions vs. switches**: When a new CI job is added (e.g., `test-scrut`), do not assume it replaces or duplicates a pre-existing job. Check the base branch to verify whether a job with that name already existed before flagging potential duplicates.
