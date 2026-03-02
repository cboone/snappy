---
applyTo: "**/*.go"
---

# Go Code Guidelines

- **Hardcoded program name in output**: Snappy is a small, single-binary CLI tool. Hardcoding "snappy" in user-facing output (e.g., version strings) is intentional. Do not suggest deriving the program name dynamically from `cmd.Root().Name()` or similar. The added indirection is unnecessary for this project.
