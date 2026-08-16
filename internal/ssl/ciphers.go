package ssl

import (
	"crypto/tls"
	"fmt"
	"strings"
)

// Cipher suite security ratings.
const (
	RatingGood     = "GOOD"
	RatingWeak     = "WEAK"
	RatingInsecure = "INSECURE"
)

// CipherSuite describes one TLS 1.0-1.2 cipher suite.
type CipherSuite struct {
	ID              uint16
	Name            string // IANA name, e.g. "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
	Code            string // "0xC02F"
	KeyBits         int    // symmetric encryption strength in bits
	KeyExchange     string // base key exchange: "RSA", "ECDHE", "DHE", "PSK", ...
	KeyDetail       string // filled during probing: "ECDH secp256r1 (eq. 3072-bit RSA)", "DH 2048 bits"
	ForwardSecrecy  bool
	Rating          string
}

// tls13SuiteIDs are the cipher suites Go's client supports for TLS 1.3.
// TLS 1.3 suites are not individually negotiable, so they are reported
// as a group rather than probed one by one.
var tls13SuiteIDs = []uint16{
	tls.TLS_AES_128_GCM_SHA256,
	tls.TLS_AES_256_GCM_SHA384,
	tls.TLS_CHACHA20_POLY1305_SHA256,
}

// cipherSuites returns every TLS 1.0-1.2 cipher suite implemented by Go,
// suitable for probing. Insecure suites (RC4, 3DES, anon, ...) are included
// on purpose: those are exactly the ones a scanner must detect.
func cipherSuites() []*CipherSuite {
	seen := make(map[uint16]bool)
	var out []*CipherSuite
	for _, s := range tls.CipherSuites() {
		out = append(out, describeSuite(s.ID, s.Name))
		seen[s.ID] = true
	}
	for _, s := range tls.InsecureCipherSuites() {
		if !seen[s.ID] {
			out = append(out, describeSuite(s.ID, s.Name))
		}
	}
	return out
}

// describeSuite derives the key exchange, encryption strength, forward
// secrecy flag and security rating of a suite from its IANA name.
func describeSuite(id uint16, name string) *CipherSuite {
	s := &CipherSuite{ID: id, Name: name, Code: fmt.Sprintf("0x%04X", id)}
	n := strings.TrimPrefix(name, "TLS_")

	// Split into key exchange part and cipher part, e.g.
	//   TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 -> ECDHE_RSA | AES_128_GCM | SHA256
	// TLS 1.3 style names (TLS_AES_128_GCM_SHA256) have no key exchange part.
	var kxPart, cipherPart, mac string
	if sep := strings.Index(n, "_WITH_"); sep >= 0 {
		kxPart = n[:sep]
		rest := n[sep+len("_WITH_"):]
		if i := strings.LastIndex(rest, "_"); i > 0 {
			mac = rest[i+1:]
			cipherPart = rest[:i]
		} else {
			cipherPart = rest
		}
	} else {
		cipherPart = n
	}

	// Key exchange and forward secrecy.
	switch {
	case kxPart == "":
		// TLS 1.3 style name (TLS_AES_128_GCM_SHA256): key exchange is
		// always ephemeral, so the suite has forward secrecy.
		s.KeyExchange, s.ForwardSecrecy = "TLS1.3", true
	case strings.HasPrefix(kxPart, "ECDHE"):
		s.KeyExchange, s.ForwardSecrecy = "ECDHE", true
	case strings.HasPrefix(kxPart, "DHE"):
		s.KeyExchange, s.ForwardSecrecy = "DHE", true
	case strings.HasPrefix(kxPart, "ECDH_"), strings.HasPrefix(kxPart, "DH_"):
		s.KeyExchange = "ECDH" // static (anonymous or fixed) key exchange
	case strings.Contains(kxPart, "PSK"):
		s.KeyExchange = kxPart
	case strings.HasPrefix(kxPart, "SRP"):
		s.KeyExchange = "SRP"
	case strings.HasPrefix(kxPart, "KRB5"):
		s.KeyExchange = "KRB5"
	case strings.HasPrefix(kxPart, "RSA"):
		s.KeyExchange = "RSA"
	default:
		s.KeyExchange = kxPart
	}

	// Symmetric encryption strength.
	switch {
	case strings.Contains(cipherPart, "RC4"):
		s.KeyBits = 128
	case strings.HasPrefix(cipherPart, "3DES"):
		s.KeyBits = 168
	case strings.HasPrefix(cipherPart, "DES_"):
		s.KeyBits = 56
	case strings.Contains(cipherPart, "AES_128"), strings.Contains(cipherPart, "CAMELLIA_128"),
		strings.Contains(cipherPart, "ARIA_128"), strings.Contains(cipherPart, "SEED"):
		s.KeyBits = 128
	case strings.Contains(cipherPart, "AES_256"), strings.Contains(cipherPart, "CAMELLIA_256"),
		strings.Contains(cipherPart, "ARIA_256"), strings.Contains(cipherPart, "CHACHA20"):
		s.KeyBits = 256
	}

	// Security rating.
	aead := strings.Contains(cipherPart, "GCM") || strings.Contains(cipherPart, "CCM") ||
		strings.Contains(cipherPart, "CHACHA20")
	switch {
	case strings.HasPrefix(cipherPart, "3DES"):
		s.Rating = RatingWeak
	case strings.Contains(cipherPart, "NULL") || strings.Contains(cipherPart, "RC4") ||
		strings.Contains(cipherPart, "DES_") || mac == "MD5" ||
		strings.Contains(n, "EXPORT") || strings.Contains(n, "ANON"):
		s.Rating = RatingInsecure
	case aead:
		s.Rating = RatingGood
	case strings.Contains(cipherPart, "CBC") && mac == "SHA":
		s.Rating = RatingWeak
	case strings.Contains(n, "PSK") || strings.Contains(n, "SRP") || strings.Contains(n, "KRB5"):
		s.Rating = RatingWeak
	default:
		s.Rating = RatingGood
	}
	return s
}
