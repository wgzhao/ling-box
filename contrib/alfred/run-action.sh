#!/bin/sh
# Alfred Run Script: execute one file action. Its stdout becomes the text of
# the Post Notification object.

set -eu

WORKFLOW_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$WORKFLOW_DIR/lib.sh"

if lingbox_override_invalid; then
	printf 'LINGBOX_BIN is not executable\n'
	exit 0
fi
if ! lb=$(find_lingbox); then
	printf 'ling-box not found. Install with: brew install wgzhao/tap/lingbox\n'
	exit 0
fi
if [ "$#" -eq 0 ]; then
	printf 'No action selected\n'
	exit 0
fi

exec "$lb" launcher action --spec "$1"
