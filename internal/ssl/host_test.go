package ssl

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// startTLSServer starts an httptest TLS server with the given TLS config.
// httptest generates a self-signed certificate for 127.0.0.1 / ::1.
func startTLSServer(t *testing.T, cfg *tls.Config) (host string, port int, closeFn func()) {
	t.Helper()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.TLS = cfg
	server.StartTLS()
	host, portStr, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		server.Close()
		t.Fatalf("split addr: %v", err)
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		server.Close()
		t.Fatalf("parse port: %v", err)
	}
	return host, port, server.Close
}

func scan(t *testing.T, host string, port int) *HostInfo {
	t.Helper()
	info, err := ScanHost(host, port, 3*time.Second, nil)
	if err != nil {
		t.Fatalf("ScanHost: %v", err)
	}
	return info
}

func protocolByName(info *HostInfo, name string) *Protocol {
	for _, p := range info.Protocols {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func TestScanTLS12Suites(t *testing.T) {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS10,
		MaxVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_RC4_128_SHA,
		},
	}
	host, port, closeFn := startTLSServer(t, cfg)
	defer closeFn()

	info := scan(t, host, port)

	// Only TLS 1.0-1.2 should be supported.
	if !protocolByName(info, "TLSv1.2").Supported {
		t.Fatal("TLSv1.2 should be supported")
	}
	if protocolByName(info, "TLSv1.3").Supported {
		t.Error("TLSv1.3 should not be supported")
	}
	if !protocolByName(info, "TLSv1.1").Supported || !protocolByName(info, "TLSv1.0").Supported {
		t.Error("TLSv1.1/TLSv1.0 should be supported")
	}

	// Exactly the three configured suites must be discovered.
	p12 := protocolByName(info, "TLSv1.2")
	names := make(map[string]bool)
	for _, s := range p12.Suites {
		names[s.Name] = true
	}
	for _, want := range []string{
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		"TLS_RSA_WITH_AES_256_CBC_SHA",
		"TLS_RSA_WITH_RC4_128_SHA",
	} {
		if !names[want] {
			t.Errorf("missing discovered suite %q (got %v)", want, names)
		}
	}
	if len(p12.Suites) != 3 {
		t.Errorf("discovered %d suites, want 3: %v", len(p12.Suites), names)
	}

	// Ratings must reflect the configured suites.
	ratings := map[string]string{}
	for _, s := range p12.Suites {
		ratings[s.Name] = s.Rating
	}
	if ratings["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"] != RatingGood {
		t.Errorf("GCM suite rating = %q, want GOOD", ratings["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"])
	}
	if ratings["TLS_RSA_WITH_AES_256_CBC_SHA"] != RatingWeak {
		t.Errorf("CBC-SHA suite rating = %q, want WEAK", ratings["TLS_RSA_WITH_AES_256_CBC_SHA"])
	}
	if ratings["TLS_RSA_WITH_RC4_128_SHA"] != RatingInsecure {
		t.Errorf("RC4 suite rating = %q, want INSECURE", ratings["TLS_RSA_WITH_RC4_128_SHA"])
	}
}

func TestScanTLS13Only(t *testing.T) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS13}
	host, port, closeFn := startTLSServer(t, cfg)
	defer closeFn()

	info := scan(t, host, port)
	p13 := protocolByName(info, "TLSv1.3")
	if !p13.Supported {
		t.Fatal("TLSv1.3 should be supported")
	}
	if len(p13.Suites) != len(tls13SuiteIDs) {
		t.Errorf("TLS 1.3 suite count = %d, want %d", len(p13.Suites), len(tls13SuiteIDs))
	}
	for _, s := range p13.Suites {
		if s.Rating != RatingGood || !s.ForwardSecrecy {
			t.Errorf("TLS 1.3 suite %s: rating=%q fs=%v, want GOOD/true", s.Name, s.Rating, s.ForwardSecrecy)
		}
	}
	if protocolByName(info, "TLSv1.2").Supported {
		t.Error("TLSv1.2 should not be supported")
	}
}

func TestScanTLS10Only(t *testing.T) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS10}
	host, port, closeFn := startTLSServer(t, cfg)
	defer closeFn()

	info := scan(t, host, port)
	if !protocolByName(info, "TLSv1.0").Supported {
		t.Fatal("TLSv1.0 should be supported")
	}
	for _, name := range []string{"TLSv1.1", "TLSv1.2", "TLSv1.3"} {
		if protocolByName(info, name).Supported {
			t.Errorf("%s should not be supported", name)
		}
	}
}

func TestScanECDHCurveDetail(t *testing.T) {
	cfg := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		MaxVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.CurveP256},
		CipherSuites:     []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
	}
	host, port, closeFn := startTLSServer(t, cfg)
	defer closeFn()

	info := scan(t, host, port)
	p12 := protocolByName(info, "TLSv1.2")
	if len(p12.Suites) != 1 {
		t.Fatalf("discovered %d suites, want 1", len(p12.Suites))
	}
	s := p12.Suites[0]
	if !strings.Contains(s.KeyDetail, "secp256r1") || !strings.Contains(s.KeyDetail, "3072") {
		t.Errorf("KeyDetail = %q, want ECDH secp256r1 (eq. 3072-bit RSA)", s.KeyDetail)
	}
	if !s.ForwardSecrecy {
		t.Error("ECDHE suite should have forward secrecy")
	}
}

func TestScanCertificateAndTrust(t *testing.T) {
	host, port, closeFn := startTLSServer(t, &tls.Config{})
	defer closeFn()

	info := scan(t, host, port)

	if info.Certificate == nil {
		t.Fatal("certificate should be fetched")
	}
	// The httptest cert has no CN; 127.0.0.1 lives in the SAN list.
	sans := strings.Join(info.Certificate.SANs, " ")
	if !strings.Contains(sans, "IP:127.0.0.1") {
		t.Errorf("SANs = %q, want the httptest 127.0.0.1 cert SAN", sans)
	}
	// httptest certs are self-signed, so trust verification must fail.
	if info.Trust == nil || info.Trust.Trusted {
		t.Fatalf("Trust = %+v, want untrusted self-signed cert", info.Trust)
	}
	if !strings.Contains(info.Trust.Reason, "未知颁发机构") {
		t.Errorf("Trust.Reason = %q, want unknown authority", info.Trust.Reason)
	}
	// 127.0.0.1 is a SAN of the httptest cert, so hostname must match.
	if !info.Trust.HostnameOK {
		t.Errorf("HostnameOK = false, want true (SAN has 127.0.0.1): %q", info.Trust.HostnameErr)
	}
	if info.Certificate.Status(time.Now()) != "valid" {
		t.Errorf("certificate should be valid: %q", info.Certificate.Status(time.Now()))
	}
}

func TestScanConnectionRefused(t *testing.T) {
	// Port 1 is essentially never open.
	if _, err := ScanHost("127.0.0.1", 1, 2*time.Second, nil); err == nil {
		t.Fatal("ScanHost should fail for a closed port")
	}
}

func TestScanProgressCallbacks(t *testing.T) {
	host, port, closeFn := startTLSServer(t, &tls.Config{})
	defer closeFn()

	var calls []string
	_, err := ScanHost(host, port, 3*time.Second, func(s string) {
		if s == "" {
			t.Error("empty progress update")
		}
		calls = append(calls, s)
	})
	if err != nil {
		t.Fatalf("ScanHost: %v", err)
	}
	if len(calls) == 0 {
		t.Fatal("no progress callbacks received")
	}
	// The first and last updates should cover connect and trust phases.
	if !strings.Contains(calls[0], "正在连接") {
		t.Errorf("first progress = %q, want connect phase", calls[0])
	}
	if !strings.Contains(calls[len(calls)-1], "验证证书信任") {
		t.Errorf("last progress = %q, want trust verification phase", calls[len(calls)-1])
	}
	// Each protocol should be reported.
	joined := strings.Join(calls, "|")
	for _, proto := range []string{"TLSv1.3", "TLSv1.2", "TLSv1.1", "TLSv1.0"} {
		if !strings.Contains(joined, proto) {
			t.Errorf("progress updates missing protocol %s: %v", proto, calls)
		}
	}
}

func TestDescribeSuiteRules(t *testing.T) {
	tests := []struct {
		id     uint16
		name   string
		bits   int
		kx     string
		fs     bool
		rating string
	}{
		// GOOD + FS
		{0xC02F, "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", 128, "ECDHE", true, RatingGood},
		{0xC030, "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", 256, "ECDHE", true, RatingGood},
		{0x9C, "TLS_DHE_RSA_WITH_AES_128_GCM_SHA256", 128, "DHE", true, RatingGood},
		{0xC023, "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256", 128, "ECDHE", true, RatingGood},
		{0xCCA8, "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256", 256, "ECDHE", true, RatingGood},
		// WEAK
		{0x2F, "TLS_RSA_WITH_AES_128_CBC_SHA", 128, "RSA", false, RatingWeak},
		{0x35, "TLS_RSA_WITH_AES_256_CBC_SHA", 256, "RSA", false, RatingWeak},
		{0xC013, "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA", 128, "ECDHE", true, RatingWeak},
		{0x0A, "TLS_RSA_WITH_3DES_EDE_CBC_SHA", 168, "RSA", false, RatingWeak},
		{0xC027, "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256", 128, "ECDHE", true, RatingGood},
		// INSECURE
		{0x05, "TLS_RSA_WITH_RC4_128_SHA", 128, "RSA", false, RatingInsecure},
		{0xC011, "TLS_ECDHE_RSA_WITH_RC4_128_SHA", 128, "ECDHE", true, RatingInsecure},
		{0x09, "TLS_RSA_WITH_DES_CBC_SHA", 56, "RSA", false, RatingInsecure},
		{0x18, "TLS_DH_anon_WITH_RC4_128_MD5", 128, "ECDH", false, RatingInsecure},
		{0x32, "TLS_RSA_WITH_NULL_SHA", 0, "RSA", false, RatingInsecure},
		// TLS 1.3
		{0x1301, "TLS_AES_128_GCM_SHA256", 128, "TLS1.3", true, RatingGood},
		{0x1302, "TLS_AES_256_GCM_SHA384", 256, "TLS1.3", true, RatingGood},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := describeSuite(tt.id, tt.name)
			if s.KeyBits != tt.bits {
				t.Errorf("KeyBits = %d, want %d", s.KeyBits, tt.bits)
			}
			if s.KeyExchange != tt.kx {
				t.Errorf("KeyExchange = %q, want %q", s.KeyExchange, tt.kx)
			}
			if s.ForwardSecrecy != tt.fs {
				t.Errorf("ForwardSecrecy = %v, want %v", s.ForwardSecrecy, tt.fs)
			}
			if s.Rating != tt.rating {
				t.Errorf("Rating = %q, want %q", s.Rating, tt.rating)
			}
		})
	}
}

func TestParseSKE(t *testing.T) {
	// ECDHE with secp256r1 (named curve 0x0017 = 23).
	body := []byte{3, 0, 23, 5, 1, 2, 3, 4, 5}
	if got := parseSKE(body); got != "ECDH secp256r1 (eq. 3072-bit RSA)" {
		t.Errorf("ECDHE parse = %q", got)
	}
	// DHE with a 2048-bit prime (256 bytes).
	dh := make([]byte, 2+256)
	dh[0], dh[1] = 0x01, 0x00
	if got := parseSKE(dh); got != "DH 2048 bits" {
		t.Errorf("DHE parse = %q", got)
	}
	// Too short.
	if got := parseSKE([]byte{1, 2}); got != "" {
		t.Errorf("short body parse = %q, want empty", got)
	}
}
