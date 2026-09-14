#!/usr/bin/env bash
#
# Rewrite the tap formula for a release and push it to the tap branch.
#
# Usage: push-formula.sh <tag> <assets-dir> <tap-dir> <branch>
#
#   <tap-dir>  an existing clone of the tap whose `origin` can be pushed to.
#   <branch>   the branch to push to, e.g. main for a release, or a scratch
#              branch from the dry-run workflow.
#
# Exits 0 without committing when the formula is already at <tag>, so a
# re-run of a release is a no-op rather than an empty commit.

set -euo pipefail

if [ "$#" -ne 4 ]; then
  echo "usage: $0 <tag> <assets-dir> <tap-dir> <branch>" >&2
  exit 2
fi

TAG="$1"
ASSETS_DIR="$2"
TAP_DIR="$3"
BRANCH="$4"

FORMULA="$TAP_DIR/Formula/lingbox.rb"
[ -f "$FORMULA" ] || { echo "error: no such formula: $FORMULA" >&2; exit 1; }

"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/update-formula.sh" "$TAG" "$ASSETS_DIR" "$FORMULA"

if git -C "$TAP_DIR" diff --quiet; then
  echo "Formula is already at ${TAG}; nothing to push."
  exit 0
fi

git -C "$TAP_DIR" config user.name "github-actions[bot]"
git -C "$TAP_DIR" config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git -C "$TAP_DIR" commit -am "lingbox: bump to ${TAG}"
git -C "$TAP_DIR" push origin "HEAD:${BRANCH}"

echo "pushed ${TAG} to ${BRANCH}:"
git -C "$TAP_DIR" --no-pager show --stat HEAD
