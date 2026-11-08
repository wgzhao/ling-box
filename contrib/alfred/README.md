# ling-box for Alfred

An Alfred Script Filter that exposes the lingbox text tools through the `lb`
keyword. Alfred owns the hotkey, matching, ranking and clipboard handling;
lingbox only supplies the data.

Requires [Alfred](https://www.alfredapp.com/) with the Powerpack (Script
Filters are a Powerpack feature) and the `lingbox` binary:

```sh
brew install wgzhao/tap/lingbox
```

## Install

Download `lingbox.alfredworkflow` from the
[releases page](https://github.com/wgzhao/ling-box/releases) and double-click
it. Alternatively, build it from this directory:

```sh
sh build.sh
open dist/lingbox.alfredworkflow
```

The workflow finds `lingbox` on `PATH`; if it is somewhere else, set
`LINGBOX_BIN` in the workflow's environment variables to its absolute path.
The workflow also looks for `bin/lingbox` inside the bundle, so a bundled
build is possible without changing the script.

## Use

| Query | Result |
| --- | --- |
| `lb` | List the supported tools |
| `lb uuid` | Generate a UUID (v4) |
| `lb password` | Generate a 16-character password |
| `lb url hello world` | `hello+world` |
| `lb b64 Hello World` | `SGVsbG8gV29ybGQ=` |
| `lb uni 你好` | `你好` |
| `lb ip 10.0.0.0/8` | Subnet breakdown |
| `lb ip 10.0.0.1 255.255.0.0` | Subnet breakdown from address + mask |
| `lb color #FF0000` | Hex, RGB and HSL |
| `lb plate 湘` | Hunan plate regions |
| `lb date add 2026-01-01 10` | `2026-01-11` |
| `lb json {"a":1}` | Formatted JSON |

Select a result and press Enter to copy it. Suggestions and errors are
non-selectable so a stray Enter cannot copy the wrong thing.

## Scope

This first version deliberately covers only fast, text-in / text-out tools.
File-based tools (`dbf`, `convert`, `imgcat`) belong in Alfred File Actions
and long-running tools (`ssl`, `ipgeo`) need asynchronous handling; both are
follow-up work. The registry lives in `internal/launcher`, so adding a tool is
a data change, not a new adapter.

## Limitations

- `date` arguments are split on whitespace, so pass date-only values
  (`date diff 2026-01-01 2026-07-26`); a datetime containing a space is not
  parsed yet.
- Only text-in / text-out tools are exposed. File pickers and long-running
  tools are deliberate follow-ups.

## Development

The Alfred adapter is just a JSON formatter over the CLI:

```sh
lingbox tools --json
lingbox ipcalc --json -- 10.0.0.0/8
lingbox launcher alfred --query "ip 10.0.0.0/8"
```

Test the Script Filter without Alfred:

```sh
LINGBOX_BIN=$(command -v lingbox) sh script-filter.sh "uuid"
```

The end-to-end suite in `test/e2e` builds the real binary, drives the filter
with a set of queries, and writes the outputs to `test/e2e/artifacts`.
