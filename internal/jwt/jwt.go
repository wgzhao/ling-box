// Package jwt parses and verifies JSON Web Tokens.
package jwt

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Token is a decoded JWT. The signature is kept as raw bytes so callers can
// verify it or display its size.
type Token struct {
	Header    map[string]any
	Claims    map[string]any
	Signature []byte

	alg          string
	signingInput string
}

// Algorithm returns the "alg" header value.
func (t *Token) Algorithm() string { return t.alg }

// SigningInput returns the header.payload bytes covered by the signature.
func (t *Token) SigningInput() []byte { return []byte(t.signingInput) }

// Parse splits and decodes a JWT without verifying its signature.
func Parse(token string) (*Token, error) {
	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed JWT: expected 3 segments, got %d", len(parts))
	}

	header, err := decodeObject(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	claims, err := decodeObject(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	sig, err := decodeSegment(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	alg, _ := header["alg"].(string)
	if alg == "" {
		return nil, fmt.Errorf(`malformed JWT: header has no "alg" string`)
	}

	return &Token{
		Header:       header,
		Claims:       claims,
		Signature:    sig,
		alg:          alg,
		signingInput: parts[0] + "." + parts[1],
	}, nil
}

func decodeSegment(s string) ([]byte, error) {
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return nil, fmt.Errorf("invalid base64url data")
}

func decodeObject(seg string) (map[string]any, error) {
	b, err := decodeSegment(seg)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var obj map[string]any
	if err := dec.Decode(&obj); err != nil {
		return nil, fmt.Errorf("not a JSON object: %w", err)
	}
	if obj == nil {
		return nil, fmt.Errorf("not a JSON object")
	}
	return obj, nil
}

// TimeStatus reports the registered time claims relative to now.
type TimeStatus struct {
	Expired     bool       `json:"expired"`
	NotYetValid bool       `json:"not_yet_valid"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	NotBefore   *time.Time `json:"not_before,omitempty"`
	IssuedAt    *time.Time `json:"issued_at,omitempty"`
}

// Times evaluates the exp, nbf and iat claims. Invalid values are ignored.
func Times(claims map[string]any, now time.Time) TimeStatus {
	var st TimeStatus
	if exp, ok := numericDate(claims["exp"]); ok {
		st.ExpiresAt = &exp
		st.Expired = !now.Before(exp)
	}
	if nbf, ok := numericDate(claims["nbf"]); ok {
		st.NotBefore = &nbf
		st.NotYetValid = now.Before(nbf)
	}
	if iat, ok := numericDate(claims["iat"]); ok {
		st.IssuedAt = &iat
	}
	return st
}

func numericDate(v any) (time.Time, bool) {
	var f float64
	switch n := v.(type) {
	case json.Number:
		parsed, err := n.Float64()
		if err != nil {
			return time.Time{}, false
		}
		f = parsed
	case float64:
		f = n
	case int64:
		f = float64(n)
	case int:
		f = float64(n)
	case string:
		parsed, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return time.Time{}, false
		}
		f = parsed
	default:
		return time.Time{}, false
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return time.Time{}, false
	}
	sec := int64(f)
	nsec := int64((f - float64(sec)) * 1e9)
	return time.Unix(sec, nsec), true
}
