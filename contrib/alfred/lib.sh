#!/bin/sh
# Shared helpers for the ling-box Alfred workflow scripts.
# The caller must set WORKFLOW_DIR before sourcing this file.

lingbox_override_invalid() {
	[ -n "${LINGBOX_BIN:-}" ] && [ ! -x "$LINGBOX_BIN" ]
}

find_lingbox() {
	if [ -n "${LINGBOX_BIN:-}" ]; then
		if [ -x "$LINGBOX_BIN" ]; then
			printf '%s\n' "$LINGBOX_BIN"
			return 0
		fi
		return 1
	fi
	# Alfred does not source the login shell profile, so Homebrew and go
	# install directories are often missing from PATH.
	for candidate in \
		"$WORKFLOW_DIR/bin/lingbox" \
		/opt/homebrew/bin/lingbox \
		/usr/local/bin/lingbox \
		/opt/local/bin/lingbox \
		"$HOME/.local/bin/lingbox" \
		"$HOME/go/bin/lingbox" \
		"$HOME/bin/lingbox"
	do
		if [ -x "$candidate" ]; then
			printf '%s\n' "$candidate"
			return 0
		fi
	done
	if command -v lingbox >/dev/null 2>&1; then
		command -v lingbox
		return 0
	fi
	return 1
}

json_error() {
	printf '{"items":[{"title":"%s","subtitle":"%s","valid":false}]}\n' "$1" "$2"
	exit 0
}
