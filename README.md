# ling-box

玲珑盒，一款工具集 - A collection of useful utility tools for developers.

## Features

- **URL Encoding/Decoding**: Encode and decode URL strings
- **Base64 Encoding/Decoding**: Encode and decode Base64 strings (supports URL-safe mode)
- **BCrypt Password Hashing**: Generate and verify bcrypt password hashes
- **QR Code Generation**: Generate QR codes as PNG, JPG, or GIF images
- **Password Generation**: Generate secure random passwords with customizable options

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

## Cross-Platform Support

This tool is written in Go and compiles to a single static binary for any platform:
- Windows
- macOS
- Linux

## Release Process

To create a new release:

```bash
# Prepare the release (updates version, creates tag)
mvn release:prepare

# Perform the release
mvn release:perform
```

This will automatically trigger GitHub Actions to:
1. Build native binaries for Linux, Windows, and macOS
2. Generate release notes from commits between tags
3. Create a GitHub release with all binaries attached

## License

Apache License 2.0
