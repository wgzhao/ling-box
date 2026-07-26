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

## Architecture

The project is a Go CLI toolbox (玲珑盒) built with Cobra. Every tool follows the same two-layer pattern:

### Command-Utility Separation

Each subcommand has two packages:

1. **`cmd/`** — Cobra command definitions. Handles CLI argument parsing and I/O. Never contains business logic.

2. **`internal/<tool>/`** — Utility packages with pure functions for the actual logic. Stateless, no CLI dependency. This is what tests exercise.

Example: `cmd/url.go` parses `-e`/`-d` flags and calls `url.Encode()`/`url.Decode()`.

### Entry Point

`main.go` calls `cmd.Execute()` — the root Cobra command which adds all subcommands (`url`, `base64`, `bcrypt`, `qrcode`, `password`, `uuid`, `convert`, `unicode`, `color`).

### Dependencies

| Purpose | Library |
|---------|---------|
| CLI framework | github.com/spf13/cobra |
| BCrypt hashing | golang.org/x/crypto/bcrypt |
| QR code generation | github.com/skip2/go-qrcode |
| UUID generation | github.com/google/uuid |
| YAML parsing | gopkg.in/yaml.v3 |

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
  yaml2json.go           # YAML to JSON conversion
  json2yaml.go           # JSON to YAML conversion
  unicode.go             # Unicode encode/decode
  color.go               # Color code conversion (Hex/RGB/HSL)
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
    convert.go           # YAML↔JSON bidirectional conversion
    convert_test.go
  unicode/
    unicode.go           # Unicode \uXXXX encode/decode
    unicode_test.go
  color/
    color.go             # Color format conversion (Hex/RGB/HSL/named)
    color_test.go
```
