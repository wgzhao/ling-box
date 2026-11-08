#!/bin/sh
# Alfred Script Filter for ling-box.
#
# Alfred runs Script Filter scripts with the workflow directory as the
# working directory, so "./script-filter.sh" in info.plist resolves to this
# file. "$0" is used only to locate an optional bundled bin/lingbox.
#
# Alfred does not source the user's login shell profile, so Homebrew and
# "go install" directories are often missing from PATH. The binary is looked
# up in this order:
#   1. $LINGBOX_BIN (custom installs and tests)
#   2. ./bin/lingbox inside the workflow bundle
#   3. common install locations (Homebrew, ~/.local/bin, ~/go/bin, ~/bin)
#   4. lingbox on PATH
#
# It must always print one JSON document: Alfred shows nothing otherwise.

set -eu

# Alfred may pass the query as one argument or split into several; "$*"
# handles both.
query="$*"
workflow_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

json_error() {
	printf '{"items":[{"title":"%s","subtitle":"%s","valid":false}]}\n' "$1" "$2"
	exit 0
}

# Run the launcher and always print JSON, including when the binary found is
# too old to know the "launcher" command.
run_launcher() {
	lb=$1
	if out=$("$lb" launcher alfred --query="$query" 2>/dev/null) && [ -n "$out" ]; then
		printf '%s\n' "$out"
		return 0
	fi
	json_error "ling-box could not run" "Run: lingbox tools --json (upgrade an old install) or set LINGBOX_BIN"
}

if [ -n "${LINGBOX_BIN:-}" ]; then
	if [ -x "$LINGBOX_BIN" ]; then
		run_launcher "$LINGBOX_BIN"
		exit 0
	fi
	json_error "LINGBOX_BIN is not executable" "Fix the workflow environment variable or unset it"
fi

# Prefer a bundled binary, then the usual install locations. Alfred does not
# inherit the interactive shell's PATH, which is why "command -v lingbox"
# alone is not enough.
for candidate in \
	"$workflow_dir/bin/lingbox" \
	/opt/homebrew/bin/lingbox \
	/usr/local/bin/lingbox \
	/opt/local/bin/lingbox \
	"$HOME/.local/bin/lingbox" \
	"$HOME/go/bin/lingbox" \
	"$HOME/bin/lingbox"
do
	if [ -x "$candidate" ]; then
		run_launcher "$candidate"
		exit 0
	fi
done

if command -v lingbox >/dev/null 2>&1; then
	run_launcher "$(command -v lingbox)"
	exit 0
fi

json_error "ling-box not found" "Install: brew install wgzhao/tap/lingbox — or set LINGBOX_BIN"
