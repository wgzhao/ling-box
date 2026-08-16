package ssl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

// sctOID is the Certificate Transparency SCT list extension OID, used as an
// unknown extension in test certificates.
var sctOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 2}

// policyInformation mirrors the ASN.1 structure of a PolicyInformation entry.
type policyInformation struct {
	PolicyIdentifier asn1.ObjectIdentifier
}

// certPolicyOID is the CA/Browser Forum domain-validated policy OID.
var certPolicyOID = asn1.ObjectIdentifier{2, 23, 140, 1, 2, 1}

func testCertTemplate(subject, commonName string, notBefore, notAfter time.Time) *x509.Certificate {
	// Go's CreateCertificate does not emit certificatePolicies from the
	// template, so build the extension manually.
	policyValue, err := asn1.Marshal([]policyInformation{{PolicyIdentifier: certPolicyOID}})
	if err != nil {
		panic(err)
	}
	return &x509.Certificate{
		SerialNumber: big.NewInt(0x12345), // odd-length hex: tests serial padding
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{subject + " Org"},
			Country:      []string{"CN"},
			Locality:     []string{"Beijing"},
		},
		NotBefore:          notBefore,
		NotAfter:           notAfter,
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:           []string{"example.com", "www.example.com"},
		IPAddresses:        []net.IP{net.ParseIP("10.0.0.1")},
		EmailAddresses:     []string{"admin@example.com"},
		SubjectKeyId:       []byte{0xde, 0xad, 0xbe, 0xef},
		BasicConstraintsValid: true,
		ExtraExtensions: []pkix.Extension{
			{Id: sctOID, Critical: true, Value: []byte{0x04, 0x00}},
			{Id: asn1.ObjectIdentifier{2, 5, 29, 32}, Value: policyValue},
		},
	}
}

// testAIAExtension builds an Authority Information Access extension pointing
// at an OCSP responder, so parseCRLURLs/OCSPServer can be exercised.
func testAIAExtension(t *testing.T) pkix.Extension {
	t.Helper()
	raw := asn1.RawValue{
		Class:  asn1.ClassContextSpecific,
		Tag:    6, // GeneralName: uniformResourceIdentifier
		Bytes:  []byte("http://ocsp.example.com"),
	}
	desc := accessDescription{Method: asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 48, 1}, Location: raw}
	value, err := asn1.Marshal([]accessDescription{desc})
	if err != nil {
		t.Fatalf("marshal AIA: %v", err)
	}
	return pkix.Extension{Id: asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 1, 1}, Value: value}
}

func createSelfSigned(t *testing.T, tmpl *x509.Certificate, key any) []byte {
	t.Helper()
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, publicKeyOf(t, key), key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return der
}

func publicKeyOf(t *testing.T, key any) any {
	t.Helper()
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	}
	t.Fatalf("unsupported key type %T", key)
	return nil
}

func pemEncode(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestParseSelfSignedPEM(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	now := time.Now()
	tmpl := testCertTemplate("Example Co.", "example.com", now.Add(-24*time.Hour), now.Add(365*24*time.Hour))
	tmpl.ExtraExtensions = append(tmpl.ExtraExtensions, testAIAExtension(t))
	data := pemEncode(createSelfSigned(t, tmpl, key))

	certs, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(certs) != 1 {
		t.Fatalf("got %d certs, want 1", len(certs))
	}
	c := certs[0]

	if c.Index != 1 {
		t.Errorf("Index = %d, want 1", c.Index)
	}
	if c.Version != 3 {
		t.Errorf("Version = %d, want 3 (extensions promote to v3)", c.Version)
	}
	// serial 0x12345 = "12345" (odd length) must be padded: 01:23:45
	if c.SerialNumber != "01:23:45" {
		t.Errorf("SerialNumber = %q, want %q", c.SerialNumber, "01:23:45")
	}
	if !strings.Contains(c.Subject, "CN=example.com") || !strings.Contains(c.Subject, "O=Example Co. Org") || !strings.Contains(c.Subject, "C=CN") {
		t.Errorf("Subject = %q, missing expected attributes", c.Subject)
	}
	if c.Issuer != c.Subject {
		t.Errorf("self-signed issuer = %q, want %q", c.Issuer, c.Subject)
	}
	if c.SignatureAlgorithm != "SHA256-RSA" {
		t.Errorf("SignatureAlgorithm = %q, want SHA256-RSA", c.SignatureAlgorithm)
	}
	if c.PublicKeyAlgorithm != "RSA" || c.PublicKeyBits != 2048 {
		t.Errorf("PublicKey = %s %d bits, want RSA 2048", c.PublicKeyAlgorithm, c.PublicKeyBits)
	}
	if !strings.Contains(c.PublicKeyPEM, "-----BEGIN PUBLIC KEY-----") {
		t.Errorf("PublicKeyPEM missing header: %q", c.PublicKeyPEM)
	}
	if c.KeyUsage != "digitalSignature, keyEncipherment" {
		t.Errorf("KeyUsage = %q", c.KeyUsage)
	}
	if c.ExtKeyUsage != "serverAuth, clientAuth" {
		t.Errorf("ExtKeyUsage = %q", c.ExtKeyUsage)
	}
	if c.IsCA {
		t.Error("IsCA = true, want false")
	}
	sans := strings.Join(c.SANs, " ")
	for _, want := range []string{"DNS:example.com", "DNS:www.example.com", "IP:10.0.0.1", "email:admin@example.com"} {
		if !strings.Contains(sans, want) {
			t.Errorf("SANs %q missing %q", c.SANs, want)
		}
	}
	if c.SubjectKeyID != "DE:AD:BE:EF" {
		t.Errorf("SubjectKeyID = %q, want DE:AD:BE:EF", c.SubjectKeyID)
	}
	if len(c.OCSPURLs) != 1 || c.OCSPURLs[0] != "http://ocsp.example.com" {
		t.Errorf("OCSPURLs = %v", c.OCSPURLs)
	}
	if len(c.PolicyIDs) != 1 || c.PolicyIDs[0] != "2.23.140.1.2.1" {
		t.Errorf("PolicyIDs = %v", c.PolicyIDs)
	}
	found := false
	for _, ext := range c.UnknownExtensions {
		if strings.HasPrefix(ext, sctOID.String()) && strings.Contains(ext, "critical") {
			found = true
		}
	}
	if !found {
		t.Errorf("UnknownExtensions = %v, missing critical SCT OID", c.UnknownExtensions)
	}
	if c.Status(now) != "valid" {
		t.Errorf("Status = %q, want valid", c.Status(now))
	}
}

func TestParseDER(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	now := time.Now()
	tmpl := testCertTemplate("DER Co.", "der.example.com", now.Add(-time.Hour), now.Add(48*time.Hour))
	der := createSelfSigned(t, tmpl, key)

	certs, err := Parse(der) // raw DER, no PEM armor
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(certs) != 1 || !strings.Contains(certs[0].Subject, "CN=der.example.com") {
		t.Fatalf("got %d certs, subject %q", len(certs), certs[0].Subject)
	}
}

func TestParseChain(t *testing.T) {
	now := time.Now()
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate root key: %v", err)
	}
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}

	rootTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Root CA", Organization: []string{"Test CA Org"}},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTmpl, rootTmpl, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	leafTmpl := testCertTemplate("Leaf Co.", "leaf.example.com", now.Add(-time.Hour), now.Add(24*time.Hour))
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, rootTmpl, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}

	// Root first, leaf second — like a fullchain.pem bundle.
	bundle := append(pemEncode(rootDER), pemEncode(leafDER)...)
	certs, err := Parse(bundle)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(certs) != 2 {
		t.Fatalf("got %d certs, want 2", len(certs))
	}
	if certs[0].Index != 1 || certs[1].Index != 2 {
		t.Errorf("Indexes = %d, %d; want 1, 2", certs[0].Index, certs[1].Index)
	}
	if !certs[0].IsCA {
		t.Error("root IsCA = false, want true")
	}
	if certs[0].KeyUsage != "certSign, crlSign" {
		t.Errorf("root KeyUsage = %q", certs[0].KeyUsage)
	}
	if certs[1].IsCA {
		t.Error("leaf IsCA = true, want false")
	}
	// The leaf was issued by the root, so issuer must match the root subject.
	if certs[1].Issuer != certs[0].Subject {
		t.Errorf("leaf issuer %q != root subject %q", certs[1].Issuer, certs[0].Subject)
	}
}

func TestParseECDSA(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	now := time.Now()
	tmpl := testCertTemplate("EC Co.", "ec.example.com", now.Add(-time.Hour), now.Add(24*time.Hour))
	data := pemEncode(createSelfSigned(t, tmpl, key))

	certs, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c := certs[0]
	if c.PublicKeyAlgorithm != "ECDSA P-256" || c.PublicKeyBits != 256 {
		t.Errorf("PublicKey = %s %d bits, want ECDSA P-256 256", c.PublicKeyAlgorithm, c.PublicKeyBits)
	}
	if c.SignatureAlgorithm != "ECDSA-SHA256" {
		t.Errorf("SignatureAlgorithm = %q, want ECDSA-SHA256", c.SignatureAlgorithm)
	}
}

func TestParseNoCertificate(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	data := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if _, err := Parse(data); err == nil {
		t.Fatal("Parse succeeded, want error for PEM without certificates")
	}
}

func TestParseInvalid(t *testing.T) {
	for _, data := range [][]byte{
		[]byte("this is not a certificate"),
		[]byte("-----BEGIN CERTIFICATE-----\nnot-base64!\n-----END CERTIFICATE-----"),
	} {
		if _, err := Parse(data); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", string(data))
		}
	}
}

func TestSecurityBits(t *testing.T) {
	tests := []struct {
		algo string
		bits int
		want int
	}{
		{"RSA", 1024, 80},
		{"RSA", 2048, 112},
		{"RSA", 3072, 128},
		{"RSA", 7680, 192},
		{"RSA", 15360, 256},
		{"ECDSA P-256", 256, 128},
		{"ECDSA P-384", 384, 192},
		{"ECDSA P-521", 521, 260},
		{"Ed25519", 256, 128},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-%d", tt.algo, tt.bits), func(t *testing.T) {
			c := &Certificate{PublicKeyAlgorithm: tt.algo, PublicKeyBits: tt.bits}
			if got := c.SecurityBits(); got != tt.want {
				t.Errorf("SecurityBits = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStatus(t *testing.T) {
	now := time.Now()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tests := []struct {
		name      string
		notBefore time.Time
		notAfter  time.Time
		want      string
	}{
		{"valid", now.Add(-24 * time.Hour), now.Add(24 * time.Hour), "valid"},
		{"expired", now.Add(-48 * time.Hour), now.Add(-24 * time.Hour), "expired"},
		{"not yet valid", now.Add(24 * time.Hour), now.Add(48 * time.Hour), "not yet valid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := testCertTemplate("Status Co.", "status.example.com", tt.notBefore, tt.notAfter)
			data := pemEncode(createSelfSigned(t, tmpl, key))
			certs, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := certs[0].Status(now); got != tt.want {
				t.Errorf("Status = %q, want %q", got, tt.want)
			}
		})
	}
}
