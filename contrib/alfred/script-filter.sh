#!/bin/sh
# Alfred Script Filter for ling-box.
#
# Alfred runs Script Filter scripts with the workflow directory as the
# working directory, so "./script-filter.sh" in info.plist resolves to this
# file. "$0" is used only to locate the shared helpers and an optional
# bundled bin/lingbox.
#
# Alfred may pass the query as one argument or split into several; "$*"
# handles both. It must always print one JSON document.

set -eu

query="$*"
WORKFLOW_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$WORKFLOW_DIR/lib.sh"

if lingbox_override_invalid; then
	json_error "LINGBOX_BIN is not executable" "Fix the workflow environment variable or unset it"
fi
if ! lb=$(find_lingbox); then
	json_error "ling-box not found" "Install: brew install wgzhao/tap/lingbox — or set LINGBOX_BIN"
fi

if out=$("$lb" launcher alfred --query="$query" 2>/dev/null) && [ -n "$out" ]; then
	printf '%s\n' "$out"
else
	json_error "ling-box could not run" "Run: lingbox tools --json (upgrade an old install) or set LINGBOX_BIN"
fi
