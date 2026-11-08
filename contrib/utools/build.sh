#!/bin/sh
# Build the uTools plugin package: dist/ling-box.upx
#
# A .upx file is a zip with plugin.json at its root. uTools' developer tool
# can also package the directory directly.
set -eu

dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
out=$dir/dist
stage=$(mktemp -d)
trap 'rm -rf "$stage"' 0 1 2 15

mkdir -p "$out"
rm -f "$out/ling-box.upx"

cp "$dir/plugin.json" "$dir/preload.js" "$dir/index.html" "$dir/logo.png" "$stage/"

( cd "$stage" && zip -q -X -r "$out/ling-box.upx" plugin.json preload.js index.html logo.png )

echo "built $out/ling-box.upx"
