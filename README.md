# ling-box

玲珑盒，一款工具集 - A collection of useful utility tools for developers.

## Features

- **URL Encoding/Decoding**: Encode and decode URL strings
- **Base64 Encoding/Decoding**: Encode and decode Base64 strings (supports URL-safe mode)
- **BCrypt Password Hashing**: Generate and verify bcrypt password hashes
- **QR Code Generation**: Generate QR codes as PNG, JPG, or GIF images
- **Password Generation**: Generate secure random passwords with customizable options

## Requirements

- Java 17 or higher
- Maven 3.6 or higher

## Building

```bash
mvn clean package
```

## Running

You can run the tool using Maven:

```bash
mvn exec:java -Dexec.args="<command> [options]"
```

Or build and run the JAR file:

```bash
mvn package
java -jar target/ling-box-1.0.0-SNAPSHOT.jar <command> [options]
```

## Usage

### General Help

```bash
mvn exec:java -Dexec.args="--help"
```

### URL Encoding/Decoding

```bash
# Encode a URL string
mvn exec:java -Dexec.args="url -e 'hello world'"
# Output: hello+world

# Decode a URL string
mvn exec:java -Dexec.args="url -d 'hello+world'"
# Output: hello world
```

### Base64 Encoding/Decoding

```bash
# Encode a string to Base64
mvn exec:java -Dexec.args="base64 -e 'Hello World'"
# Output: SGVsbG8gV29ybGQ=

# Decode a Base64 string
mvn exec:java -Dexec.args="base64 -d 'SGVsbG8gV29ybGQ='"
# Output: Hello World

# Use URL-safe Base64
mvn exec:java -Dexec.args="base64 -e -u 'test+/string'"
```

### BCrypt Password Hashing

```bash
# Generate a bcrypt hash
mvn exec:java -Dexec.args="bcrypt -g mypassword"
# Output: $2a$12$...

# Verify a password against a hash
mvn exec:java -Dexec.args="bcrypt -v mypassword '\$2a\$12\$...' "
```

### QR Code Generation

```bash
# Generate a QR code (default: qrcode.png, 300x300)
mvn exec:java -Dexec.args="qrcode 'https://example.com'"

# Custom output file and size
mvn exec:java -Dexec.args="qrcode 'Hello World' -o mycode.png -s 500"

# Different format
mvn exec:java -Dexec.args="qrcode 'Test' -o mycode.jpg -f JPG"
```

### Password Generation

```bash
# Generate a 16-character password (default)
mvn exec:java -Dexec.args="password"

# Generate a 24-character password
mvn exec:java -Dexec.args="password -l 24"

# Generate multiple passwords
mvn exec:java -Dexec.args="password -c 5"

# Generate digits-only password
mvn exec:java -Dexec.args="password -d"

# Generate uppercase-only password
mvn exec:java -Dexec.args="password -u"

# Generate password without special characters
mvn exec:java -Dexec.args="password -n"
```

## Cross-Platform Support

This tool is built with Kotlin/JVM and runs on any platform that supports Java 17:
- Windows
- macOS
- Linux

### Native Binaries

Pre-built native binaries are available for each release:
- `ling-box-linux-amd64` - Linux x86_64
- `ling-box-windows-amd64.exe` - Windows x86_64
- `ling-box-macos-amd64` - macOS x86_64

Download from the [Releases](https://github.com/wgzhao/ling-box/releases) page.

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
