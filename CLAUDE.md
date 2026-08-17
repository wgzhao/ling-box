# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build the binary
go build -o lingbox .

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/url/...

# Run a single test
go test ./internal/url -run TestEncode

# Run tests verbosely
go test -v ./...

# Install dependencies
go mod tidy
```

## Release

```bash
# Build with the version injected (--version flag reads it)
go build -ldflags "-X github.com/wgzhao/ling-box/cmd.version=vX.Y.Z" -o lingbox .
```

1. Tag and push: `git tag vX.Y.Z && git push origin vX.Y.Z`
2. Update the Homebrew formula: `Formula/lingbox.rb` in [wgzhao/homebrew-tap](https://github.com/wgzhao/homebrew-tap) (bump `url` tag + `sha256` of the new tarball), push to the tap repo. The formula builds from source on all platforms (cgo statically links MuPDF); there are no precompiled-binary branches. Note: release assets built by cross-compilation lack MuPDF and cannot be used as formula downloads.
3. The homebrew-core formula (branch `lingbox` in wgzhao/homebrew-core fork) is parked until the repo meets Homebrew's notability requirement (≥30 forks or ≥30 watchers or ≥75 stars; ×3 for self-submitted PRs). Submit with `brew bump-formula-pr` / PR from that branch once met.

## Architecture

The project is a Go CLI toolbox (玲珑盒) built with Cobra. Every tool follows the same two-layer pattern:

### Command-Utility Separation

Each subcommand has two packages:

1. **`cmd/`** — Cobra command definitions. Handles CLI argument parsing and I/O. Never contains business logic.

2. **`internal/<tool>/`** — Utility packages with pure functions for the actual logic. Stateless, no CLI dependency. This is what tests exercise.

Example: `cmd/url.go` parses `-e`/`-d` flags and calls `url.Encode()`/`url.Decode()`.

### Entry Point

`main.go` calls `cmd.Execute()` — the root Cobra command which adds all subcommands (`url`, `base64`, `bcrypt`, `qrcode`, `password`, `uuid`, `json`, `convert`, `unicode`, `color`, `ssl`, `plate`).

### Dependencies

| Purpose | Library |
|---------|---------|
| CLI framework | github.com/spf13/cobra |
| BCrypt hashing | golang.org/x/crypto/bcrypt |
| QR code generation | github.com/skip2/go-qrcode |
| UUID generation | github.com/google/uuid |
| YAML parsing | gopkg.in/yaml.v3 |
| PDF rendering | github.com/gen2brain/go-fitz (MuPDF bindings) |

### Testing

Tests live in `internal/<tool>/` alongside the source. Each package has a `_test.go` file. All tests use the standard `testing` package — no external test framework.

## Source Layout

```
main.go                  # Entry point
cmd/
  root.go                # Root command definition
  url.go                 # URL encode/decode
  base64.go              # Base64 encode/decode
  bcrypt.go              # BCrypt hash/verify
  qrcode.go              # QR code generation
  password.go            # Password generation
  uuid.go                # UUID generation (v1, v4, v7)
  convert.go             # Format conversion: -i/-o/-t (json/yaml/csv/markdown)
  json.go                # JSON tools: format, verify
  unicode.go             # Unicode encode/decode
  color.go               # Color code conversion (Hex/RGB/HSL)
  imgcat.go              # Terminal image display
  pdf.go                 # Terminal PDF rendering & browsing
  ssl.go                 # SSL tools: cert inspection + host scanning
  plate.go               # License plate region lookup
internal/
  url/
    url.go               # URL utility functions
    url_test.go
  base64/
    base64.go            # Base64 utility functions
    base64_test.go
  bcrypt/
    bcrypt.go            # BCrypt utility functions
    bcrypt_test.go
  qrcode/
    qrcode.go            # QR code generation functions
    qrcode_test.go
  password/
    password.go          # Password generation functions
    password_test.go
  uuid/
    uuid.go              # UUID generation (v1, v4, v7)
    uuid_test.go
  convert/
    convert.go           # Parse/Render layer + YAML↔JSON conversion
    convert_test.go
    csv.go               # CSV parse/render + input format detection
    csv_test.go
    csv_input_test.go
    markdown.go          # Markdown renderer (tables, lists, key/value)
    markdown_test.go
  jsonx/
    jsonx.go             # JSON formatting & validation (key order preserved)
    jsonx_test.go
  unicode/
    unicode.go           # Unicode \uXXXX encode/decode
    unicode_test.go
  color/
    color.go             # Color format conversion (Hex/RGB/HSL/named)
    color_test.go
  imgcat/
    imgcat.go            # Terminal image rendering engine
    imgcat_test.go
    browse.go            # Multi-image interactive browser
    browse_test.go
  pdf/
    pdf.go               # PDF page rendering (MuPDF wrapper)
    pdf_test.go
  ssl/
    cert.go              # X.509 certificate parsing & inspection
    cert_test.go
    ciphers.go           # Cipher suite security rating database
    host.go              # TLS host scanning (protocols, suites, trust)
    host_test.go
  plate/
    data.go              # Plate code database (generated from qq.com page)
    plate.go             # Province lookup (short/full/suffixless names)
    plate_test.go
```
