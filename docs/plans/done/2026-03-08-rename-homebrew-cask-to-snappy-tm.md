# Rename Homebrew cask from snappy to snappy-tm

## Context

The Homebrew cask is currently named `snappy`, which conflicts with an existing
homebrew-core formula for Google's snappy compression library. Renaming the cask
to `snappy-tm` eliminates the ambiguity. The binary name remains `snappy`.

## Changes

### 1. `.goreleaser.yml` -- add `name` field

Add `name: snappy-tm` as the first property under the `homebrew_casks` list
item (before `repository:`):

```yaml
homebrew_casks:
  - name: snappy-tm
    repository:
      owner: cboone
      ...
```

Everything else in the section stays the same. The `binaries: [snappy]`,
`caveats`, `hooks`, and `uninstall` entries all reference the binary name or
launchd label, not the cask name.

### 2. `README.md` -- update install commands

Change all three occurrences of `brew install cboone/tap/snappy` to
`brew install cboone/tap/snappy-tm`:

- Line 11 (intro paragraph)
- Line 45 (Quick Start code block)
- Line 135 (Installation section code block)

### 3. Tap repo (`/Users/ctm/Development/homebrew-tap`) -- rename cask and update migrations

Create a branch, make changes, push, and open a PR via `gh pr create`.

#### a. Rename cask file

Rename `Casks/snappy.rb` to `Casks/snappy-tm.rb`. Inside the file, update:

- `cask "snappy"` to `cask "snappy-tm"`
- `name "snappy"` to `name "snappy-tm"`

Everything else (version, sha256, URLs, postflight, caveats, uninstall) stays
the same. GoReleaser will overwrite the file contents on the next release
anyway.

#### b. Update `tap_migrations.json`

Change the migration target from `"cask/snappy"` to `"cask/snappy-tm"`:

```json
{
  "snappy": "cask/snappy-tm"
}
```

This ensures users who previously had the formula get redirected to the renamed
cask.

#### c. Regenerate README

Run `bin/update-readme` to regenerate the auto-generated sections. The script
reads `Casks/*.rb` dynamically, so the table, install commands, and notes will
automatically reflect the `snappy-tm` name.

## Not changing

- **Binary name** (`snappy`) -- unchanged
- **LaunchD label** (`com.cboone.snappy`) -- unchanged
- **Service log filename** (`snappy-service.log`) -- unchanged
- **install.sh** -- references binary name and GitHub repo, not the cask
- **Go source code** -- no cask references
- **Historical docs/plans/CHANGELOG** -- historical records, left as-is

## Verification

1. Run `goreleaser check` to validate `.goreleaser.yml` syntax
2. In the tap repo, run `bin/update-readme --check` after regeneration to
   confirm the README is consistent
3. After release: confirm `brew install cboone/tap/snappy-tm` works
4. Confirm `brew install cboone/tap/snappy` no longer resolves to a stale cask

## Migration note for existing users

Users who previously installed via `brew install cboone/tap/snappy` will need
to uninstall and reinstall:

```sh
brew uninstall cboone/tap/snappy
brew install cboone/tap/snappy-tm
```

Consider mentioning this in the release notes.
