// Package ssl provides X.509 certificate inspection utilities.
package ssl

import (
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Certificate is a parsed X.509 certificate with display-ready fields.
type Certificate struct {
	Index              int       // 1-based position within the input bundle
	Version            int       // 1-indexed X.509 version (1 = v1, 3 = v3)
	SerialNumber       string    // colon-separated uppercase hex
	SignatureAlgorithm string    // e.g. "SHA256-RSA"
	Issuer             string    // one-line distinguished name
	Subject            string    // one-line distinguished name
	NotBefore          time.Time
	NotAfter           time.Time
	PublicKeyAlgorithm string // e.g. "RSA", "ECDSA P-256", "Ed25519"
	PublicKeyBits      int    // key strength in bits
	PublicKeyPEM       string // PEM-encoded SPKI public key
	KeyUsage           string // comma-separated usage names, empty if absent
	ExtKeyUsage        string // comma-separated names/OIDs, empty if absent
	IsCA               bool
	MaxPathLen         int // -1 if no pathLen constraint
	SANs               []string
	SubjectKeyID       string
	AuthorityKeyID     string
	OCSPURLs           []string
	IssuerURLs         []string
	CRLURLs            []string
	PolicyIDs          []string
	UnknownExtensions  []string // "OID (critical)" for extensions not parsed
}

// Parse parses PEM data (possibly a bundle of multiple certificates) or
// a single DER-encoded certificate into one or more Certificates.
func Parse(data []byte) ([]*Certificate, error) {
	// Try PEM first. The loop keeps parsing blocks so bundles (cert chains,
	// e.g. fullchain.pem) yield every certificate.
	var certs []*Certificate
	rest := data
	foundPEM := false
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		foundPEM = true
		if block.Type != "CERTIFICATE" {
			continue
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse certificate in PEM block: %w", err)
		}
		certs = append(certs, fromX509(c))
	}

	if foundPEM {
		if len(certs) == 0 {
			return nil, fmt.Errorf("PEM data contains no certificates")
		}
		for i, c := range certs {
			c.Index = i + 1
		}
		return certs, nil
	}

	// Not PEM, try raw DER.
	c, err := x509.ParseCertificate(data)
	if err != nil {
		return nil, fmt.Errorf("not valid PEM or DER certificate data: %w", err)
	}
	certs = append(certs, fromX509(c))
	certs[0].Index = 1
	return certs, nil
}

// Status reports the certificate validity state relative to now:
// "valid", "expired", or "not yet valid".
func (c *Certificate) Status(now time.Time) string {
	switch {
	case now.Before(c.NotBefore):
		return "not yet valid"
	case now.After(c.NotAfter):
		return "expired"
	default:
		return "valid"
	}
}

// SecurityBits approximates the effective security strength in bits
// (NIST SP 800-57 table 2): a 256-bit ECC key has roughly the same
// strength as a 3072-bit RSA key, so raw key sizes are not comparable.
func (c *Certificate) SecurityBits() int {
	switch {
	case strings.HasPrefix(c.PublicKeyAlgorithm, "ECDSA"), strings.HasPrefix(c.PublicKeyAlgorithm, "ECDH"):
		return c.PublicKeyBits / 2
	case c.PublicKeyAlgorithm == "Ed25519":
		return 128
	case c.PublicKeyAlgorithm == "RSA":
		switch {
		case c.PublicKeyBits >= 15360:
			return 256
		case c.PublicKeyBits >= 7680:
			return 192
		case c.PublicKeyBits >= 3072:
			return 128
		case c.PublicKeyBits >= 2048:
			return 112
		case c.PublicKeyBits >= 1024:
			return 80
		}
	}
	return 0
}

// fromX509 converts an x509.Certificate into the display-ready form.
func fromX509(c *x509.Certificate) *Certificate {
	var bits int
	algo := publicKeyDescription(c.PublicKey, &bits)
	return &Certificate{
		Version:            c.Version,
		SerialNumber:       formatSerial(c.SerialNumber),
		SignatureAlgorithm: c.SignatureAlgorithm.String(),
		Issuer:             formatName(&c.Issuer),
		Subject:            formatName(&c.Subject),
		NotBefore:          c.NotBefore,
		NotAfter:           c.NotAfter,
		PublicKeyAlgorithm: algo,
		PublicKeyBits:      bits,
		PublicKeyPEM:       publicKeyPEM(c.PublicKey),
		KeyUsage:           formatKeyUsage(c.KeyUsage),
		ExtKeyUsage:        formatExtKeyUsage(c),
		IsCA:               c.IsCA,
		MaxPathLen:         c.MaxPathLen,
		SANs:               formatSANs(c),
		SubjectKeyID:       formatKeyID(c.SubjectKeyId),
		AuthorityKeyID:     formatKeyID(c.AuthorityKeyId),
		OCSPURLs:           c.OCSPServer,
		IssuerURLs:         c.IssuingCertificateURL,
		CRLURLs:            parseCRLURLs(c),
		PolicyIDs:          formatOIDs(c.PolicyIdentifiers),
		UnknownExtensions:  unknownExtensions(c),
	}
}

// dnLabels maps commonly used DN attribute OIDs to short labels.
var dnLabels = map[string]string{
	"2.5.4.3":    "CN",
	"2.5.4.4":    "SN",
	"2.5.4.5":    "serialNumber",
	"2.5.4.6":    "C",
	"2.5.4.7":    "L",
	"2.5.4.8":    "ST",
	"2.5.4.9":    "street",
	"2.5.4.10":   "O",
	"2.5.4.11":   "OU",
	"2.5.4.12":   "title",
	"2.5.4.17":   "postalCode",
	"2.5.4.42":   "GN",
	"2.5.4.43":   "initials",
	"0.9.2342.19200300.100.1.1":  "UID",
	"0.9.2342.19200300.100.1.25": "DC",
	"1.2.840.113549.1.9.1":       "emailAddress",
	"1.2.840.113549.1.9.2":       "unstructuredName",
	"1.3.6.1.4.1.311.60.2.1.2":   "jurisdictionST",
	"1.3.6.1.4.1.311.60.2.1.3":   "jurisdictionC",
}

// formatName renders a distinguished name as "C=xx, O=xx, CN=xx" using the
// original attribute order from the certificate.
func formatName(n *pkix.Name) string {
	parts := make([]string, 0, len(n.Names))
	for _, atv := range n.Names {
		label := dnLabels[atv.Type.String()]
		if label == "" {
			label = atv.Type.String()
		}
		parts = append(parts, label+"="+dnValue(atv.Value))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

// dnValue renders a single DN attribute value.
func dnValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.([]byte); ok {
		return hex.EncodeToString(b)
	}
	return fmt.Sprint(v)
}

// formatSerial renders a serial number as colon-separated uppercase hex.
func formatSerial(n *big.Int) string {
	s := strings.ToUpper(n.Text(16))
	if len(s)%2 == 1 {
		s = "0" + s
	}
	var b strings.Builder
	for i := 0; i < len(s); i += 2 {
		if i > 0 {
			b.WriteByte(':')
		}
		b.WriteString(s[i : i+2])
	}
	return b.String()
}

// formatKeyID renders a key identifier (SKI/AKI) as colon-separated hex.
func formatKeyID(id []byte) string {
	if len(id) == 0 {
		return ""
	}
	s := strings.ToUpper(hex.EncodeToString(id))
	var b strings.Builder
	for i := 0; i < len(s); i += 2 {
		if i > 0 {
			b.WriteByte(':')
		}
		b.WriteString(s[i : i+2])
	}
	return b.String()
}

// publicKeyDescription returns a human description of the public key, e.g.
// "RSA", "ECDSA P-256" or "Ed25519", and fills bits with the key strength.
func publicKeyDescription(pub crypto.PublicKey, bits *int) string {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		*bits = k.N.BitLen()
		return "RSA"
	case *ecdsa.PublicKey:
		*bits = k.Curve.Params().BitSize
		return "ECDSA " + k.Curve.Params().Name
	case ed25519.PublicKey:
		*bits = 256
		return "Ed25519"
	case *dsa.PublicKey:
		*bits = k.Q.BitLen()
		return "DSA"
	default:
		return fmt.Sprintf("%T", pub)
	}
}

// publicKeyPEM returns the public key in PEM (SPKI) form.
func publicKeyPEM(pub crypto.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return ""
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

var keyUsageBits = []struct {
	bit  x509.KeyUsage
	name string
}{
	{x509.KeyUsageDigitalSignature, "digitalSignature"},
	{x509.KeyUsageContentCommitment, "contentCommitment"},
	{x509.KeyUsageKeyEncipherment, "keyEncipherment"},
	{x509.KeyUsageDataEncipherment, "dataEncipherment"},
	{x509.KeyUsageKeyAgreement, "keyAgreement"},
	{x509.KeyUsageCertSign, "certSign"},
	{x509.KeyUsageCRLSign, "crlSign"},
	{x509.KeyUsageEncipherOnly, "encipherOnly"},
	{x509.KeyUsageDecipherOnly, "decipherOnly"},
}

// formatKeyUsage renders the KeyUsage bitfield as comma-separated names.
func formatKeyUsage(u x509.KeyUsage) string {
	var names []string
	for _, ku := range keyUsageBits {
		if u&ku.bit != 0 {
			names = append(names, ku.name)
		}
	}
	return strings.Join(names, ", ")
}

// formatExtKeyUsage renders the Extended Key Usage, appending any unknown
// usage OIDs.
func formatExtKeyUsage(c *x509.Certificate) string {
	var names []string
	for _, eku := range c.ExtKeyUsage {
		names = append(names, eku.String())
	}
	for _, oid := range c.UnknownExtKeyUsage {
		names = append(names, oid.String())
	}
	return strings.Join(names, ", ")
}

// formatSANs renders subject alternative names as "DNS:x", "IP:x",
// "email:x" and "URI:x" entries.
func formatSANs(c *x509.Certificate) []string {
	var sans []string
	for _, d := range c.DNSNames {
		sans = append(sans, "DNS:"+d)
	}
	for _, ip := range c.IPAddresses {
		sans = append(sans, "IP:"+ip.String())
	}
	for _, e := range c.EmailAddresses {
		sans = append(sans, "email:"+e)
	}
	for _, u := range c.URIs {
		sans = append(sans, "URI:"+u.String())
	}
	return sans
}

func formatOIDs(oids []asn1.ObjectIdentifier) []string {
	out := make([]string, 0, len(oids))
	for _, oid := range oids {
		out = append(out, oid.String())
	}
	return out
}

// accessDescription is the ASN.1 structure of an Authority Information
// Access entry (RFC 5280 §4.2.2.1).
type accessDescription struct {
	Method   asn1.ObjectIdentifier
	Location asn1.RawValue
}

// knownExtensionOIDs lists extensions this package parses and presents
// in dedicated sections; everything else is listed as "unknown".
var knownExtensionOIDs = map[string]bool{
	"2.5.29.14":  true, // subjectKeyIdentifier
	"2.5.29.15":  true, // keyUsage
	"2.5.29.17":  true, // subjectAltName
	"2.5.29.19":  true, // basicConstraints
	"2.5.29.31":  true, // CRLDistributionPoints
	"2.5.29.32":  true, // certificatePolicies
	"2.5.29.35":  true, // authorityKeyIdentifier
	"2.5.29.37":  true, // extKeyUsage
	"1.3.6.1.5.5.7.1.1": true, // authorityInfoAccess
}

// unknownExtensions returns extensions not covered by the parsed sections,
// e.g. Certificate Transparency SCT list (1.3.6.1.4.1.11129.2.4.2).
func unknownExtensions(c *x509.Certificate) []string {
	var out []string
	for _, ext := range c.Extensions {
		if knownExtensionOIDs[ext.Id.String()] {
			continue
		}
		s := ext.Id.String()
		if ext.Critical {
			s += " (critical)"
		}
		out = append(out, s)
	}
	return out
}

// parseCRLURLs extracts URL distribution points from the CRLDistributionPoints
// extension (RFC 5280 §4.2.1.13), handling the common fullName form.
func parseCRLURLs(c *x509.Certificate) []string {
	var urls []string
	for _, ext := range c.Extensions {
		if ext.Id.String() != "2.5.29.31" {
			continue
		}
		var dps []struct {
			DistributionPoint asn1.RawValue `asn1:"optional,tag:0"`
		}
		if _, err := asn1.Unmarshal(ext.Value, &dps); err != nil {
			continue
		}
		for _, dp := range dps {
			if dp.DistributionPoint.Tag != 0 {
				continue // only the fullName form carries URLs
			}
			var names []asn1.RawValue
			if _, err := asn1.Unmarshal(dp.DistributionPoint.FullBytes, &names); err != nil {
				continue
			}
			for _, n := range names {
				if n.Class == asn1.ClassContextSpecific && n.Tag == 6 {
					urls = append(urls, string(n.Bytes))
				}
			}
		}
	}
	return urls
}
