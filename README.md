# ling-box

玲珑盒，一款工具集 - A collection of useful utility tools for developers.

## Features

- **URL Encoding/Decoding**: Encode and decode URL strings
- **Base64 Encoding/Decoding**: Encode and decode Base64 strings (supports URL-safe mode)
- **BCrypt Password Hashing**: Generate and verify bcrypt password hashes
- **QR Code Generation**: Generate QR codes as PNG, JPG, or GIF images
- **Password Generation**: Generate secure random passwords with customizable options
- **UUID Generation**: Generate UUIDs (v1, v4, v7) in bulk
- **YAML/JSON Conversion**: Bidirectional conversion between YAML and JSON
- **Unicode Encoding/Decoding**: Encode text to \uXXXX escapes and decode back
- **Color Conversion**: Convert between Hex, RGB, HSL, and named colors
- **BMI Calculator**: Calculate Body Mass Index from height and weight
- **Number Base Conversion**: Convert between binary, octal, decimal, and hexadecimal
- **Date Calculator**: Add/subtract days or calculate date differences

## Requirements

- Go 1.21 or higher

## Building

```bash
go build -o ling-box .
```

## Running

```bash
./ling-box <command> [options]
```

## Usage

### General Help

```bash
./ling-box --help
```

### URL Encoding/Decoding

```bash
# Encode a URL string
./ling-box url -e 'hello world'
# Output: hello+world

# Decode a URL string
./ling-box url -d 'hello+world'
# Output: hello world
```

### Base64 Encoding/Decoding

```bash
# Encode a string to Base64
./ling-box base64 -e 'Hello World'
# Output: SGVsbG8gV29ybGQ=

# Decode a Base64 string
./ling-box base64 -d 'SGVsbG8gV29ybGQ='
# Output: Hello World

# Use URL-safe Base64 (flag placement matters: -u before -e)
./ling-box base64 -u -e 'test+/string'
```

### BCrypt Password Hashing

```bash
# Generate a bcrypt hash
./ling-box bcrypt -g mypassword
# Output: $2a$12$...

# Verify a password against a hash
./ling-box bcrypt -v mypassword '$2a$12$...'
```

### QR Code Generation

```bash
# Generate a QR code (default: qrcode.png, 300x300)
./ling-box qrcode 'https://example.com'

# Custom output file and size
./ling-box qrcode 'Hello World' -o mycode.png -s 500

# Different format
./ling-box qrcode 'Test' -o mycode.jpg -f JPG
```

### Password Generation

```bash
# Generate a 16-character password (default)
./ling-box password

# Generate a 24-character password
./ling-box password -l 24

# Generate multiple passwords
./ling-box password -c 5

# Generate digits-only password
./ling-box password -d

# Generate uppercase-only password
./ling-box password -u

# Generate password without special characters
./ling-box password -n
```

### UUID Generation

```bash
# Generate a v4 UUID (default)
./ling-box uuid

# Generate 5 UUIDs
./ling-box uuid -n 5

# Generate UUID v7 (time-ordered, sortable)
./ling-box uuid -t v7

# Generate UUID v1 (time-based)
./ling-box uuid -t v1
```

### YAML/JSON Conversion

```bash
# Convert JSON to YAML (from file)
./ling-box json2yaml data.json

# Convert YAML to JSON (from file)
./ling-box yaml2json config.yaml

# Pipe data
cat data.json | ./ling-box json2yaml
curl -s https://api.example.com/data | ./ling-box json2yaml
```

### Unicode Encoding/Decoding

```bash
# Encode Chinese characters to \uXXXX
./ling-box unicode -e '你好世界'
# Output: 你好世界

# Decode \uXXXX back to text
./ling-box unicode -d '你好世界'
# Output: 你好世界

# Auto-detect mode (encodes if plain text, decodes if \uXXXX)
./ling-box unicode '你好'
```

### Color Conversion

```bash
# Named color
./ling-box color 'red'
# Output: Hex, RGB, HSL, and name

# Hex input
./ling-box color '#FF0000'

# RGB input
./ling-box color 'rgb(255, 0, 0)'

# HSL input
./ling-box color 'hsl(0, 100%, 50%)'

# Multi-word named color
./ling-box color 'dark gray'
./ling-box color 'light yellow'
```

## Shell Completion

Shell completion is built-in via Cobra. Enable tab completion for commands, flags, and subcommands:

```bash
# Bash
source <(./ling-box completion bash)

# Zsh
source <(./ling-box completion zsh)

# Fish (your shell)
./ling-box completion fish > ~/.config/fish/completions/ling-box.fish

# PowerShell
./ling-box completion powershell | Out-String | Invoke-Expression
```

After enabling, type `./ling-box yam<Tab>` to auto-complete to `yaml2json`.

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

## License

Apache License 2.0
