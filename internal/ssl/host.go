package ssl

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"time"
)

// Protocol reports whether a TLS version is supported and which cipher
// suites the server accepts on it.
type Protocol struct {
	Name      string // "TLSv1.3"
	Version   uint16
	Supported bool
	Suites    []*CipherSuite // discovered in server preference order
}

// TrustResult reports the outcome of verifying the presented certificate
// against the system root store.
type TrustResult struct {
	Trusted     bool
	Reason      string // empty when trusted
	HostnameOK  bool
	HostnameErr string
}

// HostInfo is the result of a full TLS scan of one host.
type HostInfo struct {
	Host        string
	Port        int
	RemoteIP    string
	Protocols   []*Protocol
	Certificate *Certificate
	Trust       *TrustResult
}

// ScanHost connects to host:port and probes supported TLS protocols, the
// cipher suites accepted per protocol, and the presented certificate.
// progress, if non-nil, receives human-readable status updates.
func ScanHost(host string, port int, timeout time.Duration, progress func(string)) (*HostInfo, error) {
	info := &HostInfo{Host: host, Port: port}
	report := func(s string) {
		if progress != nil {
			progress(s)
		}
	}

	report(fmt.Sprintf("正在连接 %s:%d ...", host, port))
	for _, v := range []struct {
		name string
		ver  uint16
	}{
		{"TLSv1.3", tls.VersionTLS13},
		{"TLSv1.2", tls.VersionTLS12},
		{"TLSv1.1", tls.VersionTLS11},
		{"TLSv1.0", tls.VersionTLS10},
	} {
		info.Protocols = append(info.Protocols, probeProtocol(host, port, v.ver, timeout, progress))
	}

	// Fetch the certificate over the highest supported protocol.
	version := uint16(0)
	for _, p := range info.Protocols {
		if p.Supported {
			version = p.Version
			break
		}
	}
	if version == 0 {
		return nil, fmt.Errorf("无法与 %s:%d 建立 TLS 连接 (所有协议版本均不受支持)", host, port)
	}
	report("获取证书 ...")
	leaf, cert, ip, err := fetchCertificate(host, port, version, timeout)
	if err != nil {
		return nil, fmt.Errorf("获取证书失败: %w", err)
	}
	info.Certificate = cert
	info.RemoteIP = ip
	report("验证证书信任 ...")
	info.Trust = verifyTrust(host, port, leaf, timeout)
	return info, nil
}

// probeProtocol discovers whether the server supports version and which
// cipher suites it accepts. The server picks its preferred suite from the
// offered list in the ServerHello; after each success the chosen suite is
// removed from the list and the handshake repeated, so the whole set is
// enumerated in server preference order.
func probeProtocol(host string, port int, version uint16, timeout time.Duration, progress func(string)) *Protocol {
	p := &Protocol{Name: versionName(version), Version: version}
	report := func(s string) {
		if progress != nil {
			progress(s)
		}
	}

	if version == tls.VersionTLS13 {
		// TLS 1.3 suites are not negotiable per suite; a single handshake
		// tells us the protocol is supported.
		report("探测 " + p.Name + " ...")
		conn, _, _, err := probeHandshake(host, port, version, nil, timeout)
		if err == nil {
			p.Supported = true
			for _, id := range tls13SuiteIDs {
				p.Suites = append(p.Suites, describeSuite(id, tls.CipherSuiteName(id)))
			}
			conn.Close()
		}
		if p.Supported {
			report(fmt.Sprintf("%s: 支持 (%d 个套件)", p.Name, len(p.Suites)))
		} else {
			report(p.Name + ": 不支持")
		}
		return p
	}

	remaining := cipherSuites()
	for len(remaining) > 0 {
		ids := make([]uint16, len(remaining))
		for i, s := range remaining {
			ids[i] = s.ID
		}
		report(fmt.Sprintf("探测 %s ... 已发现 %d 个套件", p.Name, len(p.Suites)))
		conn, state, kxDetail, err := probeHandshake(host, port, version, ids, timeout)
		if err != nil {
			if conn != nil {
				conn.Close()
			}
			break
		}
		found := false
		for i, s := range remaining {
			if s.ID == state.CipherSuite {
				s.KeyDetail = kxDetail
				p.Suites = append(p.Suites, s)
				remaining = slices.Delete(remaining, i, i+1)
				found = true
				break
			}
		}
		conn.Close()
		if !found {
			break // server picked a suite we did not offer; avoid an endless loop
		}
	}
	p.Supported = len(p.Suites) > 0
	if p.Supported {
		report(fmt.Sprintf("%s: 支持 (%d 个套件)", p.Name, len(p.Suites)))
	} else {
		report(p.Name + ": 不支持")
	}
	return p
}

// probeHandshake dials the host and performs one TLS handshake restricted
// to version (and suiteIDs for TLS <= 1.2). It returns the live connection
// (caller closes it), the negotiated connection state, and the key exchange
// detail parsed from the ServerKeyExchange ("ECDH secp256r1 (eq. 3072-bit
// RSA)" for ECDHE, "DH 2048 bits" for DHE, empty otherwise).
func probeHandshake(host string, port int, version uint16, suiteIDs []uint16, timeout time.Duration) (*tls.Conn, tls.ConnectionState, string, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	raw, err := (&net.Dialer{Timeout: timeout}).Dial("tcp", addr)
	if err != nil {
		return nil, tls.ConnectionState{}, "", err
	}
	rec := &recordConn{Conn: raw}
	conn := tls.Client(rec, &tls.Config{
		ServerName:         host, // Go sends no SNI for IP addresses automatically
		MinVersion:         version,
		MaxVersion:         version,
		CipherSuites:       suiteIDs,
		InsecureSkipVerify: true, // certificate trust is checked separately
	})
	conn.SetDeadline(time.Now().Add(timeout))
	if err := conn.Handshake(); err != nil {
		raw.Close()
		return nil, tls.ConnectionState{}, "", err
	}
	state := conn.ConnectionState()
	return conn, state, parseKeyExchange(rec.buf.Bytes()), nil
}

// fetchCertificate performs one handshake and returns the leaf certificate
// in both raw and display form, plus the resolved remote IP.
func fetchCertificate(host string, port int, version uint16, timeout time.Duration) (*x509.Certificate, *Certificate, string, error) {
	conn, state, _, err := probeHandshake(host, port, version, nil, timeout)
	if err != nil {
		return nil, nil, "", err
	}
	defer conn.Close()
	if len(state.PeerCertificates) == 0 {
		return nil, nil, "", fmt.Errorf("服务器未返回证书")
	}
	leaf := state.PeerCertificates[0]
	certs, err := Parse(leaf.Raw)
	if err != nil {
		return nil, nil, "", err
	}
	ip := conn.RemoteAddr().String()
	if h, _, err := net.SplitHostPort(ip); err == nil {
		ip = h
	}
	return leaf, certs[0], ip, nil
}

// verifyTrust checks the presented certificate against the system root
// store and against the target hostname.
func verifyTrust(host string, port int, leaf *x509.Certificate, timeout time.Duration) *TrustResult {
	r := &TrustResult{HostnameOK: true}
	if err := leaf.VerifyHostname(host); err != nil {
		r.HostnameOK = false
		r.HostnameErr = fmt.Sprintf("证书主机名与 %q 不匹配 (SAN: %v)", host, leaf.DNSNames)
	}

	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		r.Reason = "无法加载系统根证书"
		return r
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	dial := func(minVersion uint16) error {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", addr, &tls.Config{
			RootCAs:   roots,
			ServerName: host,
			MinVersion: minVersion,
		})
		if err == nil {
			conn.Close()
		}
		return err
	}
	err = dial(tls.VersionTLS12)
	if err != nil {
		err = dial(tls.VersionTLS10) // some servers only speak TLS 1.0/1.1
	}
	if err == nil {
		r.Trusted = true
		return r
	}

	// x509 verification errors are returned by value (not wrapped as
	// pointers), so AsType must use the value types to match them.
	if he, ok := errors.AsType[x509.HostnameError](err); ok {
		r.HostnameOK = false
		r.HostnameErr = fmt.Sprintf("证书主机名 %q 与 %q 不匹配", he.Host, host)
		r.Reason = r.HostnameErr
	} else if _, ok := errors.AsType[x509.UnknownAuthorityError](err); ok {
		r.Reason = "未知颁发机构 (证书自签名或不被系统信任)"
	} else if ci, ok := errors.AsType[x509.CertificateInvalidError](err); ok {
		switch ci.Reason {
		case x509.Expired:
			// The precise expired/not-yet-valid state comes from the
			// certificate's validity period shown in the report.
			r.Reason = "证书不在有效期内 (已过期或尚未生效)"
		case x509.CANotAuthorizedForThisName:
			r.Reason = "CA 无权签发该域名"
		default:
			r.Reason = fmt.Sprintf("证书无效: %v", ci.Reason)
		}
	} else if _, ok := errors.AsType[x509.InsecureAlgorithmError](err); ok {
		r.Reason = "证书使用了不安全的签名算法"
	} else {
		r.Reason = err.Error()
	}
	return r
}

func versionName(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLSv1.3"
	case tls.VersionTLS12:
		return "TLSv1.2"
	case tls.VersionTLS11:
		return "TLSv1.1"
	case tls.VersionTLS10:
		return "TLSv1.0"
	}
	return fmt.Sprintf("0x%04X", v)
}

// recordConn records everything read from the underlying connection so the
// ServerKeyExchange handshake message can be inspected afterwards.
type recordConn struct {
	net.Conn
	buf bytes.Buffer
}

func (c *recordConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.buf.Write(p[:n])
	}
	return n, err
}

// namedCurves maps TLS named curve IDs (RFC 8422 / RFC 7919) to names.
var namedCurves = map[uint16]string{
	19: "secp192r1", 21: "secp224r1", 22: "secp256k1", 23: "secp256r1",
	24: "secp384r1", 25: "secp521r1", 29: "x25519", 30: "x448",
}

// curveEquivRSA maps a named curve to the RSA key size with roughly the same
// security strength (NIST SP 800-57 table 2), e.g. 128-bit security = 3072-bit RSA.
var curveEquivRSA = map[uint16]int{
	19: 1024, 21: 2048, 22: 3072, 23: 3072, 24: 7680, 25: 15360, 29: 3072, 30: 7680,
}

// parseKeyExchange walks the recorded TLS records and handshake messages,
// returning the key exchange detail from the ServerKeyExchange message.
func parseKeyExchange(data []byte) string {
	for len(data) >= 5 && data[0] == 22 { // record type: handshake
		recLen := int(data[3])<<8 | int(data[4])
		if len(data) < 5+recLen {
			return ""
		}
		payload := data[5 : 5+recLen]
		for len(payload) >= 4 {
			msgType := payload[0]
			msgLen := int(payload[1])<<16 | int(payload[2])<<8 | int(payload[3])
			if len(payload) < 4+msgLen {
				return ""
			}
			if msgType == 12 { // ServerKeyExchange
				return parseSKE(payload[4 : 4+msgLen])
			}
			payload = payload[4+msgLen:]
		}
		data = data[5+recLen:]
	}
	return ""
}

// parseSKE extracts the curve (ECDHE) or prime size (DHE) from a TLS 1.2
// ServerKeyExchange body.
func parseSKE(body []byte) string {
	if len(body) < 3 {
		return ""
	}
	if body[0] == 3 { // named curve
		id := uint16(body[1])<<8 | uint16(body[2])
		name := namedCurves[id]
		if name == "" {
			name = fmt.Sprintf("curve-%d", id)
		}
		if eq := curveEquivRSA[id]; eq > 0 {
			return fmt.Sprintf("ECDH %s (eq. %d-bit RSA)", name, eq)
		}
		return "ECDH " + name
	}
	// DHE: 2-byte prime length followed by the prime.
	primeLen := int(body[0])<<8 | int(body[1])
	if primeLen > 0 && 2+primeLen <= len(body) {
		return fmt.Sprintf("DH %d bits", primeLen*8)
	}
	return ""
}
