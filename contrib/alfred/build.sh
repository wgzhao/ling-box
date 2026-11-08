#!/bin/sh
# Build the Alfred workflow package: dist/lingbox.alfredworkflow
#
# An .alfredworkflow file is a zip whose root contains info.plist. Alfred
# installs it when the user double-clicks it.
set -eu

dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
out=$dir/dist
stage=$(mktemp -d)
trap 'rm -rf "$stage"' 0 1 2 15

mkdir -p "$out"
rm -f "$out/lingbox.alfredworkflow"

cp "$dir/info.plist" "$stage/"
cp "$dir/lib.sh" "$stage/"
cp "$dir/script-filter.sh" "$stage/"
cp "$dir/file-filter.sh" "$stage/"
cp "$dir/run-action.sh" "$stage/"
chmod +x "$stage/script-filter.sh" "$stage/file-filter.sh" "$stage/run-action.sh"
if [ -f "$dir/icon.png" ]; then
	cp "$dir/icon.png" "$stage/"
fi
# An optional bin/lingbox makes the workflow independent of PATH. It is
# gitignored; build it for the target architecture before packaging.
if [ -d "$dir/bin" ]; then
	cp -R "$dir/bin" "$stage/"
fi

# List files explicitly so the archive has no "./" prefixes, which Alfred
# does not expect.
files="info.plist lib.sh script-filter.sh file-filter.sh run-action.sh"
if [ -f "$dir/icon.png" ]; then
	files="$files icon.png"
fi
if [ -d "$dir/bin" ]; then
	files="$files bin"
fi

# shellcheck disable=SC2086
( cd "$stage" && zip -q -X -r "$out/lingbox.alfredworkflow" $files )

echo "built $out/lingbox.alfredworkflow"
