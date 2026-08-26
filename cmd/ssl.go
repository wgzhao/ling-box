package cmd

import (
	"cmp"
	"crypto/tls"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/ssl"
	"golang.org/x/term"
)

var sslCmd = &cobra.Command{
	Use:   "ssl",
	Short: "SSL/TLS certificate tools",
	Long: `SSL/TLS certificate inspection and utilities.

Subcommands:
  cert    Inspect X.509 certificates in detail
  host    Scan a host's TLS protocols, cipher suites, and certificate`,
}

// Terminal colors for security ratings (auto-disabled when not a TTY or
// when NO_COLOR is set).
const (
	cRed    = "\x1b[31m"
	cYellow = "\x1b[33m"
	cGreen  = "\x1b[32m"
	cReset  = "\x1b[0m"
)

var colorEnabled = false

var sslCertCmd = &cobra.Command{
	Use:   "cert [pem-content|file]",
	Short: "Inspect X.509 certificates",
	Long: `Parse and display detailed information about X.509 certificates:
subject and issuer, signature algorithm, public key and key strength,
validity period (including whether it is expired), and extensions.

The input can be:
  - a PEM or DER (.crt) file, passed via --file
  - a file path or inline PEM content as the positional argument
  - PEM data piped through stdin

Inline PEM content is accepted as-is even though it starts with dashes.

Examples:
  lingbox ssl cert server.crt
  lingbox ssl cert -f fullchain.pem
  lingbox ssl cert '-----BEGIN CERTIFICATE-----...'
  cat cert.pem | lingbox ssl cert`,
	// Inline PEM starts with "-----BEGIN", which pflag would reject as a
	// malformed flag, so parse the few flags by hand instead.
	DisableFlagParsing: true,
	Args:               cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var file string
		var positional []string
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "-h", "--help":
				cmd.Help()
				return
			case "-f", "--file":
				if i+1 >= len(args) {
					fmt.Fprintln(cmd.OutOrStderr(), "Error: flag needs an argument: -f")
					return
				}
				i++
				file = args[i]
			default:
				positional = append(positional, args[i])
			}
		}
		if len(positional) > 1 {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: accepts at most 1 arg(s), received %d\n", len(positional))
			return
		}

		data, err := certInput(file, positional)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		// Inline PEM often arrives with literal \n escapes (single-line
		// shell arguments); unescape them so it parses like real PEM.
		// Base64 content can never contain a literal backslash, so this
		// only fires for escaped input.
		if text := string(data); strings.Contains(text, `\n`) {
			data = []byte(strings.ReplaceAll(strings.ReplaceAll(text, `\r\n`, "\n"), `\n`, "\n"))
		}

		certs, err := ssl.Parse(data)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		now := time.Now()
		for i, c := range certs {
			if i > 0 {
				fmt.Println()
			}
			printCertificate(c, now, len(certs))
		}
	},
}

var sslHostCmd = &cobra.Command{
	Use:   "host <hostname|URL>",
	Short: "Scan a host's TLS protocols, cipher suites, and certificate",
	Long: `Connect to a host and report its supported TLS protocol versions,
the cipher suites accepted on each version (with security ratings), and
the presented certificate's validity, trust status, and security.

The target defaults to port 443. A port can be given as host:port or
https://host:port. Note that SSL 3.0 cannot be probed (Go's TLS client
does not implement it) and TLS 1.3 cipher suites are not individually
negotiable, so they are reported as a group.

Examples:
  lingbox ssl host www.baidu.com
  lingbox ssl host https://www.baidu.com:8443
  lingbox ssl host example.com:8443

Progress output (shown by default on a terminal) can be disabled with
--quiet for scripted or batch use.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		timeoutSec, _ := cmd.Flags().GetInt("timeout")
		quiet, _ := cmd.Flags().GetBool("quiet")
		host, port, err := parseHostTarget(args[0])
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		// Progress goes to stderr and only when it is a terminal, so a
		// piped stdout keeps a clean report.
		var p *progress
		if !quiet && term.IsTerminal(int(os.Stderr.Fd())) {
			p = newProgress(cmd.ErrOrStderr())
			p.start()
		}
		var onProgress func(string)
		if p != nil {
			onProgress = p.update
		}
		info, err := ssl.ScanHost(host, port, time.Duration(timeoutSec)*time.Second, onProgress)
		if p != nil {
			p.stop()
		}
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}
		printHostReport(info)
	},
}

func init() {
	sslCertCmd.Flags().StringP("file", "f", "", "Read certificate from a PEM or DER (.crt) file")
	sslHostCmd.Flags().IntP("timeout", "t", 5, "Per-handshake timeout in seconds")
	sslHostCmd.Flags().BoolP("quiet", "q", false, "Suppress progress output (for batch mode)")
	sslCmd.AddCommand(sslCertCmd)
	sslCmd.AddCommand(sslHostCmd)
	rootCmd.AddCommand(sslCmd)
	colorEnabled = os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(os.Stdout.Fd()))
}

// progress renders a single-line spinner with the current scan status.
// Updates are pushed via update; the display is only rendered while the
// spinner ticks, so no output is written between updates.
type progress struct {
	w        io.Writer
	mu       sync.Mutex
	text     string
	done     chan struct{}
	finished chan struct{}
	running  bool
}

func newProgress(w io.Writer) *progress {
	return &progress{
		w:        w,
		done:     make(chan struct{}),
		finished: make(chan struct{}),
	}
}

func (p *progress) start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	done := p.done
	p.mu.Unlock()

	go func() {
		defer close(p.finished)
		frames := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				p.mu.Lock()
				text := p.text
				p.mu.Unlock()
				if text == "" {
					continue
				}
				fmt.Fprintf(p.w, "\r\x1b[K%s %s", string(frames[i%len(frames)]), text)
				i++
			}
		}
	}()
}

// update replaces the current status text.
func (p *progress) update(text string) {
	p.mu.Lock()
	p.text = text
	p.mu.Unlock()
}

// stop terminates the spinner and clears its line.
func (p *progress) stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	close(p.done)
	p.running = false
	p.mu.Unlock()
	<-p.finished
	fmt.Fprint(p.w, "\r\x1b[K")
}

// parseHostTarget parses a hostname, host:port, or scheme://host:port
// target, defaulting to port 443.
func parseHostTarget(arg string) (string, int, error) {
	s := arg
	if _, after, found := strings.Cut(s, "://"); found {
		s = after
	}
	if before, _, found := strings.Cut(s, "/"); found {
		s = before
	}
	if s == "" {
		return "", 0, fmt.Errorf("empty host")
	}
	host := s
	port := 443
	if h, p, err := net.SplitHostPort(s); err == nil {
		host = h
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return "", 0, fmt.Errorf("invalid port %q", p)
		}
		port = n
	}
	return host, port, nil
}

// colorize wraps s in ANSI color c, unless color output is disabled.
func colorize(s, c string) string {
	if !colorEnabled || c == "" {
		return s
	}
	return c + s + cReset
}

// printHostReport renders the full TLS scan report.
func printHostReport(info *ssl.HostInfo) {
	fmt.Printf("目标: %s:%d", info.Host, info.Port)
	if info.RemoteIP != "" && info.RemoteIP != info.Host {
		fmt.Printf(" (解析到 %s)", info.RemoteIP)
	}
	fmt.Println()

	// Protocol support.
	fmt.Println("\n协议支持:")
	for _, p := range info.Protocols {
		var status string
		switch {
		case !p.Supported:
			status = "✗ 不支持"
		case p.Version < tls.VersionTLS12:
			status = colorize("✓ 支持 (不安全的协议版本)", cYellow)
		default:
			status = colorize("✓ 支持", cGreen)
		}
		fmt.Printf("  %-8s %s\n", p.Name, status)
	}

	// Cipher suites per supported protocol.
	for _, p := range info.Protocols {
		if !p.Supported {
			continue
		}
		fmt.Printf("\n%s 加密套件 (共 %d 个):\n", p.Name, len(p.Suites))
		for _, s := range p.Suites {
			kx := cmp.Or(s.KeyDetail, s.KeyExchange)
			fs := ""
			if s.ForwardSecrecy {
				fs = colorize("FS", cGreen)
			}
			var rating string
			switch s.Rating {
			case ssl.RatingInsecure:
				rating = colorize("INSECURE", cRed)
			case ssl.RatingWeak:
				rating = colorize("WEAK", cYellow)
			}
			fmt.Printf("    %-57s %4d  %-30s  %-2s  %s\n",
				s.Name+" ("+s.Code+")", s.KeyBits, kx, fs, rating)
		}
	}

	// Certificate.
	c := info.Certificate
	now := time.Now()
	fmt.Println("\n证书:")
	fmt.Printf("  主题:     %s\n", c.Subject)
	fmt.Printf("  签发者:   %s\n", c.Issuer)
	fmt.Printf("  有效期:   %s ~ %s\n",
		c.NotBefore.UTC().Format("2006-01-02 15:04:05 MST"),
		c.NotAfter.UTC().Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("  状态:     %s\n", validityStatus(c, now))
	fmt.Printf("  公钥:     %s (%d bits)\n", c.PublicKeyAlgorithm, c.PublicKeyBits)
	fmt.Printf("  签名算法: %s\n", c.SignatureAlgorithm)

	trust := info.Trust
	if trust.Trusted {
		fmt.Printf("  信任校验: %s 受信任 (系统根证书验证)\n", colorize("✓", cGreen))
	} else {
		fmt.Printf("  信任校验: %s %s\n", colorize("✗", cRed), trust.Reason)
	}
	if trust.HostnameOK {
		fmt.Printf("  主机名:   %s 与证书匹配\n", colorize("✓", cGreen))
	} else if trust.HostnameErr != "" {
		fmt.Printf("  主机名:   %s %s\n", colorize("✗", cRed), trust.HostnameErr)
	}

	// Security summary.
	fmt.Println("\n安全检查:")
	for _, p := range info.Protocols {
		if p.Supported && p.Version < tls.VersionTLS12 {
			printCheck(false, fmt.Sprintf("支持不安全的协议版本 %s", p.Name))
		}
	}
	var hasInsecure, hasWeak, hasNoFS bool
	for _, p := range info.Protocols {
		for _, s := range p.Suites {
			if s.Rating == ssl.RatingInsecure {
				hasInsecure = true
			}
			if s.Rating == ssl.RatingWeak {
				hasWeak = true
			}
			if !s.ForwardSecrecy {
				hasNoFS = true
			}
		}
	}
	if hasInsecure {
		printCheck(false, "存在不安全 (INSECURE) 的加密套件")
	} else {
		printCheck(true, "未发现不安全 (INSECURE) 的加密套件")
	}
	if hasWeak {
		printCheck(false, "存在弱加密 (WEAK) 的加密套件")
	} else {
		printCheck(true, "未发现弱加密 (WEAK) 的加密套件")
	}
	if hasNoFS {
		printCheck(false, "存在无前向保密 (RSA/静态密钥交换) 的套件")
	} else {
		printCheck(true, "所有套件均支持前向保密")
	}
	if st := c.Status(now); st == "valid" {
		printCheck(true, "证书在有效期内")
	} else {
		printCheck(false, fmt.Sprintf("证书%s", validityStatus(c, now)))
	}
	if trust.Trusted {
		printCheck(true, "证书受系统信任")
	} else {
		printCheck(false, "证书不受信任")
	}
	if trust.HostnameOK {
		printCheck(true, "主机名与证书匹配")
	} else if trust.HostnameErr != "" {
		printCheck(false, "主机名与证书不匹配")
	}
	if sec := c.SecurityBits(); sec >= 112 {
		printCheck(true, fmt.Sprintf("公钥强度充足 (≈%d-bit 安全强度)", sec))
	} else {
		printCheck(false, fmt.Sprintf("公钥强度不足 (≈%d-bit 安全强度)", sec))
	}
	if strings.Contains(c.SignatureAlgorithm, "SHA1") {
		printCheck(false, "使用过时的 SHA1 签名算法")
	} else {
		printCheck(true, "签名算法安全")
	}
}

// printCheck prints one security summary line with a colored mark.
func printCheck(ok bool, text string) {
	mark := colorize("✗", cRed)
	if ok {
		mark = colorize("✓", cGreen)
	}
	fmt.Printf("  [%s] %s\n", mark, text)
}

// certInput resolves the certificate input: --file wins, then the positional
// argument (an existing file path, or inline PEM content), then stdin.
func certInput(file string, args []string) ([]byte, error) {
	if file != "" {
		return os.ReadFile(file)
	}
	if len(args) == 1 {
		if _, err := os.Stat(args[0]); err == nil {
			return os.ReadFile(args[0])
		}
		return []byte(args[0]), nil
	}
	return readStdin()
}

// printCertificate renders one certificate as aligned key-value lines.
func printCertificate(c *ssl.Certificate, now time.Time, total int) {
	fmt.Printf("证书 %d/%d (共 %d 个)\n", c.Index, total, total)
	fmt.Println(strings.Repeat("=", 36))

	label := func(name string) string { return fmt.Sprintf("%-24s", name) }
	// x509.Certificate.Version is 1-indexed (1 = v1, 3 = v3).
	fmt.Printf("%s %s\n", label("版本 (Version)"), "v"+strconv.Itoa(c.Version))
	fmt.Printf("%s %s\n", label("序列号 (Serial)"), c.SerialNumber)
	fmt.Printf("%s %s\n", label("签名算法 (Sig Alg)"), c.SignatureAlgorithm)
	fmt.Printf("%s %s\n", label("签发者 (Issuer)"), c.Issuer)
	fmt.Printf("%s %s\n", label("主题 (Subject)"), c.Subject)
	fmt.Printf("%s %s ~ %s\n", label("有效期 (Validity)"),
		c.NotBefore.UTC().Format("2006-01-02 15:04:05 MST"),
		c.NotAfter.UTC().Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("%s %s\n", label("状态 (Status)"), validityStatus(c, now))

	bits := ""
	if c.PublicKeyBits > 0 {
		bits = fmt.Sprintf(" (%d bits)", c.PublicKeyBits)
	}
	fmt.Printf("%s %s%s\n", label("公钥 (Public Key)"), c.PublicKeyAlgorithm, bits)
	for line := range strings.SplitSeq(strings.TrimRight(c.PublicKeyPEM, "\n"), "\n") {
		fmt.Printf("%s %s\n", label(""), line)
	}

	ext := label("扩展 (Extensions)")
	fmt.Println(ext)
	section := func(name, value string) {
		if value != "" {
			fmt.Printf("%s %s\n", label("  "+name), value)
		}
	}
	section("密钥用途 (Key Usage)", c.KeyUsage)
	section("扩展密钥用途 (Ext Usage)", c.ExtKeyUsage)
	constraints := "CA:FALSE"
	if c.IsCA {
		constraints = "CA:TRUE"
		if c.MaxPathLen >= 0 {
			constraints += fmt.Sprintf(", pathlen:%d", c.MaxPathLen)
		}
	}
	section("基本约束 (Constraints)", constraints)
	section("主体备用名 (SAN)", strings.Join(c.SANs, ", "))
	section("主题密钥标识 (SKI)", c.SubjectKeyID)
	section("颁发者密钥标识 (AKI)", c.AuthorityKeyID)
	section("证书策略 (Policies)", strings.Join(c.PolicyIDs, ", "))
	section("OCSP 服务器", strings.Join(c.OCSPURLs, ", "))
	section("CA 签发者 URL", strings.Join(c.IssuerURLs, ", "))
	section("CRL 分发点", strings.Join(c.CRLURLs, ", "))
	section("其他扩展 (Others)", strings.Join(c.UnknownExtensions, ", "))
}

// validityStatus renders the validity state with remaining/overdue days.
func validityStatus(c *ssl.Certificate, now time.Time) string {
	days := func(dur time.Duration) int {
		return int(math.Ceil(dur.Hours() / 24))
	}
	switch c.Status(now) {
	case "valid":
		return fmt.Sprintf("有效 (valid)，剩余 %d 天", days(c.NotAfter.Sub(now)))
	case "expired":
		return fmt.Sprintf("已过期 (expired)，逾期 %d 天", days(now.Sub(c.NotAfter)))
	default:
		return fmt.Sprintf("尚未生效，%d 天后生效", days(c.NotBefore.Sub(now)))
	}
}
