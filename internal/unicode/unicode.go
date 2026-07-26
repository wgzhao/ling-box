package unicode

import (
	"fmt"
	"strconv"
	"strings"
)

// Encode converts a string to Unicode escape sequences (\uXXXX).
// ASCII printable characters (0x20-0x7E) are kept as-is; everything else is escaped.
func Encode(input string) string {
	var buf strings.Builder
	for _, r := range input {
		if r >= 0x20 && r <= 0x7E {
			buf.WriteRune(r)
		} else if r <= 0xFFFF {
			buf.WriteString(fmt.Sprintf("\\u%04X", r))
		} else {
			// Surrogate pair for characters outside BMP
			r1, r2 := splitRune(r)
			buf.WriteString(fmt.Sprintf("\\u%04X\\u%04X", r1, r2))
		}
	}
	return buf.String()
}

// Decode converts Unicode escape sequences back to a human-readable string.
// Supports both \uXXXX and \UXXXXXXXX formats.
func Decode(input string) (string, error) {
	var buf strings.Builder
	i := 0
	for i < len(input) {
		if input[i] == '\\' && i+1 < len(input) {
			switch input[i+1] {
			case 'u':
				if i+6 <= len(input) {
					code, err := parseHex(input[i+2 : i+6])
					if err != nil {
						return "", fmt.Errorf("invalid unicode escape at position %d: %w", i, err)
					}
					// Check for surrogate pair
					if isHighSurrogate(code) && i+12 <= len(input) && input[i+6] == '\\' && input[i+7] == 'u' {
						low, err := parseHex(input[i+8 : i+12])
						if err == nil && isLowSurrogate(low) {
							r := combineSurrogate(code, low)
							buf.WriteRune(r)
							i += 12
							continue
						}
					}
					buf.WriteRune(code)
					i += 6
					continue
				}
				// Incomplete escape sequence
				return "", fmt.Errorf("incomplete unicode escape at position %d", i)
			case 'U':
				if i+10 <= len(input) {
					code, err := parseHex(input[i+2 : i+10])
					if err != nil {
						return "", fmt.Errorf("invalid unicode escape at position %d: %w", i, err)
					}
					buf.WriteRune(code)
					i += 10
					continue
				}
			}
		}
		buf.WriteByte(input[i])
		i++
	}
	return buf.String(), nil
}

// EncodeAll escapes every non-ASCII character, including ASCII control chars.
// Useful for generating fully-escaped strings.
func EncodeAll(input string) string {
	var buf strings.Builder
	for _, r := range input {
		if r <= 0xFFFF {
			buf.WriteString(fmt.Sprintf("\\u%04X", r))
		} else {
			r1, r2 := splitRune(r)
			buf.WriteString(fmt.Sprintf("\\u%04X\\u%04X", r1, r2))
		}
	}
	return buf.String()
}

// IsEncoded checks if a string contains any unicode escape sequences.
func IsEncoded(input string) bool {
	return strings.Contains(input, "\\u")
}

func parseHex(s string) (rune, error) {
	val, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, err
	}
	return rune(val), nil
}

func splitRune(r rune) (rune, rune) {
	r -= 0x10000
	return 0xD800 + (r>>10)&0x3FF, 0xDC00 + r&0x3FF
}

func isHighSurrogate(r rune) bool {
	return r >= 0xD800 && r <= 0xDBFF
}

func isLowSurrogate(r rune) bool {
	return r >= 0xDC00 && r <= 0xDFFF
}

func combineSurrogate(high, low rune) rune {
	return 0x10000 + (high-0xD800)<<10 + (low - 0xDC00)
}
