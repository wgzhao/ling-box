#!/bin/sh
# Alfred File Action Script Filter: list the ling-box actions for the
# selected files. Alfred passes the paths as argv.

set -eu

WORKFLOW_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$WORKFLOW_DIR/lib.sh"

if lingbox_override_invalid; then
	json_error "LINGBOX_BIN is not executable" "Fix the workflow environment variable or unset it"
fi
if ! lb=$(find_lingbox); then
	json_error "ling-box not found" "Install: brew install wgzhao/tap/lingbox — or set LINGBOX_BIN"
fi
if [ "$#" -eq 0 ]; then
	json_error "No file selected" "Select a file in Finder first"
fi

if out=$("$lb" launcher file -- "$@" 2>/dev/null) && [ -n "$out" ]; then
	printf '%s\n' "$out"
else
	json_error "ling-box could not run" "Upgrade lingbox or set LINGBOX_BIN"
fi
