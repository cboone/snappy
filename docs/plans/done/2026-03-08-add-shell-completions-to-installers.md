# Add shell completions to installers

## Context

The README claims Homebrew installs shell completions, but no installer actually does.
The hidden Cobra `completion` command works (`snappy completion bash` produces valid
output), but no completion files are generated, packaged, or installed. This plan adds
completion files to the release archives and installers, and documents manual setup.

The `completion` command stays hidden from `snappy --help`. No Go changes needed.

## Step 1: Add `make completions` target

Add a Makefile target that builds the binary, then generates four completion scripts
into `completions/`:

```makefile
COMPDIR := completions

completions: build
 mkdir -p $(COMPDIR)
 $(OUTDIR)/$(BINARY) completion bash > $(COMPDIR)/snappy.bash
 $(OUTDIR)/$(BINARY) completion zsh > $(COMPDIR)/_snappy
 $(OUTDIR)/$(BINARY) completion fish > $(COMPDIR)/snappy.fish
 $(OUTDIR)/$(BINARY) completion powershell > $(COMPDIR)/snappy.ps1
```

Add `completions/` to `.gitignore` (generated artifacts, not source).

Update `clean` target to remove the `completions/` directory.

**File:** `Makefile`, `.gitignore`

## Step 2: Include completions in GoReleaser archives

Add a `before` hook to generate completions, and include them in archives.

In `.goreleaser.yml`:

```yaml
before:
  hooks:
    - make completions
```

Update the `archives` section to include the completion files:

```yaml
archives:
  - formats:
      - tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - completions/*
```

This puts a `completions/` directory in every release tarball alongside the binary.

**File:** `.goreleaser.yml`

## Step 3: Update `install.sh` to install completions

After extracting the binary, also extract and install the completion files. The
install script already extracts to a temp directory, so we extend it to:

1. Extract `completions/*` from the archive (in addition to the binary)
2. Detect which shells are available and install to standard locations:
   - bash: `$(brew --prefix)/etc/bash_completion.d/snappy` if Homebrew is present,
     otherwise skip with a note
   - zsh: `$(brew --prefix)/share/zsh/site-functions/_snappy` if Homebrew is present,
     otherwise `~/.zsh/completions/_snappy` as a fallback
   - fish: `~/.config/fish/completions/snappy.fish` if fish is installed
3. Print what was installed and what wasn't (with manual instructions for skipped shells)

Keep the completion installation best-effort: if a shell isn't installed or a target
directory doesn't exist, print a note but don't fail.

**File:** `install.sh`

## Step 4: Update Homebrew cask to install completions

The Homebrew cask post-install hook in `.goreleaser.yml` currently only removes
quarantine. Add `system_command` calls to copy completion files from the staged
archive to Homebrew's completion directories:

```yaml
hooks:
  post:
    install: |
      system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/snappy"]
      system_command "/bin/mkdir", args: ["-p", "#{HOMEBREW_PREFIX}/etc/bash_completion.d"]
      system_command "/bin/cp", args: ["#{staged_path}/completions/snappy.bash", "#{HOMEBREW_PREFIX}/etc/bash_completion.d/snappy"]
      system_command "/bin/mkdir", args: ["-p", "#{HOMEBREW_PREFIX}/share/zsh/site-functions"]
      system_command "/bin/cp", args: ["#{staged_path}/completions/_snappy", "#{HOMEBREW_PREFIX}/share/zsh/site-functions/_snappy"]
      system_command "/bin/mkdir", args: ["-p", "#{HOMEBREW_PREFIX}/share/fish/vendor_completions.d"]
      system_command "/bin/cp", args: ["#{staged_path}/completions/snappy.fish", "#{HOMEBREW_PREFIX}/share/fish/vendor_completions.d/snappy.fish"]
```

**Risk:** GoReleaser's cask template may not support multiple `system_command` calls
or `HOMEBREW_PREFIX` in hooks. If this doesn't work during testing, fall back to
documenting manual completion setup for Homebrew users and file a follow-up issue for
switching from a cask to a Homebrew formula (which has native `bash_completion.install`
support).

**File:** `.goreleaser.yml`

## Step 5: Update README

1. **Command Summary table** (~line 157): add a row for `snappy completion <shell>`.

2. **Replace the "Shell completions" TODO** (line 421) with a "Shell Completions"
   section documenting:
   - Homebrew and install.sh handle completions automatically
   - Manual setup for each shell (pointing to `snappy completion <shell>` usage)
   - Note that a new shell session is needed after installation

3. **Fix the inaccurate claims** on lines 60 and 118 if completions are not yet
   reliably installed via Homebrew (depends on Step 4 outcome).

**File:** `README.md`

## Step 6: Update scrut tests for install.sh

If `install.sh` gains completion installation output, update the relevant scrut tests
in `tests/scrut/install-script/`.

**File:** `tests/scrut/install-script/*.md`

## Files

| File                              | Action |
| --------------------------------- | ------ |
| `Makefile`                        | Modify |
| `.gitignore`                      | Modify |
| `.goreleaser.yml`                 | Modify |
| `install.sh`                      | Modify |
| `README.md`                       | Modify |
| `tests/scrut/install-script/*.md` | Update |

No Go code changes. No changes to `cmd/`. No scrut help output changes (the
`completion` command remains hidden).

## Verification

1. `make completions` generates four files in `completions/`
2. Spot-check: `source completions/snappy.bash` in bash, tab-complete `snappy` subcommands
3. `goreleaser release --snapshot --clean` produces archives containing `completions/`
4. `make test-scrut-install` passes (after updating expectations)
5. `make test-all` passes (no Go changes, so unit + scrut tests unaffected)
6. `make lint` passes
