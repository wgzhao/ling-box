#!/usr/bin/env bash
#
# Rewrite the Homebrew tap formula so it points at a freshly built release.
#
# Usage: update-formula.sh <tag> <assets-dir> <formula-file>
#
#   <tag>           release tag, e.g. v0.8.2. Must be the git tag itself, since
#                   the asset URLs are .../releases/download/<tag>/<asset>.
#   <assets-dir>    directory holding the built binaries, named exactly as the
#                   release assets are (lingbox-macos-arm64, ...).
#   <formula-file>  the tap formula to rewrite, in place.
#
# Only the tag inside the download URLs and the per-platform sha256 values are
# touched. desc, comments and the install method are left alone, so anything
# handwritten in the formula survives a bump.

set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "usage: $0 <tag> <assets-dir> <formula-file>" >&2
  exit 2
fi

TAG="$1"
ASSETS_DIR="$2"
FORMULA="$3"

case "$TAG" in
  v*) ;;
  *)
    echo "error: tag '$TAG' must start with 'v'; assets are published under the tag name" >&2
    exit 1
    ;;
esac

[ -f "$FORMULA" ] || { echo "error: no such formula: $FORMULA" >&2; exit 1; }

# GNU coreutils in CI, BSD shasum on a developer's Mac.
if command -v sha256sum >/dev/null 2>&1; then
  hash_file() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null 2>&1; then
  hash_file() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
  echo "error: need either sha256sum or shasum on PATH" >&2
  exit 1
fi

# Platforms the formula installs. Keep in sync with the url blocks in the formula.
ASSETS="lingbox-macos-arm64
lingbox-macos-x86-64
lingbox-linux-arm64
lingbox-linux-x86-64"

HASHES="$(mktemp)"
trap 'rm -f "$HASHES"' EXIT

for asset in $ASSETS; do
  src="$ASSETS_DIR/$asset"
  [ -f "$src" ] || { echo "error: missing asset: $src" >&2; exit 1; }
  printf '%s %s\n' "$asset" "$(hash_file "$src")" >> "$HASHES"
done

# 1. Repoint the download URLs at the new tag. Written to a temp file rather
#    than using `sed -i`, whose argument handling differs between GNU and BSD.
sed "s|/releases/download/v[^/]*/|/releases/download/${TAG}/|g" "$FORMULA" > "$FORMULA.tmp"

# 2. Recompute each sha256 from the asset named on the url line directly above
#    it, so the values stay tied to the right platform regardless of block order.
awk -v hashes="$HASHES" '
  BEGIN {
    while ((getline line < hashes) > 0) {
      split(line, field, " ")
      h[field[1]] = field[2]
    }
  }
  /^[[:space:]]*url / {
    print
    if (getline next_line) {
      if (next_line ~ /^[[:space:]]*sha256 "/) {
        asset = ""
        if (match($0, /[^\/]+"$/)) asset = substr($0, RSTART, RLENGTH - 1)
        if (asset in h) {
          sub(/sha256 "[^"]*"/, "sha256 \"" h[asset] "\"", next_line)
        } else {
          printf "warning: no built asset matches url for %s\n", asset > "/dev/stderr"
        }
      }
      print next_line
    }
    next
  }
  { print }
' "$FORMULA.tmp" > "$FORMULA.tmp2"

mv "$FORMULA.tmp2" "$FORMULA"
rm -f "$FORMULA.tmp"

echo "updated $FORMULA to $TAG"
