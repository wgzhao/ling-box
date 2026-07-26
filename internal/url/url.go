package url

import (
	"net/url"
)

// Encode encodes a string for use in a URL query string.
func Encode(input string) string {
	return url.QueryEscape(input)
}

// Decode decodes a URL-encoded string.
func Decode(input string) string {
	result, err := url.QueryUnescape(input)
	if err != nil {
		return input
	}
	return result
}
