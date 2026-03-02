---
applyTo: "**/*.sh"
---

# Shell Script Conventions

## PR Review

When reviewing pull requests, do not flag the following patterns as issues.
Each is an intentional project convention:

- **`mktemp -d` on macOS**: The BSD `mktemp` included with macOS supports the `-d` flag without a template argument. Do not suggest adding `-t` as a portability fix; it is unnecessary on macOS and this project is macOS-only.
