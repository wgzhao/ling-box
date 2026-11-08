# ling-box for Alfred

An Alfred Script Filter that exposes the lingbox text tools through the `lb`
keyword. Alfred owns the hotkey, matching, ranking and clipboard handling;
lingbox only supplies the data.

Requires [Alfred](https://www.alfredapp.com/) with the Powerpack (Script
Filters and File Actions are Powerpack features) and a `lingbox` binary with
launcher support (v0.10.0 or later; it provides `launcher query`, `file` and
`action`):

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

### Finding the lingbox binary

Alfred does **not** source your login shell profile, so its `PATH` usually
does not include Homebrew (`/opt/homebrew/bin`, `/usr/local/bin`) or
`go install` (`~/go/bin`) directories. The Script Filter therefore looks in
this order:

1. `LINGBOX_BIN`, if set in the workflow's environment variables;
2. `bin/lingbox` inside the workflow bundle;
3. `/opt/homebrew/bin`, `/usr/local/bin`, `/opt/local/bin`,
   `~/.local/bin`, `~/go/bin`, `~/bin`;
4. `lingbox` on `PATH`.

If it still cannot find the binary, set `LINGBOX_BIN` to its absolute path.
The binary must be new enough to support `lingbox launcher alfred`; an older
install produces a "ling-box could not run" item instead of silence.

To bundle the binary and remove the PATH question entirely (build it on the
target architecture):

```sh
go build -o contrib/alfred/bin/lingbox .
sh contrib/alfred/build.sh
open contrib/alfred/dist/lingbox.alfredworkflow
```

`contrib/alfred/bin/` is gitignored. The download from the releases page is
not bundled; it expects the CLI to be installed.

Alfred runs Script Filter scripts with the workflow directory as the working
directory, which is why `info.plist` can call `./script-filter.sh`.

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
| `lb ssl example.com` | TLS scan (runs in the background, refreshes automatically) |
| `lb geo 8.8.8.8` | IP geolocation (first run may download qqwry.dat) |

Select a result and press Enter to copy it. Suggestions and errors are
non-selectable so a stray Enter cannot copy the wrong thing.

## File Actions

Select one or more files in Finder, open Alfred's File Actions, type `lb`
and choose the ling-box action:

- `.dbf`: Show info (copies it), Export JSON, Export CSV. Exports are written
  next to the source and revealed in Finder.
- `.json` / `.yaml` / `.csv` / `.md`: Convert to another format, written next
  to the source.
- images: View in Terminal (opens Terminal with
  `lingbox imgcat -r halfblock`).

Set `LINGBOX_NO_GUI=1` in the workflow environment to keep actions headless:
`dbf info` is returned as text, and file actions print their result instead
of copying or opening anything. This is also what the E2E suite uses.

## Scope

Text tools, File Actions and the async `ssl` / `ipgeo` tools share the same
launcher registry. The registry lives in `internal/launcher`, so adding a
tool is a data change, not a new adapter. A uTools adapter reuses
`lingbox launcher query` in [`contrib/utools`](../utools/README.md).

## Limitations

- `date` arguments are split on whitespace, so pass date-only values
  (`date diff 2026-01-01 2026-07-26`); a datetime containing a space is not
  parsed yet.
- File Actions are Alfred-only for now. The uTools plugin is text-tool only;
  a uTools "files" feature can reuse `lingbox launcher file/action` later.
- `imgcat` opens Terminal with the portable half-block renderer. Users of
  iTerm2 can edit the Run Script to drop `-r halfblock`.

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
