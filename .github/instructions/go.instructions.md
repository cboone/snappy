---
applyTo: "**/*.go"
---

# Go Code Guidelines

- **Hardcoded program name in output**: Snappy is a small, single-binary CLI tool. Hardcoding "snappy" in user-facing output (e.g., version strings) is intentional. Do not suggest deriving the program name dynamically from `cmd.Root().Name()` or similar. The added indirection is unnecessary for this project.
- **Build constraints and `./...` on Linux**: Only `main.go` has `//go:build darwin`. With Go 1.21+, `go test ./...`, `go vet ./...`, and `golangci-lint run ./...` silently skip packages where all files are excluded by build constraints. Do not suggest adding `!darwin` stub files or narrowing the package list to exclude the root package. The `./...` pattern works correctly on Linux as-is.
- **Log format parsing in `parseLogLine`**: Historical logs may use either the old format (`[ts] EventType message`) or the new format (`[ts] Level Category message`). The old EventType values `INFO` and `ERROR` overlap with the new Level constants, so `parseLogLine` validates the second token against `isKnownCategory` to disambiguate. If the second token is a known Category, the line is parsed as the new format; otherwise, the first token is treated as an old EventType and the rest is the message. Do not remove the `isKnownCategory` guard; it is required for correct backward-compatible parsing.
