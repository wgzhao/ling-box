package password

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	lowercase  = "abcdefghijklmnopqrstuvwxyz"
	uppercase  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits     = "0123456789"
	special    = "!@#$%^&*()_+-=[]{}|;:,.<>?"
)

// Options configures password generation.
type Options struct {
	Length         int
	IncludeSpecial bool
	UppercaseOnly  bool
	DigitsOnly     bool
	// ExcludeChars is a set of characters to remove from the final charset.
	// This is useful for excluding characters that cause encoding issues in
	// certain systems (e.g., |, !, [, ], $, backtick).
	ExcludeChars string
}

// removeChars strips every character in chars from s.
func removeChars(s, chars string) string {
	exclude := make(map[rune]bool, len(chars))
	for _, c := range chars {
		exclude[c] = true
	}
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if !exclude[rune(s[i])] {
			result = append(result, s[i])
		}
	}
	return string(result)
}

// Generate creates a secure random password according to the given options.
func Generate(opts Options) (string, error) {
	charset := lowercase + uppercase + digits
	switch {
	case opts.DigitsOnly:
		charset = digits
	case opts.UppercaseOnly:
		charset = uppercase
	case !opts.IncludeSpecial:
		charset = lowercase + uppercase + digits
	default:
		charset = lowercase + uppercase + digits + special
	}

	if opts.ExcludeChars != "" {
		charset = removeChars(charset, opts.ExcludeChars)
		if len(charset) == 0 {
			return "", fmt.Errorf("charset is empty after excluding characters")
		}
	}

	length := opts.Length
	if length <= 0 {
		length = 16
	}

	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := range length {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}

	return string(result), nil
}
