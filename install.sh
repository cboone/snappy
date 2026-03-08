#!/usr/bin/env bash
# install.sh — Install snappy from GitHub Releases
# Author: Christopher Boone
# Date: 2026-03-01

readonly REPO="cboone/snappy"
readonly BINARY="snappy"
readonly INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"

# Parse command-line arguments.
# Arguments:
#   --version VERSION - Install a specific version (e.g., v1.0.0)
# Sets: VERSION
function parse_args() {
  VERSION=""
  while [[ ${#} -gt 0 ]]; do
    case "${1}" in
      --version)
        if [[ ${#} -lt 2 || -z "${2-}" ]]; then
          printf 'Error: --version requires a non-empty argument.\n' >&2
          printf 'Usage: %s --version VERSION\n' "${0##*/}" >&2
          exit 1
        fi
        VERSION="${2}"
        shift 2
        ;;
      *)
        printf 'Unknown argument: %s\n' "${1}" >&2
        exit 1
        ;;
    esac
  done
}

# Fetch the latest release tag from GitHub.
# Outputs:
#   Writes the tag name to stdout (e.g., "v0.2.0")
# Returns:
#   0 on success, 1 if the tag cannot be determined
function fetch_latest_version() {
  local tag
  local api_url="https://api.github.com/repos/${REPO}/releases/latest"

  local curl_opts=(--fail --silent --show-error --location)
  if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    curl_opts+=(--header "Authorization: Bearer ${GITHUB_TOKEN}"
      --header "Accept: application/vnd.github+json")
  fi

  tag="$(curl "${curl_opts[@]}" "${api_url}" 2> /dev/null \
    | grep '"tag_name"' \
    | sed -E 's/.*"([^"]+)".*/\1/' || true)"

  # Fallback: resolve the /releases/latest redirect to extract the tag.
  if [[ -z "${tag}" ]]; then
    local redirect_url
    redirect_url="$(curl --silent --show-error --head --location \
      --output /dev/null --write-out '%{url_effective}' \
      "https://github.com/${REPO}/releases/latest" 2> /dev/null || true)"

    if [[ -n "${redirect_url}" ]]; then
      tag="$(printf '%s\n' "${redirect_url}" \
        | sed -E 's#.*/tag/([^/?]+).*#\1#' || true)"
    fi
  fi

  if [[ -z "${tag}" ]]; then
    printf 'Error: could not determine latest version.\n' >&2
    return 1
  fi

  printf '%s' "${tag}"
}

# Detect the system architecture.
# Outputs:
#   Writes the normalized architecture to stdout (amd64 or arm64)
# Returns:
#   0 on success, 1 if architecture is unsupported
function detect_arch() {
  local raw_arch
  raw_arch="$(uname -m)"

  case "${raw_arch}" in
    x86_64) printf 'amd64' ;;
    aarch64) printf 'arm64' ;;
    arm64) printf 'arm64' ;;
    *)
      printf 'Unsupported architecture: %s\n' "${raw_arch}" >&2
      return 1
      ;;
  esac
}

# Verify that the current OS is macOS.
# Returns:
#   0 if macOS, 1 otherwise
function require_macos() {
  local os
  os="$(uname -s)"

  if [[ "${os}" != "Darwin" ]]; then
    printf 'Error: snappy requires macOS. Detected OS: %s\n' "${os}" >&2
    return 1
  fi
}

# Verify the checksum of a downloaded file against checksums.txt.
# Arguments:
#   $1 - Path to the downloaded tarball
#   $2 - Expected tarball filename (for matching in checksums.txt)
#   $3 - Path to the checksums file
# Returns:
#   0 on success, 1 on mismatch
function verify_checksum() {
  local tarball_path="${1}"
  local tarball_name="${2}"
  local checksums_path="${3}"

  local expected
  expected="$(awk -v t="${tarball_name}" '$2 == t || $2 == "*" t || $2 == "./" t { print $1 }' "${checksums_path}")"

  if [[ -z "${expected}" ]]; then
    printf 'Error: checksum entry for %s not found in %s; aborting installation.\n' "${tarball_name}" "${checksums_path}" >&2
    return 1
  fi

  local actual
  if command -v sha256sum > /dev/null 2>&1; then
    actual="$(sha256sum "${tarball_path}" | awk '{ print $1 }')"
  elif command -v shasum > /dev/null 2>&1; then
    actual="$(shasum -a 256 "${tarball_path}" | awk '{ print $1 }')"
  else
    printf 'Warning: no sha256 tool found, skipping checksum verification.\n' >&2
    return 0
  fi

  if [[ "${actual}" != "${expected}" ]]; then
    printf 'Checksum mismatch: expected %s, got %s\n' "${expected}" "${actual}" >&2
    return 1
  fi

  printf 'Checksum verified.\n'
}

# Validate that a tarball contains no unsafe paths (absolute or directory traversal).
# Arguments:
#   $1 - Path to the tarball
# Returns:
#   0 if safe, 1 if unsafe paths detected
function validate_archive() {
  local tarball_path="${1}"

  if tar -tzf "${tarball_path}" | grep -qE '(^/|(^|/)\.\.(\/|$))'; then
    printf 'Error: archive contains unsafe paths, refusing to install.\n' >&2
    return 1
  fi
}

# Install shell completion files for available shells.
# Arguments:
#   $1 - Path to the extraction directory containing completions/
# Prints what was installed and notes for skipped shells.
# Best-effort: never fails the overall install.
function install_completions() {
  local extract_dir="${1}"
  local comp_dir="${extract_dir}/completions"

  if [[ ! -d "${comp_dir}" ]]; then
    printf '\nNote: no completion files found in archive. '
    printf 'Generate them manually with: snappy completion <shell>\n'
    return 0
  fi

  printf '\nInstalling shell completions...\n'

  local brew_prefix=""
  if command -v brew > /dev/null 2>&1; then
    brew_prefix="$(brew --prefix)"
  fi

  # Bash completions.
  if [[ -f "${comp_dir}/snappy.bash" ]]; then
    if [[ -n "${brew_prefix}" ]]; then
      mkdir -p "${brew_prefix}/etc/bash_completion.d"
      cp "${comp_dir}/snappy.bash" "${brew_prefix}/etc/bash_completion.d/snappy"
      printf '  bash: %s/etc/bash_completion.d/snappy\n' "${brew_prefix}"
    else
      printf '  bash: skipped (Homebrew not found)\n'
      printf '    To install manually: snappy completion bash > /usr/local/etc/bash_completion.d/snappy\n'
    fi
  fi

  # Zsh completions.
  if [[ -f "${comp_dir}/_snappy" ]]; then
    if [[ -n "${brew_prefix}" ]]; then
      mkdir -p "${brew_prefix}/share/zsh/site-functions"
      cp "${comp_dir}/_snappy" "${brew_prefix}/share/zsh/site-functions/_snappy"
      printf '  zsh:  %s/share/zsh/site-functions/_snappy\n' "${brew_prefix}"
    else
      local zsh_comp_dir="${HOME}/.zsh/completions"
      mkdir -p "${zsh_comp_dir}"
      cp "${comp_dir}/_snappy" "${zsh_comp_dir}/_snappy"
      printf '  zsh:  %s/_snappy\n' "${zsh_comp_dir}"
      printf '    Ensure %s is in your fpath. Add to ~/.zshrc:\n' "${zsh_comp_dir}"
      # ${fpath} is intentionally literal: showing the user what to type.
      # shellcheck disable=SC2016
      printf '      fpath=(%s ${fpath})\n' "${zsh_comp_dir}"
      printf '      autoload -Uz compinit && compinit\n'
    fi
  fi

  # Fish completions.
  if [[ -f "${comp_dir}/snappy.fish" ]]; then
    if command -v fish > /dev/null 2>&1; then
      local fish_comp_dir="${HOME}/.config/fish/completions"
      mkdir -p "${fish_comp_dir}"
      cp "${comp_dir}/snappy.fish" "${fish_comp_dir}/snappy.fish"
      printf '  fish: %s/snappy.fish\n' "${fish_comp_dir}"
    else
      printf '  fish: skipped (fish not installed)\n'
      printf '    To install manually: snappy completion fish > ~/.config/fish/completions/snappy.fish\n'
    fi
  fi

  printf '\nOpen a new shell session to activate completions.\n'
}

function main() {
  parse_args "${@}"

  require_macos

  if [[ -z "${VERSION}" ]]; then
    VERSION="$(fetch_latest_version)"
  fi

  local arch
  arch="$(detect_arch)"

  # GoReleaser strips the v prefix from the version in archive names.
  local version_bare="${VERSION#v}"
  local tarball="${BINARY}_${version_bare}_darwin_${arch}.tar.gz"
  local url="https://github.com/${REPO}/releases/download/${VERSION}/${tarball}"

  # Not local: the EXIT trap references tmp_dir after main() returns.
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "${tmp_dir}"' EXIT

  printf 'Downloading %s %s for darwin/%s...\n' "${BINARY}" "${VERSION}" "${arch}"
  curl --fail --silent --show-error --location --output "${tmp_dir}/${tarball}" "${url}"

  # Verify checksum if checksums file is available.
  local checksums_url="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"
  if curl --fail --silent --show-error --location --output "${tmp_dir}/checksums.txt" "${checksums_url}" 2> /dev/null; then
    printf 'Verifying checksum...\n'
    verify_checksum "${tmp_dir}/${tarball}" "${tarball}" "${tmp_dir}/checksums.txt"
  fi

  validate_archive "${tmp_dir}/${tarball}"

  local extract_dir="${tmp_dir}/extract"
  mkdir -p "${extract_dir}"

  # Extract the binary and completions from the archive.
  tar -xzf "${tmp_dir}/${tarball}" -C "${extract_dir}" -- "${BINARY}"
  tar -xzf "${tmp_dir}/${tarball}" -C "${extract_dir}" -- "completions/" 2> /dev/null || true

  local extracted_binary="${extract_dir}/${BINARY}"

  if [[ ! -f "${extracted_binary}" ]]; then
    printf 'Error: expected binary "%s" not found in archive.\n' "${BINARY}" >&2
    exit 1
  fi
  if [[ -L "${extracted_binary}" ]]; then
    printf 'Error: extracted file "%s" is a symlink; refusing to install.\n' "${BINARY}" >&2
    exit 1
  fi

  mkdir -p "${INSTALL_DIR}"
  install -m 755 "${extracted_binary}" "${INSTALL_DIR}/${BINARY}"

  printf 'Installed %s to %s/%s\n' "${BINARY}" "${INSTALL_DIR}" "${BINARY}"

  install_completions "${extract_dir}"

  # Extract and install the man page if present in the archive.
  local man_page="share/man/man1/snappy.1"
  if tar -tzf "${tmp_dir}/${tarball}" -- "${man_page}" > /dev/null 2>&1; then
    tar -xzf "${tmp_dir}/${tarball}" -C "${extract_dir}" -- "${man_page}"
    local man_dir="${HOME}/.local/share/man/man1"
    mkdir -p "${man_dir}"
    install -m 644 "${extract_dir}/${man_page}" "${man_dir}/snappy.1"
    printf 'Installed man page to %s/snappy.1\n' "${man_dir}"

    # Warn if the man directory is not in MANPATH.
    local man_parent="${HOME}/.local/share/man"
    case ":${MANPATH:-}:" in
      *":${man_parent}:"*) ;;
      *)
        printf '\nNote: %s may not be in your MANPATH.\n' "${man_parent}"
        # ${MANPATH} is intentionally literal: showing the user what to type.
        # shellcheck disable=SC2016
        printf 'Add it with: export MANPATH="%s:${MANPATH}"\n' "${man_parent}"
        ;;
    esac
  fi

  # Warn if the install directory is not in PATH.
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
      printf '\nNote: %s is not in your PATH.\n' "${INSTALL_DIR}"
      # ${PATH} is intentionally literal: showing the user what to type.
      # shellcheck disable=SC2016
      printf 'Add it with: export PATH="%s:${PATH}"\n' "${INSTALL_DIR}"
      ;;
  esac

  printf '\nTo run snappy automatically in the background:\n'
  printf '  %s/%s service install\n\n' "${INSTALL_DIR}" "${BINARY}"
}

# Guard lets callers source this file and test individual functions
# without triggering the full install flow.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  set -euo pipefail
  main "${@}"
fi
