# Add a Detailed Man Page

## Context

The README already promises `man snappy` works after Homebrew install (lines 60, 64, 142), but no man page exists. This plan creates a hand-written `snappy(1)` man page in mdoc format, integrates it into the build/release pipeline, and adds local development tooling.

## Decisions

- **Hand-written, not auto-generated.** Cobra's `cobra/doc` produces bare-bones pages lacking EXAMPLES, CONFIGURATION, FILES, and INTERACTIVE MODE sections. The CLI surface is small and stable; maintenance cost is low.
- **Single page: `snappy(1)`.** ~10 subcommands and 3 global flags fit comfortably in one page, following the pattern of `tmutil(8)`.
- **mdoc format.** The modern BSD/macOS semantic macro set, native to the platform Snappy targets. More readable as source than raw roff.
- **Location: `docs/snappy.1`.** Keeps the man page alongside other project docs.

## Steps

### 1. Create `docs/snappy.1`

Write the man page in mdoc format with these sections:

| Section | Content |
|---------|---------|
| NAME | `snappy -- automatically increase your Time Machine snapshot frequency` |
| SYNOPSIS | Multiple lines: bare `snappy` (TUI), `snappy command [options]`, `snappy service subcommand`, `snappy config [init]` |
| DESCRIPTION | What Snappy does, APFS snapshots via tmutil, macOS-only, TUI vs non-interactive |
| COMMANDS | Tagged list of all commands: `create`, `list`, `status`, `thin`, `run`, `config`, `config init`, `service` (with all 6 subcommands), `version`, `help`. Each with flags and behavior. |
| GLOBAL OPTIONS | `--config`, `-h/--help`, `-v/--version` |
| INTERACTIVE MODE | TUI panels (info, snapshot, log), key bindings (s, r, a, l, Tab, Shift+Tab, j/k, q), mouse support |
| CONFIGURATION | All 8 settings with defaults and descriptions, YAML format, config file path |
| ENVIRONMENT | `SNAPPY_`-prefixed env vars for all settings |
| FILES | Config file, launchd plist, service log, shared log, lock file paths |
| EXIT STATUS | 0 success, non-zero error |
| EXAMPLES | Quick start, create, list --json, status, custom config, service log |
| SEE ALSO | `tmutil(8)`, `diskutil(8)`, `launchctl(1)`, `asr(8)` |
| AUTHORS | Christopher Boone |
| BUGS | GitHub issues URL |

Source content from the README and Cobra command definitions in `cmd/*.go`.

### 2. Update Makefile

Add targets:

- `man` -- preview with `man ./docs/snappy.1`
- `man-lint` -- validate with `mandoc -Tlint docs/snappy.1`

Add `man-lint` to the `lint` target (after `lint-actions`). Update `.PHONY`.

### 3. Update `.goreleaser.yml`

Include the man page in release archives:

```yaml
archives:
  - formats:
      - tar.gz
    name_template: "..."
    files:
      - LICENSE
      - README.md
      - src: docs/snappy.1
        dst: share/man/man1
        strip_parent: true
```

Note: adding `files` overrides GoReleaser's default inclusion of LICENSE/README, so list them explicitly.

For the Homebrew cask, check whether GoReleaser v2 supports a `manpages` field on `homebrew_casks`. If so, add:

```yaml
homebrew_casks:
  - manpages:
      - share/man/man1/snappy.1
```

If not, add a `system_command` post-install hook to copy the man page:

```yaml
hooks:
  post:
    install: |
      system_command "/usr/bin/install", args: ["-m", "644",
        "#{staged_path}/share/man/man1/snappy.1",
        "/opt/homebrew/share/man/man1/snappy.1"]
```

Verify the generated cask with `goreleaser release --snapshot --skip=publish`.

### 4. Update CI (`.github/workflows/ci.yml`)

Remove `"docs/**"` from `paths-ignore` in both `push` and `pull_request` triggers. The `docs/` directory now contains a distributable artifact (the man page), not just documentation prose.

Add a man page lint step to the `test-scrut` job (macOS runner, mandoc is preinstalled):

```yaml
- name: Lint man page
  run: mandoc -Tlint docs/snappy.1
```

### 5. Update `install.sh`

After extracting the binary, conditionally extract and install the man page:

```bash
if tar -tzf "${tmp_dir}/${tarball}" -- "share/man/man1/snappy.1" >/dev/null 2>&1; then
  tar -xzf "${tmp_dir}/${tarball}" -C "${extract_dir}" -- "share/man/man1/snappy.1"
  local man_dir="${HOME}/.local/share/man/man1"
  mkdir -p "${man_dir}"
  install -m 644 "${extract_dir}/share/man/man1/snappy.1" "${man_dir}/snappy.1"
  printf 'Installed man page to %s/snappy.1\n' "${man_dir}"
fi
```

Older release archives without the man page are handled gracefully by the conditional.

### 6. Update `README.md`

Add man page references where appropriate:

- Line 60 already mentions "shell completions and a man page" -- no change needed.
- Line 64 already mentions `man snappy` -- no change needed.
- Line 118 says "installs the `snappy` binary and shell completions" but omits the man page. Update to include "and a man page".
- Line 142 already mentions `man snappy` -- no change needed.

## Files to modify

| File | Change |
|------|--------|
| `docs/snappy.1` | **Create.** The man page itself. |
| `Makefile` | Add `man`, `man-lint` targets; add `man-lint` to `lint`. |
| `.goreleaser.yml` | Add `files` to `archives`; add man page to cask install. |
| `.github/workflows/ci.yml` | Remove `docs/**` from `paths-ignore`; add `mandoc -Tlint` step to `test-scrut` job. |
| `README.md` | Add man page mention to line 118 (Homebrew install description). |
| `install.sh` | Extract and install man page when present in archive. |

## Verification

1. `mandoc -Tlint docs/snappy.1` reports no warnings
2. `man ./docs/snappy.1` renders correctly in the terminal
3. `make man-lint` passes
4. `make lint` passes (including man-lint)
5. `goreleaser release --snapshot --skip=publish` produces an archive containing the man page at the expected path
6. Existing tests still pass (`make test-all`)
