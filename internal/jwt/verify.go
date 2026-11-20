package jwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
)

// Verify checks the token signature. key is the HMAC secret for HS* tokens,
// or a PEM-encoded public key, certificate, or private key for RS/PS/ES.
func Verify(t *Token, key []byte) error {
	alg := t.Algorithm()
	switch {
	case strings.HasPrefix(alg, "HS"):
		return verifyHMAC(t, alg, key)
	case strings.HasPrefix(alg, "RS"), strings.HasPrefix(alg, "PS"):
		return verifyRSA(t, alg, key)
	case strings.HasPrefix(alg, "ES"):
		return verifyECDSA(t, alg, key)
	case alg == "none":
		return fmt.Errorf(`alg "none" carries no signature and cannot be verified`)
	default:
		return fmt.Errorf("unsupported algorithm %q", alg)
	}
}

func verifyHMAC(t *Token, alg string, key []byte) error {
	if len(key) == 0 {
		return fmt.Errorf("HS* tokens require an HMAC secret")
	}
	h, err := hashFor(alg)
	if err != nil {
		return err
	}
	mac := hmac.New(h.New, key)
	mac.Write(t.SigningInput())
	if !hmac.Equal(mac.Sum(nil), t.Signature) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

func verifyRSA(t *Token, alg string, key []byte) error {
	h, err := hashFor(alg)
	if err != nil {
		return err
	}
	pub, err := parsePublicKey(key)
	if err != nil {
		return err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("%s requires an RSA key, got %T", alg, pub)
	}
	sum := hashBytes(h, t.SigningInput())
	if strings.HasPrefix(alg, "PS") {
		err = rsa.VerifyPSS(rsaPub, h, sum, t.Signature, &rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthEqualsHash,
			Hash:       h,
		})
	} else {
		err = rsa.VerifyPKCS1v15(rsaPub, h, sum, t.Signature)
	}
	if err != nil {
		return fmt.Errorf("signature mismatch: %w", err)
	}
	return nil
}

func verifyECDSA(t *Token, alg string, key []byte) error {
	h, err := hashFor(alg)
	if err != nil {
		return err
	}

	var curve elliptic.Curve
	switch alg {
	case "ES256":
		curve = elliptic.P256()
	case "ES384":
		curve = elliptic.P384()
	case "ES512":
		curve = elliptic.P521()
	default:
		return fmt.Errorf("unsupported algorithm %q", alg)
	}

	pub, err := parsePublicKey(key)
	if err != nil {
		return err
	}
	ecPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("%s requires an ECDSA key, got %T", alg, pub)
	}
	if ecPub.Curve != curve {
		return fmt.Errorf("%s requires curve %s", alg, curve.Params().Name)
	}

	size := (curve.Params().BitSize + 7) / 8
	if len(t.Signature) != 2*size {
		return fmt.Errorf("invalid ECDSA signature length %d, want %d", len(t.Signature), 2*size)
	}
	r := new(big.Int).SetBytes(t.Signature[:size])
	s := new(big.Int).SetBytes(t.Signature[size:])

	sum := hashBytes(h, t.SigningInput())
	if !ecdsa.Verify(ecPub, sum, r, s) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

func hashFor(alg string) (crypto.Hash, error) {
	switch {
	case strings.HasSuffix(alg, "256"):
		return crypto.SHA256, nil
	case strings.HasSuffix(alg, "384"):
		return crypto.SHA384, nil
	case strings.HasSuffix(alg, "512"):
		return crypto.SHA512, nil
	default:
		return 0, fmt.Errorf("unsupported algorithm %q", alg)
	}
}

func hashBytes(h crypto.Hash, data []byte) []byte {
	hasher := h.New()
	hasher.Write(data)
	return hasher.Sum(nil)
}

func parsePublicKey(data []byte) (any, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("key is not PEM-encoded")
	}

	if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
		return cert.PublicKey, nil
	}
	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		return pub, nil
	}
	if pub, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return pub, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return publicKeyOf(key)
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return &key.PublicKey, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return &key.PublicKey, nil
	}
	return nil, fmt.Errorf("unsupported PEM key type %q", block.Type)
}

func publicKeyOf(key any) (any, error) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey, nil
	case *ecdsa.PrivateKey:
		return &k.PublicKey, nil
	default:
		return nil, fmt.Errorf("unsupported private key type %T", key)
	}
}
