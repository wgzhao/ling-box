# lingbox

玲珑盒，一款工具集 - A collection of useful utility tools for developers.

> [中文文档](README_zh.md)

## Demo

<img src="docs/demo.gif" width="700" alt="lingbox demo: IP geolocation, subnet calculation, UUIDs, date math, and a QR code rendered in the terminal">

The demo is recorded with asciinema and rendered to GIF; regenerate it with
[docs/record_demo.py](docs/record_demo.py) and [docs/cast2gif.py](docs/cast2gif.py).

## Installation

### Homebrew (macOS / Linux)

```bash
brew tap wgzhao/tap
brew install lingbox
```

### Shell script (macOS / Linux)

Downloads the latest precompiled binary for your OS and architecture (no Go required):

```bash
curl -fsSL https://raw.githubusercontent.com/wgzhao/ling-box/master/install.sh | bash
```

Customize with `--install-dir <dir>` (or `LINGBOX_INSTALL_DIR`), or pin a
version with `LINGBOX_VERSION` (e.g. `LINGBOX_VERSION=0.5.0`). The binary is
SHA-256 verified against the release digest.

### From source

Requires [Go](https://go.dev/dl/) 1.26 or higher. See [Building](#building) below.

## Features

- **URL Encoding/Decoding**: Encode and decode URL strings
- **Base64 Encoding/Decoding**: Encode and decode Base64 strings (supports URL-safe mode)
- **BCrypt Password Hashing**: Generate and verify bcrypt password hashes
- **QR Code Generation**: Generate QR codes as PNG, JPG, or GIF images
- **Password Generation**: Generate secure random passwords with customizable options
- **UUID Generation**: Generate UUIDs (v1, v4, v7) in bulk
- **Format Conversion**: Convert between JSON, YAML, CSV, and Markdown (pandoc-style `-i`/`-o`/`-t`)
- **JSON Formatting & Validation**: Re-indent JSON and validate its syntax (`json format`, `json verify`)
- **Unicode Encoding/Decoding**: Encode text to \uXXXX escapes and decode back
- **Color Conversion**: Convert between Hex, RGB, HSL, and named colors
- **BMI Calculator**: Calculate Body Mass Index from height and weight
- **Number Base Conversion**: Convert between binary, octal, decimal, and hexadecimal
- **Date Calculator**: Add/subtract days or calculate date differences
- **Terminal Image Display**: Display images directly in the terminal (iTerm2/Kitty/half-block/ASCII)
- **Terminal PDF Browsing**: Render and browse PDF files in the terminal
- **SSL Certificate Inspection**: Inspect X.509 certificates (subject, issuer, key strength, validity, extensions)
- **SSL Host Scanning**: Scan a host's TLS protocols, cipher suites with security ratings, and certificate trust
- **License Plate Lookup**: Query Chinese license plate codes by province (`plate`)
- **IPv4 Subnet Calculator**: Calculate networks, broadcast, wildcard masks, and host ranges; sub-/supernets, deaggregation, and splitting (`ipcalc`)
- **Temporary Web Server**: Serve a directory over HTTP with python `http.server`-style listings and logs (`webserver`)
- **DBF File Reading**: Read dBase/FoxPro tables (`dbf info|view|export`), with GBK/GB18030/Big5 decoding and .fpt/.dbt memo support
- **IP Geolocation**: Look up the region and ISP of IPv4 addresses from the qqwry.dat database (`ipgeo`); the database is downloaded on first use and cached

## Requirements

- Go 1.26 or higher

## Building

```bash
go build -ldflags "-s -w" -trimpath -o lingbox .
```

## Running

```bash
./lingbox <command> [options]
```

## Usage

### General Help

```bash
./lingbox --help
```

### URL Encoding/Decoding

```bash
# Encode a URL string
./lingbox url -e 'hello world'
# Output: hello+world

# Decode a URL string
./lingbox url -d 'hello+world'
# Output: hello world
```

### Base64 Encoding/Decoding

```bash
# Encode a string to Base64
./lingbox base64 -e 'Hello World'
# Output: SGVsbG8gV29ybGQ=

# Decode a Base64 string
./lingbox base64 -d 'SGVsbG8gV29ybGQ='
# Output: Hello World

# Use URL-safe Base64 (flag placement matters: -u before -e)
./lingbox base64 -u -e 'test+/string'
```

### BCrypt Password Hashing

```bash
# Generate a bcrypt hash
./lingbox bcrypt -g mypassword
# Output: $2a$12$...

# Verify a password against a hash
./lingbox bcrypt -v mypassword '$2a$12$...'
```

### QR Code Generation

```bash
# Generate a QR code (default: qrcode.png, 300x300)
./lingbox qrcode 'https://example.com'

# Custom output file and size
./lingbox qrcode 'Hello World' -o mycode.png -s 500

# Different format
./lingbox qrcode 'Test' -o mycode.jpg -f JPG
```

### Password Generation

```bash
# Generate a 16-character password (default)
./lingbox password

# Generate a 24-character password
./lingbox password -l 24

# Generate multiple passwords
./lingbox password -c 5

# Generate digits-only password
./lingbox password -d

# Generate uppercase-only password
./lingbox password -u

# Generate password without special characters
./lingbox password -n
```

### UUID Generation

```bash
# Generate a v4 UUID (default)
./lingbox uuid

# Generate 5 UUIDs
./lingbox uuid -n 5

# Generate UUID v7 (time-ordered, sortable)
./lingbox uuid -t v7

# Generate UUID v1 (time-based)
./lingbox uuid -t v1
```

### Format Conversion (convert)

Convert between JSON, YAML, CSV, and Markdown with a pandoc-style `-i`/`-o`/`-t`
interface. The input format is read from the `-i` file extension (or detected
from stdin); the target format comes from `-t`, or is guessed from the `-o`
extension when `-t` is omitted.

Input formats: `json`, `yaml`/`yml`, `csv`. Output formats: `json`, `yaml`,
`csv`, `markdown`.

CSV output requires an array of objects (first row = header) or an array of
scalar values. Markdown output also accepts a scalar array (bullet list) and a
top-level object (key/value table). Nested objects and arrays inside cells are
encoded as compact JSON strings. CSV input uses the first row as the header;
cells stay strings (no type inference).

```bash
# Convert JSON to YAML (target format guessed from -o extension)
./lingbox convert -i data.json -o data.yaml

# CSV to Markdown
./lingbox convert -i data.csv -t markdown

# JSON to CSV
./lingbox convert -i data.json -o data.csv

# From stdin (format auto-detected: JSON, then YAML, then CSV)
cat data.json | ./lingbox convert -t yaml
curl -s https://api.example.com/data | ./lingbox convert -o out.yaml

# Markdown table output
./lingbox convert -i data.yaml -o table.md
```

### JSON Tools (json)

```bash
# Format (re-indent) JSON; key order is preserved
./lingbox json format data.json
cat data.json | ./lingbox json format --indent 4

# Sort keys and/or collapse to a single line
cat data.json | ./lingbox json format --sort-keys --compact

# Validate JSON syntax (exits 1 on failure with line/column)
./lingbox json verify data.json
cat data.json | ./lingbox json verify -q   # silent on success
```

### Unicode Encoding/Decoding

```bash
# Encode Chinese characters to \uXXXX
./lingbox unicode -e '你好世界'
# Output: 你好世界

# Decode \uXXXX back to text
./lingbox unicode -d '你好世界'
# Output: 你好世界

# Auto-detect mode (encodes if plain text, decodes if \uXXXX)
./lingbox unicode '你好'
```

### Color Conversion

```bash
# Named color
./lingbox color 'red'
# Output: Hex, RGB, HSL, and name

# Hex input
./lingbox color '#FF0000'

# RGB input
./lingbox color 'rgb(255, 0, 0)'

# HSL input
./lingbox color 'hsl(0, 100%, 50%)'

# Multi-word named color
./lingbox color 'dark gray'
./lingbox color 'light yellow'
```

### Terminal Image Display (imgcat)

Display images directly in the terminal. Default renderer is OSC 1337 (iTerm2
protocol), which produces lossless output and is supported by iTerm2, WezTerm,
Warp, kaku, kitty (compat mode), VS Code terminal, and many others.

```bash
# Display an image (auto-detect best renderer)
./lingbox imgcat photo.jpg

# Specify output width in character columns
./lingbox imgcat photo.png -w 60

# Force a specific renderer
./lingbox imgcat photo.jpg -r halfblock   # ANSI ▀ blocks (universal)
./lingbox imgcat photo.jpg -r iterm2      # OSC 1337 lossless (default)
./lingbox imgcat photo.jpg -r kitty       # Kitty native protocol
./lingbox imgcat photo.jpg -r ascii       # Grayscale ASCII art

# Read from stdin
cat photo.png | ./lingbox imgcat

# Browse multiple images interactively (arrows/space for next, q to quit)
./lingbox imgcat photo1.jpg photo2.jpg photo3.png

# Glob patterns are expanded automatically (useful in quoted arguments)
./lingbox imgcat "875*.png"
./lingbox imgcat photo1.jpg "screenshot-*.png"

```

### Terminal PDF Browsing (pdf)

Render and browse PDF files directly in the terminal using the same renderers as imgcat.

```bash
# Interactive page browsing (arrow keys flip pages, q quits)
./lingbox pdf document.pdf

# Display a specific page
./lingbox pdf -p 1 document.pdf

# Use a specific renderer and width
./lingbox pdf -r ascii -w 80 document.pdf
./lingbox pdf -r halfblock document.pdf

# Higher DPI for better quality
./lingbox pdf --dpi 300 document.pdf

# Read PDF from stdin (renders page 1)
cat document.pdf | ./lingbox pdf
```

### SSL Certificate Tools (ssl)

#### Inspecting a certificate file (ssl cert)

Parse and display detailed information about X.509 certificates: subject and
issuer, signature algorithm, public key and key strength, validity period
(including expired status), and extensions (SAN, key usages, SKI/AKI, OCSP,
CRL, policies).

```bash
# From a file (PEM or DER .crt)
./lingbox ssl cert server.crt
./lingbox ssl cert -f fullchain.pem

# Inline PEM content (single-line with \n escapes also works)
./lingbox ssl cert '-----BEGIN CERTIFICATE-----...'

# From stdin
cat cert.pem | ./lingbox ssl cert
```

#### Scanning a remote host (ssl host)

Connect to a host and report the supported TLS protocol versions, the cipher
suites accepted per version (with INSECURE/WEAK ratings in red/yellow), and
the presented certificate's validity, trust status, and hostname match.

```bash
# Default port 443
./lingbox ssl host www.baidu.com

# Explicit port
./lingbox ssl host https://www.baidu.com:8443
./lingbox ssl host example.com:8443

# Per-handshake timeout (default 5s)
./lingbox ssl host www.baidu.com -t 10

# Quiet mode: no progress display (for batch/scripted use)
./lingbox ssl host www.baidu.com -q
```

Progress is shown by default on a terminal and written to stderr, so a
piped/redirected stdout always contains only the report. It is suppressed
automatically when stderr is not a terminal.

Notes: SSL 3.0 cannot be probed (Go's TLS client does not implement it), and
TLS 1.3 cipher suites are not individually negotiable, so they are reported
as a group.

### License Plate Lookup (plate)

Query Chinese license plate codes by province or municipality. The argument
accepts the one-character abbreviation (湘), the full name (湖南省), or the
name without its administrative suffix (湖南).

```bash
# Query one province
./lingbox plate 湘
./lingbox plate 湖南省
./lingbox plate 湖南

# List all 31 provinces, paged (Enter: next page, q: quit)
./lingbox plate

# More provinces per page (interactive mode)
./lingbox plate -n 10

# Non-interactive: piped or redirected output prints everything at once
./lingbox plate | grep 湘
```

### IPv4 Subnet Calculator (ipcalc)

Port of [Krischan Jodies' ipcalc](http://jodies.de/ipcalc) (v0.41). Takes an
IP address and netmask and calculates the resulting broadcast, network, Cisco
wildcard mask, and host range, presenting the results as easy-to-understand
binary values. By giving a second netmask you can design sub- and supernetworks.

```bash
# Basic calculation (netmask defaults to /24 when omitted)
./lingbox ipcalc 192.168.0.1/24

# Dotted or wildcard (inverse) netmasks are recognized
./lingbox ipcalc 192.168.0.1/255.255.128.0
./lingbox ipcalc 192.168.0.1 0.0.63.255

# Subnets after transition (second netmask > first)
./lingbox ipcalc 192.168.0.1 255.255.255.0 255.255.255.128

# Supernet (second netmask < first)
./lingbox ipcalc 10.0.0.1 255.255.255.0 255.255.0.0

# Deaggregate an address range into CIDR blocks
./lingbox ipcalc 192.168.1.10 - 192.168.2.5
./lingbox ipcalc 192.168.1.10 192.168.2.5 -r

# Split a network into subnets sized for the given host counts
./lingbox ipcalc 10.0.0.0/24 -s 100 50 25

# Print the natural class bit-count mask only
./lingbox ipcalc -c 192.168.0.1   # -> 24

# No binary display / no colors
./lingbox ipcalc 192.168.0.1/24 -b
./lingbox ipcalc 192.168.0.1/24 -n
```

Colors are enabled automatically on a terminal (respecting `NO_COLOR`) and can
be disabled with `-n`. The original's two stray debug lines ("WILDCARD" and
"INVALID NETMASK") are intentionally not reproduced.

### Temporary Web Server (webserver)

Serve the current directory over HTTP with the same interface as
`python3 -m http.server`: identical startup message, request log format,
directory listings, and error pages.

```bash
# Serve ./ on port 8000 (default)
./lingbox webserver

# Serve a specific directory on a specific port
./lingbox webserver 8080
./lingbox webserver -d ~/public

# Bind to localhost only
./lingbox webserver -b 127.0.0.1

# Port 0 picks a free port (shown in the startup message)
./lingbox webserver 0
```

Serves files with MIME type detection, index.html support, percent-decoded
paths, and directory listings. Path traversal attempts ("..") are blocked.
Request logs go to stderr in python's format; Ctrl-C stops the server
gracefully ("Keyboard interrupt received, exiting.").

### DBF File Reading (dbf)

Read dBase / FoxPro database table files (.dbf), including the common field
types (C/N/F/D/L/M/I/T/Y/B) and memo files (.fpt/.dbt).

```bash
# Show the file header, encoding, and fields
./lingbox dbf info 客户.dbf

# Browse records as an aligned table (CJK-aware)
./lingbox dbf view 客户.dbf -n 10

# Deleted records are hidden by default; show them with a * marker
./lingbox dbf view 客户.dbf --include-deleted

# Force an encoding (auto-detected from the language driver ID,
# defaulting to GBK)
./lingbox dbf view 客户.dbf -e gbk

# Export to CSV or typed JSON
./lingbox dbf export 客户.dbf -o 客户.csv
./lingbox dbf export 客户.dbf --json
```

Text encoding is auto-detected from the language driver ID (0x4D/0x7A = GBK)
and defaults to GBK when unset. Encrypted tables (encryption flag set) are
refused. Microsoft Access (.mdb/.accdb) is a different format and is not
supported.

### IP Geolocation (ipgeo)

Look up the region and ISP of IPv4 addresses using the qqwry.dat database
(纯真 IP 库), mirrored from
[metowolf/qqwry.dat](https://github.com/metowolf/qqwry.dat) (updated weekly).

The ~27 MB database is downloaded on first use into the user cache directory
and reused offline afterwards:

| Platform | Default database path |
|----------|-----------------------|
| macOS | `~/Library/Caches/ling-box/qqwry.dat` |
| Linux | `~/.cache/ling-box/qqwry.dat` (or `$XDG_CACHE_HOME/ling-box`) |
| Windows | `%LocalAppData%\ling-box\qqwry.dat` |

Both the modern CZDB-rebuilt layout and the classic qqwry.dat layout are
supported, so a custom database file can be used with `--db`.

```bash
# Single lookup (downloads the database on first use)
./lingbox ipgeo 8.8.8.8

# Multiple addresses
./lingbox ipgeo 114.114.114.114 223.5.5.5

# JSON output
./lingbox ipgeo --json 8.8.8.8

# Force a database update
./lingbox ipgeo --update 8.8.8.8

# Use a custom database file
./lingbox ipgeo --db /path/to/qqwry.dat 8.8.8.8
```

## Shell Completion

Shell completion is built-in via Cobra. Enable tab completion for commands, flags, and subcommands:

```bash
# Bash
source <(./lingbox completion bash)

# Zsh
source <(./lingbox completion zsh)

# Fish (your shell)
./lingbox completion fish > ~/.config/fish/completions/lingbox.fish

# PowerShell
./lingbox completion powershell | Out-String | Invoke-Expression
```

After enabling, type `./lingbox con<Tab>` to auto-complete to `convert`.

## Cross-Platform Support

This tool is written in Go and compiles to a single static binary for any platform:
- Windows
- macOS
- Linux

## Release Process

To create a new release, tag the commit and push:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This will automatically trigger GitHub Actions to:
1. Build native binaries for Linux (x86-64, ARM64), Windows (x86-64), and macOS (x86-64, ARM64)
2. Generate release notes from commits between tags
3. Create a GitHub release with all binaries attached

## Quick Reference

| Command | Description | Example |
|---------|-------------|---------|
| `url` | URL encode/decode | `url -e 'hello world'` |
| `base64` | Base64 encode/decode | `base64 -e 'Hello'` |
| `bcrypt` | Password hashing | `bcrypt -g mypass` |
| `qrcode` | QR code generation | `qrcode 'text' -o qr.png` |
| `password` | Password generator | `password -l 24 -c 5` |
| `uuid` | UUID generation | `uuid -n 5 -t v7` |
| `convert` | Format conversion (JSON/YAML/CSV/Markdown) | `convert -i data.json -o data.yaml` |
| `json format` | Re-indent JSON | `json format data.json` |
| `json verify` | Validate JSON syntax | `json verify data.json` |
| `unicode` | Unicode encoder/decoder | `unicode -e '你好'` |
| `color` | Color converter | `color '#FF0000'` |
| `bmi` | BMI calculator | `bmi 170 65` |
| `base` | Base converter | `base FF -f hex` |
| `date` | Date calculator | `date diff 2026-01-01 2026-07-26` |
| `imgcat` | Terminal image display | `imgcat photo.jpg` |
| `pdf` | Terminal PDF browsing | `pdf document.pdf` |
| `ssl` | SSL certificate inspection | `ssl cert server.crt` |
| `ssl host` | TLS host scanning | `ssl host www.baidu.com` |
| `plate` | License plate lookup | `plate 湘` |
| `ipcalc` | IPv4 subnet calculator | `ipcalc 192.168.0.1/24` |
| `webserver` | Temporary HTTP server | `webserver -d . 8080` |
| `dbf` | dBase/FoxPro table reader | `dbf view 客户.dbf` |
| `ipgeo` | IP geolocation lookup (qqwry.dat) | `ipgeo 8.8.8.8` |

Full help: `lingbox <command> --help`

## License

Apache License 2.0
