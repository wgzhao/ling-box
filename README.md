# lingbox

玲珑盒，一款工具集 - A collection of useful utility tools for developers.

> [中文文档](README_zh.md)

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
- **Terminal Image Display**: Display images directly in the terminal (iTerm2/Kitty/half-block/ASCII)

## Requirements

- Go 1.21 or higher

## Building

```bash
go build -o lingbox .
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

### YAML/JSON Conversion

```bash
# Convert JSON to YAML (from file)
./lingbox json2yaml data.json

# Convert YAML to JSON (from file)
./lingbox yaml2json config.yaml

# Pipe data
cat data.json | ./lingbox json2yaml
curl -s https://api.example.com/data | ./lingbox json2yaml
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

# Browse all images in a directory
./lingbox imgcat -d ~/Pictures
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

After enabling, type `./lingbox yam<Tab>` to auto-complete to `yaml2json`.

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
| `json2yaml` | JSON → YAML | `json2yaml data.json` |
| `yaml2json` | YAML → JSON | `yaml2json config.yaml` |
| `unicode` | Unicode encoder/decoder | `unicode -e '你好'` |
| `color` | Color converter | `color '#FF0000'` |
| `bmi` | BMI calculator | `bmi 170 65` |
| `base` | Base converter | `base FF -f hex` |
| `date` | Date calculator | `date diff 2026-01-01 2026-07-26` |
| `imgcat` | Terminal image display | `imgcat photo.jpg` or `imgcat -d ~/Pictures` |

Full help: `lingbox <command> --help`

## License

Apache License 2.0
