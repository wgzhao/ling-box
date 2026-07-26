package password

import (
	"crypto/rand"
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

	length := opts.Length
	if length <= 0 {
		length = 16
	}

	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}

	return string(result), nil
}
