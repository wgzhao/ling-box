#!/bin/sh
# Alfred Script Filter for ling-box.
#
# Alfred calls this with the query, either as one argument or split into
# several; joining with "$*" handles both:
#   /bin/sh script-filter.sh "<query>"
#
# The binary is looked up in this order:
#   1. $LINGBOX_BIN (custom installs and tests)
#   2. ./bin/lingbox inside the workflow bundle
#   3. lingbox on PATH
#
# It must always print one JSON document: Alfred shows nothing otherwise.

set -eu

query="$*"
workflow_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

not_installed() {
	printf '%s\n' '{"items":[{"title":"ling-box is not installed","subtitle":"Install with: brew install wgzhao/tap/lingbox","valid":false}]}'
	exit 0
}

if [ -n "${LINGBOX_BIN:-}" ]; then
	if [ -x "$LINGBOX_BIN" ]; then
		lb=$LINGBOX_BIN
	else
		not_installed
	fi
elif [ -x "$workflow_dir/bin/lingbox" ]; then
	lb=$workflow_dir/bin/lingbox
elif command -v lingbox >/dev/null 2>&1; then
	lb=$(command -v lingbox)
else
	not_installed
fi

exec "$lb" launcher alfred --query="$query"
