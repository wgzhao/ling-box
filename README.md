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
- Gradle 8.0 or higher (or use the included Gradle wrapper)

## Building

```bash
./gradlew build
```

## Running

You can run the tool using Gradle:

```bash
./gradlew run --args="<command> [options]"
```

Or build and run the JAR file:

```bash
./gradlew jar
java -jar build/libs/ling-box-1.0.0-SNAPSHOT.jar <command> [options]
```

## Usage

### General Help

```bash
./gradlew run --args="--help"
```

### URL Encoding/Decoding

```bash
# Encode a URL string
./gradlew run --args="url -e 'hello world'"
# Output: hello+world

# Decode a URL string
./gradlew run --args="url -d 'hello+world'"
# Output: hello world
```

### Base64 Encoding/Decoding

```bash
# Encode a string to Base64
./gradlew run --args="base64 -e 'Hello World'"
# Output: SGVsbG8gV29ybGQ=

# Decode a Base64 string
./gradlew run --args="base64 -d 'SGVsbG8gV29ybGQ='"
# Output: Hello World

# Use URL-safe Base64
./gradlew run --args="base64 -e -u 'test+/string'"
```

### BCrypt Password Hashing

```bash
# Generate a bcrypt hash
./gradlew run --args="bcrypt -g mypassword"
# Output: $2a$12$...

# Verify a password against a hash
./gradlew run --args="bcrypt -v mypassword '\$2a\$12\$...' "
```

### QR Code Generation

```bash
# Generate a QR code (default: qrcode.png, 300x300)
./gradlew run --args="qrcode 'https://example.com'"

# Custom output file and size
./gradlew run --args="qrcode 'Hello World' -o mycode.png -s 500"

# Different format
./gradlew run --args="qrcode 'Test' -o mycode.jpg -f JPG"
```

### Password Generation

```bash
# Generate a 16-character password (default)
./gradlew run --args="password"

# Generate a 24-character password
./gradlew run --args="password -l 24"

# Generate multiple passwords
./gradlew run --args="password -c 5"

# Generate digits-only password
./gradlew run --args="password -d"

# Generate uppercase-only password
./gradlew run --args="password -u"

# Generate password without special characters
./gradlew run --args="password -n"
```

## Cross-Platform Support

This tool is built with Kotlin/JVM and runs on any platform that supports Java 17:
- Windows
- macOS
- Linux

## License

Apache License 2.0
