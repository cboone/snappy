# 2026-03-07 Fix Homebrew install: tap migration redirect and cask caveats

## Context

Running `brew install cboone/tap/snappy` fails with:

```text
==> Tapping cask/snappy
Cloning into '/opt/homebrew/Library/Taps/cask/homebrew-snappy'...
Username for 'https://github.com':
```

Homebrew never reaches the actual cask because `tap_migrations.json` in `cboone/homebrew-tap` redirects it to a non-existent repo first.

Separately, the generated `Casks/snappy.rb` has a `caveats do` block with bare string literals. In Ruby, only the last expression in a block is returned, so caveats would show only `"  snappy service uninstall"` instead of the full multi-line text. This is a GoReleaser templating issue.

## Changes

### 1. Remove snappy entry from `tap_migrations.json` (cboone/homebrew-tap repo)

The file currently contains:

```json
{
  "snappy": "cask/snappy"
}
```

This was likely added when the old `Formula/snappy.rb` was removed, intending to redirect formula users to the cask. But `cask/snappy` is not a valid tap reference. Homebrew interprets it as the GitHub repo `cask/homebrew-snappy`, which does not exist. Since the cask still lives in the same tap (`cboone/homebrew-tap/Casks/snappy.rb`), no migration is needed.

**Fix:** Remove the `snappy` entry. If other entries exist, keep them; otherwise the file becomes `{}`.

This change will be made directly in the `cboone/homebrew-tap` repo via `gh` API or a local clone.

### 2. Fix caveats in `.goreleaser.yml` (this repo)

**File:** `.goreleaser.yml` (lines 47-52)

The current multi-line `caveats` YAML string:

```yaml
caveats: |
  To run snappy automatically in the background:
    snappy service install

  To stop the background service:
    snappy service uninstall
```

Gets templated by GoReleaser into:

```ruby
caveats do
  "To run snappy automatically in the background:"
  "  snappy service install"
  ""
  "To stop the background service:"
  "  snappy service uninstall"
end
```

In Ruby, a `do` block returns only the last expression, so only `"  snappy service uninstall"` would be shown as caveats.

**Fix:** Test whether a single-line caveats string (or a different format) produces a working cask. Run `goreleaser release --snapshot --clean` and inspect the generated `Casks/snappy.rb` to determine the best workaround. If GoReleaser always splits multi-line strings into separate expressions, file a GoReleaser issue and use a single-line workaround in the meantime.

## Verification

1. After fixing `tap_migrations.json`, update the local tap: `brew tap --force cboone/tap` or `cd /opt/homebrew/Library/Taps/cboone/homebrew-tap && git pull`
2. Run `brew install cboone/tap/snappy` and confirm it installs successfully (no redirect to `cask/snappy`)
3. Run `brew info --cask cboone/tap/snappy` and confirm caveats display the full multi-line text
4. If a `.goreleaser.yml` change was made, run `goreleaser release --snapshot --clean` and inspect the generated cask file
