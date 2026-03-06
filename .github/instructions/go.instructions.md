---
applyTo: "**/*.go"
---

# Go Code Guidelines

- **Hardcoded program name in output**: Snappy is a small, single-binary CLI tool. Hardcoding "snappy" in user-facing output (e.g., version strings) is intentional. Do not suggest deriving the program name dynamically from `cmd.Root().Name()` or similar. The added indirection is unnecessary for this project.
- **Build constraints and `./...` on Linux**: Only `main.go` has `//go:build darwin`. With Go 1.21+, `go test ./...`, `go vet ./...`, and `golangci-lint run ./...` silently skip packages where all files are excluded by build constraints. Do not suggest adding `!darwin` stub files or narrowing the package list to exclude the root package. The `./...` pattern works correctly on Linux as-is.
- **Log format parsing in `parseLogLine`**: The old log format used category names (STARTUP, REFRESH, SNAPSHOT, etc.) as the first token after the timestamp. The new format prepends a level (INFO, WARN, ERROR) before the category. The `switch` in `parseLogLine` correctly disambiguates these formats because no old category name overlaps with the level constants. Do not suggest adding `isKnownCategory` guards or secondary validation; the existing logic handles both formats correctly.
