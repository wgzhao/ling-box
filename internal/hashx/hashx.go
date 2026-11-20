// Package hashx computes message digests and HMACs for the hash command.
package hashx

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"strings"
)

type constructor func() hash.Hash

var plainHashes = map[string]constructor{
	"md5":    md5.New,
	"sha1":   sha1.New,
	"sha224": sha256.New224,
	"sha256": sha256.New,
	"sha384": sha512.New384,
	"sha512": sha512.New,
	"sm3":    NewSM3,
	"crc32":  func() hash.Hash { return crc32.NewIEEE() },
}

// Algorithms returns every supported algorithm name, plain hashes first.
func Algorithms() []string {
	return []string{
		"md5", "sha1", "sha224", "sha256", "sha384", "sha512", "sm3", "crc32",
		"hmac-sha1", "hmac-sha256", "hmac-sha384", "hmac-sha512", "hmac-sm3",
	}
}

// Sum hashes data and returns the lowercase hex digest.
//
// algo is one of the plain names from Algorithms (md5, sha256, sm3, ...) or
// an hmac-<hash> name. key is required for hmac-* and must be empty otherwise.
func Sum(data []byte, algo string, key []byte) (string, error) {
	return SumReader(bytes.NewReader(data), algo, key)
}

// SumReader streams r through the selected digest.
func SumReader(r io.Reader, algo string, key []byte) (string, error) {
	a := strings.ToLower(strings.TrimSpace(algo))

	var h hash.Hash
	if newHash, ok := plainHashes[a]; ok {
		if len(key) > 0 {
			return "", fmt.Errorf("%s does not use a key (use hmac-%s for a keyed digest)", a, a)
		}
		h = newHash()
	} else if base, ok := strings.CutPrefix(a, "hmac-"); ok {
		newHash, ok := plainHashes[base]
		if !ok {
			return "", unknownAlgorithm(algo)
		}
		if len(key) == 0 {
			return "", fmt.Errorf("%s requires a key", a)
		}
		h = hmac.New(newHash, key)
	} else {
		return "", unknownAlgorithm(algo)
	}

	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func unknownAlgorithm(algo string) error {
	return fmt.Errorf("unknown algorithm %q (supported: %s)", algo, strings.Join(Algorithms(), ", "))
}
