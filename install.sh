#!/usr/bin/env bash
#
# ling-box (玲珑盒) installer
#
# Detects the host OS/architecture, fetches the matching precompiled binary
# from the latest GitHub release, and installs it.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/wgzhao/ling-box/master/install.sh | bash
#
# Options:
#   --install-dir <dir>   Install into <dir> (default: $LINGBOX_INSTALL_DIR,
#                         then $XDG_BIN_HOME, $XDG_DATA_HOME/../bin,
#                         ~/.local/bin, ~/bin)
#   -h, --help            Show this help
#
# Environment:
#   LINGBOX_VERSION   Version to install (e.g. "0.5.0" or "v0.5.0");
#                     default "latest"
#   LINGBOX_INSTALL_DIR   Same as --install-dir

set -euo pipefail

REPO="wgzhao/ling-box"
BIN_NAME="lingbox"

usage() {
  cat <<'EOF'
ling-box (玲珑盒) installer

Detects the host OS/architecture, fetches the matching precompiled binary
from the latest GitHub release, and installs it.

Usage: curl -fsSL https://raw.githubusercontent.com/wgzhao/ling-box/master/install.sh | bash

Options:
  --install-dir <dir>   Install into <dir> (default: $LINGBOX_INSTALL_DIR,
                        then $XDG_BIN_HOME, $XDG_DATA_HOME/../bin,
                        ~/.local/bin, ~/bin)
  -h, --help            Show this help

Environment:
  LINGBOX_VERSION       Version to install (e.g. "0.5.0" or "v0.5.0");
                        default "latest"
  LINGBOX_INSTALL_DIR   Same as --install-dir
EOF
  exit "${1:-0}"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h | --help) usage ;;
    --install-dir=*) LINGBOX_INSTALL_DIR="${1#*=}" ;;
    --install-dir)
      shift
      if [[ -z "${1:-}" || "$1" == -* ]]; then
        echo "error: --install-dir requires a directory argument" >&2
        exit 1
      fi
      LINGBOX_INSTALL_DIR="$1"
      ;;
    -*) echo "error: unknown option: $1" >&2; usage 1 ;;
    *) echo "error: unexpected argument: $1" >&2; usage 1 ;;
  esac
  shift
done

# ---------------------------------------------------------------------------
# 1. Detect host OS and architecture
# ---------------------------------------------------------------------------

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux | darwin) ;;
  *)
    echo "error: unsupported OS '$OS'. ling-box ships precompiled binaries for" >&2
    echo "       Linux and macOS; on Windows grab lingbox-windows-x86-64.exe" >&2
    echo "       from https://github.com/$REPO/releases/latest" >&2
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64 | amd64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  armv7l | armv6l)
    echo "error: no precompiled binary for 32-bit ARM. Build from source:" >&2
    echo "       https://github.com/$REPO#building" >&2
    exit 1
    ;;
  *)
    echo "error: unsupported architecture '$ARCH'" >&2
    exit 1
    ;;
esac

# Map to the release asset naming convention: lingbox-<os>-<arch>
# (darwin binaries are published under "macos", amd64 under "x86-64").
case "$OS" in
  darwin) REL_OS="macos" ;;
  *) REL_OS="$OS" ;;
esac
case "$ARCH" in
  amd64) REL_ARCH="x86-64" ;;
  *) REL_ARCH="$ARCH" ;;
esac
ASSET="lingbox-${REL_OS}-${REL_ARCH}"

# ---------------------------------------------------------------------------
# 2. Resolve the release and matching asset
# ---------------------------------------------------------------------------

# Prefer curl; fall back to wget. Both get bounded timeouts so the script
# fails cleanly instead of hanging on unreachable hosts.
if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL --connect-timeout 15 --max-time 300 "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO- --timeout=15 --tries=3 "$1"; }
else
  echo "error: neither curl nor wget is available" >&2
  exit 1
fi

VERSION="${LINGBOX_VERSION:-latest}"
case "$VERSION" in
  latest | "") RELEASE_URL="https://api.github.com/repos/$REPO/releases/latest" ;;
  v*) RELEASE_URL="https://api.github.com/repos/$REPO/releases/tags/$VERSION" ;;
  *) RELEASE_URL="https://api.github.com/repos/$REPO/releases/tags/v$VERSION" ;;
esac

echo "==> Resolving release ($VERSION)..."
RELEASE_JSON="$(fetch "$RELEASE_URL")"

# Extract the tag name (e.g. "v0.5.0").
TAG="$(printf '%s' "$RELEASE_JSON" | tr ',' '\n' | sed -n 's/.*"tag_name": "\([^"]*\)".*/\1/p' | head -n1)"
if [[ -z "$TAG" ]]; then
  echo "error: could not resolve release '$VERSION'" >&2
  echo "       check https://github.com/$REPO/releases" >&2
  exit 1
fi

echo "==> Found $TAG; looking for $ASSET"

# Walk the assets array; when the object's name matches our asset, capture
# its browser_download_url and sha256 digest. Field order within the object
# is not guaranteed, so collect both in any order and emit on the next object
# (or at EOF) as a single space-separated line. Portable awk: no
# backreferences, values split on double quotes.
read -r DOWNLOAD_URL DIGEST < <(printf '%s' "$RELEASE_JSON" | awk -v want="$ASSET" '
  /"name":/ {
    if (found) { print url, dg; done = 1; exit }
    split($0, a, "\"")
    found = (a[4] == want) ? 1 : 0
    url = ""
    dg = ""
    next
  }
  found && /"browser_download_url":/ {
    split($0, a, "\"")
    url = a[4]
    next
  }
  found && /"digest":/ {
    split($0, a, "\"")
    dg = a[4]
    sub(/^sha256:/, "", dg)
    next
  }
  END { if (found && !done) print url, dg }
') || true

if [[ -z "$DOWNLOAD_URL" ]]; then
  echo "error: no asset '$ASSET' in release $TAG" >&2
  echo "       available assets:" >&2
  printf '%s' "$RELEASE_JSON" | tr ',' '\n' | sed -n 's/.*"name": "\([^"]*\)".*/\1/p' | sort -u >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# 3. Download and verify
# ---------------------------------------------------------------------------

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/lingbox-install.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "==> Downloading $ASSET..."
fetch "$DOWNLOAD_URL" > "$TMP_DIR/$BIN_NAME"
chmod +x "$TMP_DIR/$BIN_NAME"

if [[ -n "${DIGEST:-}" ]]; then
  echo "==> Verifying SHA-256 checksum..."
  if command -v shasum >/dev/null 2>&1; then
    ACTUAL="$(shasum -a 256 "$TMP_DIR/$BIN_NAME" | awk '{print $1}')"
  elif command -v sha256sum >/dev/null 2>&1; then
    ACTUAL="$(sha256sum "$TMP_DIR/$BIN_NAME" | awk '{print $1}')"
  else
    ACTUAL=""
  fi
  if [[ -n "$ACTUAL" && "$ACTUAL" != "$DIGEST" ]]; then
    echo "error: checksum mismatch" >&2
    echo "  expected: $DIGEST" >&2
    echo "  actual:   $ACTUAL" >&2
    exit 1
  fi
  echo "==> Checksum OK"
fi

# ---------------------------------------------------------------------------
# 4. Install
# ---------------------------------------------------------------------------

resolve_install_dir() {
  if [[ -n "${LINGBOX_INSTALL_DIR:-}" ]]; then
    if mkdir -p "$LINGBOX_INSTALL_DIR" && [[ -w "$LINGBOX_INSTALL_DIR" ]]; then
      INSTALL_DIR="$LINGBOX_INSTALL_DIR"
      return
    fi
    echo "error: install dir '$LINGBOX_INSTALL_DIR' is not writable" >&2
    exit 1
  fi
  # XDG conventions first, then the usual user-bin dirs (same precedence as
  # the cargo-dist installers, e.g. uv).
  local candidates=()
  if [[ -n "${XDG_BIN_HOME:-}" ]]; then
    candidates+=("$XDG_BIN_HOME")
  fi
  if [[ -n "${XDG_DATA_HOME:-}" ]]; then
    candidates+=("${XDG_DATA_HOME%/}/../bin")
  fi
  candidates+=("$HOME/.local/bin" "$HOME/bin")
  for dir in "${candidates[@]}"; do
    if mkdir -p "$dir" 2>/dev/null && [[ -w "$dir" ]]; then
      INSTALL_DIR="$dir"
      return
    fi
  done
  echo "error: no writable install location found" >&2
  echo "       set LINGBOX_INSTALL_DIR (or pass --install-dir) and retry" >&2
  exit 1
}
resolve_install_dir

PREVIOUS=""
if [[ -x "$INSTALL_DIR/$BIN_NAME" ]]; then
  PREVIOUS="$("$INSTALL_DIR/$BIN_NAME" --version 2>/dev/null || echo "unknown")"
fi

install -m 0755 "$TMP_DIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
echo "==> Installed to $INSTALL_DIR/$BIN_NAME"

if [[ -n "$PREVIOUS" ]]; then
  echo "==> Upgraded from $PREVIOUS"
fi

# ---------------------------------------------------------------------------
# 5. Verify and report
# ---------------------------------------------------------------------------

if INSTALLED_VERSION="$("$INSTALL_DIR/$BIN_NAME" --version 2>/dev/null)"; then
  echo "==> Installed lingbox $INSTALLED_VERSION"
fi

if ! command -v "$BIN_NAME" >/dev/null 2>&1; then
  echo ""
  echo "  Note: $INSTALL_DIR is not on your PATH. Add it with:"
  case "$(basename "$SHELL" 2>/dev/null)" in
    fish) echo "      fish_add_path $INSTALL_DIR" ;;
    *) echo "      export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
  esac
elif [[ "$(command -v "$BIN_NAME")" != "$INSTALL_DIR/$BIN_NAME" ]]; then
  echo ""
  echo "  Note: $(command -v "$BIN_NAME") exists on your PATH and shadows the new install."
fi
