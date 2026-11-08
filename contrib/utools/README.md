# ling-box for uTools

A [uTools](https://u.tools/) plugin that exposes the lingbox text tools
through the `lb` keyword. It uses the same launcher JSON contract as the
Alfred workflow, so both adapters share one implementation.

Requires a `lingbox` binary that includes `lingbox launcher query`
(v0.10.0 or later) and uTools 2.x or later.

## Install

For development:

1. Enable uTools developer mode (设置 → 开发者).
2. Create a plugin project pointing at this directory, or import
   `dist/ling-box.upx` after building it:

```sh
sh build.sh
```

3. Type `lb` in uTools to open the plugin.

Publishing to the uTools plugin store requires a uTools developer account
and the store's review process; the source and build script live here.

## Usage

Type `lb`, press Enter, then query:

| Query | Result |
| --- | --- |
| `uuid` | Generate a UUID (v4) |
| `url hello world` | `hello+world` |
| `b64 Hello World` | `SGVsbG8gV29ybGQ=` |
| `ip 10.0.0.0/8` | Subnet breakdown |
| `color #FF0000` | Hex, RGB and HSL |
| `plate 湘` | Hunan plate regions |
| `date add 2026-01-01 10` | `2026-01-11` |
| `json {"a":1}` | Formatted JSON |
| `ssl example.com` | TLS scan (runs in the background, polls automatically) |
| `geo 8.8.8.8` | IP geolocation (first run may download qqwry.dat) |

Enter copies the result and closes the plugin. Tab/Enter on a suggestion
autocompletes the tool prefix.

## Finding lingbox

uTools does not always inherit the login shell's `PATH`. The preload checks:

1. `LINGBOX_BIN`, if set;
2. macOS Homebrew/MacPorts paths, Windows Scoop/Chocolatey paths,
   `~/.local/bin`, `~/go/bin`, `~/bin`;
3. `lingbox` on `PATH`.

Set the `LINGBOX_BIN` environment variable for uTools if the binary lives
somewhere else.

## Test without uTools

The adapter's lookup and query path can be exercised from Node:

```sh
node test.js "$(command -v lingbox)" uuid
node test.js "$(command -v lingbox)" "ip 10.0.0.0/8"
```

The Go E2E suite also runs this script and stores the output in
`test/e2e/artifacts`.

## Scope

Text tools are supported. File actions (DBF/convert/imgcat) are currently
Alfred-only; a uTools "files" feature can reuse `lingbox launcher file` and
`lingbox launcher action` in a follow-up.
