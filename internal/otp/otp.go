// Package otp generates TOTP and HOTP one-time passwords (RFC 4226/6238).
package otp

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Defaults for one-time password generation.
const (
	DefaultDigits    = 6
	DefaultPeriod    = 30
	DefaultAlgorithm = "SHA1"
)

// Config controls HOTP/TOTP generation. Zero values select the defaults.
type Config struct {
	Digits    int
	Period    int
	Algorithm string
}

// Result is a generated TOTP code and its window metadata.
type Result struct {
	Code      string    `json:"code"`
	Algorithm string    `json:"algorithm"`
	Digits    int       `json:"digits"`
	Period    int       `json:"period"`
	Counter   uint64    `json:"counter"`
	Remaining int       `json:"remaining"`
	At        time.Time `json:"at"`
}

// URI is a parsed otpauth:// URI.
type URI struct {
	Type      string `json:"type"`
	Label     string `json:"label,omitempty"`
	Issuer    string `json:"issuer,omitempty"`
	Secret    string `json:"secret"`
	Algorithm string `json:"algorithm"`
	Digits    int    `json:"digits"`
	Period    int    `json:"period"`
	Counter   uint64 `json:"counter,omitempty"`
}

// Normalize applies defaults and validates the configuration.
func (c Config) Normalize() (Config, error) {
	if c.Digits == 0 {
		c.Digits = DefaultDigits
	}
	if c.Digits < 4 || c.Digits > 10 {
		return c, fmt.Errorf("digits must be between 4 and 10")
	}
	if c.Period == 0 {
		c.Period = DefaultPeriod
	}
	if c.Period < 0 {
		return c, fmt.Errorf("period must be positive")
	}
	if c.Algorithm == "" {
		c.Algorithm = DefaultAlgorithm
	}
	c.Algorithm = strings.ToUpper(c.Algorithm)
	switch c.Algorithm {
	case "SHA1", "SHA256", "SHA512":
	default:
		return c, fmt.Errorf("unsupported algorithm %q (supported: SHA1, SHA256, SHA512)", c.Algorithm)
	}
	return c, nil
}

func hashFor(algorithm string) func() hash.Hash {
	switch algorithm {
	case "SHA256":
		return sha256.New
	case "SHA512":
		return sha512.New
	default:
		return sha1.New
	}
}

// DecodeSecret decodes a base32 OTP secret, tolerating lowercase, spaces and
// padding.
func DecodeSecret(secret string) ([]byte, error) {
	s := strings.ToUpper(strings.Join(strings.Fields(secret), ""))
	s = strings.TrimRight(s, "=")
	if s == "" {
		return nil, fmt.Errorf("empty OTP secret")
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid base32 secret: %w", err)
	}
	return key, nil
}

func hotpCode(key []byte, counter uint64, cfg Config) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)

	mac := hmac.New(hashFor(cfg.Algorithm), key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	binaryCode := (uint64(sum[offset]&0x7f) << 24) |
		(uint64(sum[offset+1]) << 16) |
		(uint64(sum[offset+2]) << 8) |
		uint64(sum[offset+3])
	mod := uint64(1)
	for i := 0; i < cfg.Digits; i++ {
		mod *= 10
	}
	code := binaryCode % mod
	return fmt.Sprintf("%0*d", cfg.Digits, code)
}

// HOTP returns the counter-based one-time password.
func HOTP(secret string, counter uint64, cfg Config) (string, error) {
	cfg, err := cfg.Normalize()
	if err != nil {
		return "", err
	}
	key, err := DecodeSecret(secret)
	if err != nil {
		return "", err
	}
	return hotpCode(key, counter, cfg), nil
}

// TOTP returns the time-based one-time password for at.
func TOTP(secret string, at time.Time, cfg Config) (Result, error) {
	cfg, err := cfg.Normalize()
	if err != nil {
		return Result{}, err
	}
	key, err := DecodeSecret(secret)
	if err != nil {
		return Result{}, err
	}
	unix := at.Unix()
	if unix < 0 {
		return Result{}, fmt.Errorf("time must not be before 1970-01-01")
	}

	counter := uint64(unix / int64(cfg.Period))
	remaining := cfg.Period - int(unix%int64(cfg.Period))

	return Result{
		Code:      hotpCode(key, counter, cfg),
		Algorithm: cfg.Algorithm,
		Digits:    cfg.Digits,
		Period:    cfg.Period,
		Counter:   counter,
		Remaining: remaining,
		At:        at,
	}, nil
}

// ParseURI parses an otpauth:// URI as used by authenticator apps.
func ParseURI(raw string) (URI, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return URI{}, fmt.Errorf("invalid URI: %w", err)
	}
	if !strings.EqualFold(u.Scheme, "otpauth") {
		return URI{}, fmt.Errorf("not an otpauth:// URI")
	}

	typ := strings.ToLower(u.Host)
	if typ != "totp" && typ != "hotp" {
		return URI{}, fmt.Errorf("unsupported OTP type %q", u.Host)
	}

	q := u.Query()
	secret := q.Get("secret")
	if secret == "" {
		return URI{}, fmt.Errorf("URI has no secret")
	}
	if _, err := DecodeSecret(secret); err != nil {
		return URI{}, err
	}

	out := URI{
		Type:      typ,
		Label:     strings.TrimPrefix(u.Path, "/"),
		Issuer:    q.Get("issuer"),
		Secret:    secret,
		Algorithm: DefaultAlgorithm,
		Digits:    DefaultDigits,
		Period:    DefaultPeriod,
	}

	if raw := q.Get("algorithm"); raw != "" {
		out.Algorithm = strings.ToUpper(raw)
	}
	if raw := q.Get("digits"); raw != "" {
		d, err := strconv.Atoi(raw)
		if err != nil || d < 4 || d > 10 {
			return URI{}, fmt.Errorf("invalid digits %q", raw)
		}
		out.Digits = d
	}
	if raw := q.Get("period"); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil || p <= 0 {
			return URI{}, fmt.Errorf("invalid period %q", raw)
		}
		out.Period = p
	}

	if _, err := (Config{
		Digits:    out.Digits,
		Period:    out.Period,
		Algorithm: out.Algorithm,
	}).Normalize(); err != nil {
		return URI{}, err
	}

	if typ == "hotp" {
		raw := q.Get("counter")
		if raw == "" {
			return URI{}, fmt.Errorf("HOTP URI has no counter")
		}
		counter, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return URI{}, fmt.Errorf("invalid counter %q", raw)
		}
		out.Counter = counter
	}

	return out, nil
}
