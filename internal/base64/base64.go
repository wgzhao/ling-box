package base64

import (
	"encoding/base64"
)

// Encode encodes a string to Base64. If urlSafe is true, uses URL-safe encoding.
func Encode(input string, urlSafe bool) string {
	if urlSafe {
		return base64.URLEncoding.EncodeToString([]byte(input))
	}
	return base64.StdEncoding.EncodeToString([]byte(input))
}

// Decode decodes a Base64 string. If urlSafe is true, uses URL-safe decoding.
func Decode(input string, urlSafe bool) (string, error) {
	var decoded []byte
	var err error

	if urlSafe {
		decoded, err = base64.URLEncoding.DecodeString(input)
	} else {
		decoded, err = base64.StdEncoding.DecodeString(input)
	}

	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
