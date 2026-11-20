package jwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"
	"time"
)

func segment(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func hs256Token(t *testing.T, claims map[string]any, secret string) string {
	t.Helper()
	h := segment(t, map[string]any{"alg": "HS256", "typ": "JWT"})
	p := segment(t, claims)
	input := h + "." + p
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return input + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestParseAndVerifyHS256(t *testing.T) {
	token := hs256Token(t, map[string]any{
		"sub": "1234567890",
		"exp": 4102444800,
	}, "s3cret")

	tok, err := Parse(token)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if tok.Algorithm() != "HS256" {
		t.Errorf("alg = %q", tok.Algorithm())
	}
	if tok.Claims["sub"] != "1234567890" {
		t.Errorf("sub = %v", tok.Claims["sub"])
	}
	if err := Verify(tok, []byte("s3cret")); err != nil {
		t.Errorf("Verify: %v", err)
	}
	if err := Verify(tok, []byte("wrong")); err == nil {
		t.Error("Verify with wrong secret: expected error")
	}
}

func TestVerifyDetectsTampering(t *testing.T) {
	token := hs256Token(t, map[string]any{"sub": "alice"}, "s3cret")
	parts := strings.Split(token, ".")
	parts[1] = segment(t, map[string]any{"sub": "admin"})
	tok, err := Parse(strings.Join(parts, "."))
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(tok, []byte("s3cret")); err == nil {
		t.Error("tampered token verified")
	}
}

func TestVerifyRS256(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	input := segment(t, map[string]any{"alg": "RS256"}) + "." + segment(t, map[string]any{"sub": "rsa"})
	sum := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	token := input + "." + base64.RawURLEncoding.EncodeToString(sig)

	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	tok, err := Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(tok, pubPEM); err != nil {
		t.Errorf("Verify RS256: %v", err)
	}
}

func TestVerifyES256(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	input := segment(t, map[string]any{"alg": "ES256"}) + "." + segment(t, map[string]any{"sub": "ecdsa"})
	sum := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, key, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	raw := make([]byte, 64)
	r.FillBytes(raw[:32])
	s.FillBytes(raw[32:])
	token := input + "." + base64.RawURLEncoding.EncodeToString(raw)

	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	tok, err := Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(tok, pubPEM); err != nil {
		t.Errorf("Verify ES256: %v", err)
	}
}

func TestTimeStatus(t *testing.T) {
	now := time.Unix(150, 0)
	st := Times(map[string]any{
		"exp": json.Number("100"),
		"nbf": float64(200),
		"iat": json.Number("50"),
	}, now)

	if !st.Expired || !st.NotYetValid {
		t.Errorf("status = %+v, want expired and not yet valid", st)
	}
	if st.ExpiresAt == nil || !st.ExpiresAt.Equal(time.Unix(100, 0)) {
		t.Errorf("expires_at = %v", st.ExpiresAt)
	}
	if st.IssuedAt == nil || !st.IssuedAt.Equal(time.Unix(50, 0)) {
		t.Errorf("issued_at = %v", st.IssuedAt)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []string{
		"",
		"a.b",
		"!!!.e30.",
		base64.RawURLEncoding.EncodeToString([]byte("not-json")) + "." + segment(t, map[string]any{"a": 1}) + ".",
		segment(t, map[string]any{"alg": "HS256"}) + "." + segment(t, []int{1, 2}) + ".",
	}
	for _, token := range tests {
		if _, err := Parse(token); err == nil {
			t.Errorf("Parse(%q): expected error", token)
		}
	}
}

func TestVerifyRejectsNoneAndUnknown(t *testing.T) {
	for _, alg := range []string{"none", "HS999"} {
		token := segment(t, map[string]any{"alg": alg}) + "." + segment(t, map[string]any{"a": 1}) + "."
		tok, err := Parse(token)
		if err != nil {
			t.Fatal(err)
		}
		if err := Verify(tok, []byte("key")); err == nil {
			t.Errorf("alg %q: expected verification error", alg)
		}
	}
}

func TestVerifyRejectsWrongKeyType(t *testing.T) {
	token := hs256Token(t, map[string]any{"a": 1}, "secret")
	tok, err := Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	if err := Verify(tok, pubPEM); err == nil {
		t.Error("HS256 verified against an RSA key")
	}
}
