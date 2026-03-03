# Reduce GitHub Actions Runner Usage

## Context

Snappy is a private repository, which means GitHub Actions minutes are metered
at a premium. macOS runners cost **10x** more than Linux runners per minute. All
three CI jobs currently run on `macos-latest`, even though most of the work
(linting, formatting checks, Go unit tests) has no macOS dependency. Combined
with the lack of path filtering, every push and PR triggers the full suite,
including documentation-only changes.

This plan restructures CI to minimize macOS runner usage and skip unnecessary
runs entirely.

## Current State

**ci.yml** (triggers: push to main, PRs to main):

| Job | Runner | What it does |
|---|---|---|
| `test` | macos-latest | `make test` (go test ./...) |
| `lint` | macos-latest | go vet, fmt-check, golangci-lint, markdownlint, actionlint, shfmt |
| `test-scrut` | macos-latest | Build binary, run scrut CLI tests + snappy-ez bash tests |

**release.yml** (triggers: tag push `v*`):

| Job | Runner | What it does |
|---|---|---|
| `goreleaser` | macos-latest | Build darwin binaries, create GH release, update Homebrew tap |

**Total: 4 macOS jobs, 0 Linux jobs, no path filtering, no concurrency controls.**

## Analysis

### What can move to Linux

1. **Go unit tests** (`go test ./...`): All test files use a `mockRunner`
   interface to stub macOS commands (tmutil, diskutil, df). No test file has a
   `//go:build darwin` constraint. Only `main.go` does, and Go simply skips the
   root package on Linux (no test files there anyway).

2. **Linting and formatting** (go vet, gofmt, golangci-lint, prettier,
   markdownlint, actionlint, shfmt): All tools are cross-platform. None depend
   on macOS APIs or commands.

### What must stay on macOS

1. **Scrut CLI tests** (`make test-scrut`): The Go binary requires
   `//go:build darwin` to compile. Tests like `startup.md` and `environment.md`
   expect TUI launch behavior that depends on tmutil being present.

2. **Snappy-ez scrut tests** (`make test-scrut-ez`): The bash script uses
   `date -j` (macOS-specific), and `dependency-checks.md` explicitly tests
   `require_macos()` and `require_tmutil()`.

3. **Release** (`goreleaser`): Builds darwin-only binaries. Runs infrequently
   (only on tag pushes), so the cost impact is minimal.

### What can be filtered out

Documentation-only changes (Markdown files, `docs/` directory, community files
like `LICENSE`, `CODE_OF_CONDUCT.md`) do not need Go tests or scrut tests. Only
the Markdown linter is relevant, and even that could be skipped for non-docs PRs
or handled separately.

## Changes

### 1. Move `lint` job to `ubuntu-latest`

**File:** `.github/workflows/ci.yml`, `lint` job

Change `runs-on: macos-latest` to `runs-on: ubuntu-latest`. No other changes
needed; all lint tools (golangci-lint action, raven-actions/actionlint, Node.js
for prettier/markdownlint, shfmt) have Linux support.

### 2. Move `test` job to `ubuntu-latest`

**File:** `.github/workflows/ci.yml`, `test` job

Change `runs-on: macos-latest` to `runs-on: ubuntu-latest`. The `go test ./...`
command will skip the root package (whose only file, `main.go`, has
`//go:build darwin`) and run all other package tests normally. Every test uses
the `CommandRunner` mock interface, so no macOS commands are invoked.

### 3. Add `paths-ignore` to skip docs-only changes

**File:** `.github/workflows/ci.yml`, workflow-level triggers

Add `paths-ignore` to both `push` and `pull_request` triggers:

```yaml
on:
  push:
    branches: [main]
    paths-ignore:
      - "**.md"
      - "docs/**"
      - "LICENSE"
      - ".github/ISSUE_TEMPLATE/**"
      - ".github/PULL_REQUEST_TEMPLATE/**"
      - "CODE_OF_CONDUCT.md"
      - "CONTRIBUTING.md"
      - "SECURITY.md"
  pull_request:
    branches: [main]
    paths-ignore:
      - "**.md"
      - "docs/**"
      - "LICENSE"
      - ".github/ISSUE_TEMPLATE/**"
      - ".github/PULL_REQUEST_TEMPLATE/**"
      - "CODE_OF_CONDUCT.md"
      - "CONTRIBUTING.md"
      - "SECURITY.md"
```

**Trade-off:** If any CI job is a _required_ status check in branch protection,
`paths-ignore` will prevent the check from reporting, blocking the PR from
merging. If branch protection requires these checks, we have two options:

- (a) Remove them as required checks (simplest).
- (b) Keep them required but use a job-level `if` condition with
  `dorny/paths-filter` instead of workflow-level `paths-ignore`, so the job
  always runs but skips its steps. This reports a "success" status to branch
  protection even when skipped.

### 4. Add concurrency controls

**File:** `.github/workflows/ci.yml`

Add a `concurrency` block at the workflow level to cancel in-progress runs when
a new commit is pushed to the same PR branch:

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

This prevents wasting minutes on outdated runs during rapid push cycles.

### 5. Keep `test-scrut` on macOS (no change)

This job must stay on `macos-latest`. Both `make test-scrut` (requires
compiling the darwin binary) and `make test-scrut-ez` (requires macOS date
command and tmutil) need macOS.

### 6. Keep `release` on macOS (no change)

Runs only on tag pushes, so the cost impact is negligible. While goreleaser
_could_ cross-compile from Linux (CGO_ENABLED=0), keeping this on macOS avoids
risk for infrequent releases. Not worth changing.

## Expected Impact

| Job | Before | After | Savings |
|---|---|---|---|
| `test` | macos (10x) | Linux (1x) | ~90% |
| `lint` | macos (10x) | Linux (1x) | ~90% |
| `test-scrut` | macos (10x) | macos (10x) | 0% |
| Docs-only runs | Full CI | Skipped | 100% |
| Stale PR runs | Run to completion | Cancelled | Variable |

Net result: **2 of 3 CI jobs move from 10x to 1x cost**, docs-only changes skip
CI entirely, and stale runs are cancelled. Estimated ~60-70% reduction in
overall CI minute spend.

## Files to Modify

- `.github/workflows/ci.yml`: All changes (runner OS, paths-ignore, concurrency)

## Verification

1. Push a branch with Go code changes and confirm:
   - `test` and `lint` jobs run on `ubuntu-latest` (visible in Actions UI)
   - `test-scrut` job runs on `macos-latest`
   - All three jobs pass
2. Push a docs-only commit (only `.md` files changed) and confirm CI does not
   trigger
3. Push two commits in quick succession to a PR branch and confirm the first
   run is cancelled
4. Run `make lint-actions` locally to validate the workflow file syntax
