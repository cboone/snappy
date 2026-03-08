# Completion installation

Tests for `install_completions()` in `install.sh`.

## No completions directory prints a note

```scrut
$ tmp="$(mktemp -d)" && source "${INSTALL_SH_BIN}" && install_completions "${tmp}"

Note: no completion files found in archive. Generate them manually with: snappy completion <shell>
```

## Installs zsh completions to fallback directory without Homebrew

Runs with a PATH that excludes Homebrew to simulate Homebrew not being installed.

```scrut
$ tmp="$(mktemp -d)" && mkdir -p "${tmp}/completions" && printf 'zsh-comp' > "${tmp}/completions/_snappy" && PATH="/usr/bin:/bin" source "${INSTALL_SH_BIN}" && HOME="${tmp}/fakehome" PATH="/usr/bin:/bin" install_completions "${tmp}" && cat "${tmp}/fakehome/.zsh/completions/_snappy"

Installing shell completions...
  zsh:  */fakehome/.zsh/completions/_snappy (glob)
    Ensure */fakehome/.zsh/completions is in your fpath. Add to ~/.zshrc: (glob)
      fpath=(*/fakehome/.zsh/completions ${fpath}) (glob)
      autoload -Uz compinit && compinit

Open a new shell session to activate completions.
zsh-comp (no-eol)
```

## Installs fish completions when fish is available

```scrut
$ tmp="$(mktemp -d)" && mkdir -p "${tmp}/completions" && printf 'fish-comp' > "${tmp}/completions/snappy.fish" && mkdir -p "${tmp}/fakebin" && printf '#!/usr/bin/env bash\nexit 0\n' > "${tmp}/fakebin/fish" && chmod +x "${tmp}/fakebin/fish" && PATH="${tmp}/fakebin:/usr/bin:/bin" source "${INSTALL_SH_BIN}" && HOME="${tmp}/fakehome" PATH="${tmp}/fakebin:/usr/bin:/bin" install_completions "${tmp}" && cat "${tmp}/fakehome/.config/fish/completions/snappy.fish"

Installing shell completions...
  fish: */fakehome/.config/fish/completions/snappy.fish (glob)

Open a new shell session to activate completions.
fish-comp (no-eol)
```

## Skips fish when not installed

```scrut
$ tmp="$(mktemp -d)" && mkdir -p "${tmp}/completions" && printf 'fish-comp' > "${tmp}/completions/snappy.fish" && PATH="/usr/bin:/bin" source "${INSTALL_SH_BIN}" && HOME="${tmp}/fakehome" PATH="/usr/bin:/bin" install_completions "${tmp}"

Installing shell completions...
  fish: skipped (fish not installed)
    To install manually: snappy completion fish > ~/.config/fish/completions/snappy.fish

Open a new shell session to activate completions.
```

## Skips bash when Homebrew is not available

```scrut
$ tmp="$(mktemp -d)" && mkdir -p "${tmp}/completions" && printf 'bash-comp' > "${tmp}/completions/snappy.bash" && PATH="/usr/bin:/bin" source "${INSTALL_SH_BIN}" && PATH="/usr/bin:/bin" install_completions "${tmp}"

Installing shell completions...
  bash: skipped (Homebrew not found)
    To install manually: snappy completion bash > /usr/local/etc/bash_completion.d/snappy

Open a new shell session to activate completions.
```
