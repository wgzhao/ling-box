package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/jwt"
)

var jwtCmd = &cobra.Command{
	Use:   "jwt [token]",
	Short: "Decode and verify JSON Web Tokens",
	Long: `Decode a JWT and optionally verify its signature and time claims.

Without a key the token is only decoded. Use --secret for HS* tokens and
--public-key for RS*/PS*/ES* tokens. The key may also be a PEM private key
or certificate; its public part is used.

Examples:
  lingbox jwt eyJhbGciOiJIUzI1NiIs...
  lingbox jwt --secret mysecret eyJhbGciOiJIUzI1NiIs...
  lingbox jwt --secret-base64 c2VjcmV0 eyJhbGciOiJIUzI1NiIs...
  lingbox jwt --public-key public.pem eyJhbGciOiJSUzI1NiIs...
  cat token.txt | lingbox jwt --secret mysecret`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		secret, _ := cmd.Flags().GetString("secret")
		secretBase64, _ := cmd.Flags().GetBool("secret-base64")
		publicKeyPath, _ := cmd.Flags().GetString("public-key")
		asJSON, _ := cmd.Flags().GetBool("json")

		raw := ""
		if len(args) == 1 {
			raw = args[0]
		} else {
			stdin, err := readStdin()
			if err != nil {
				reportJWTError(cmd, asJSON, err)
				return
			}
			if stdin == nil {
				reportJWTError(cmd, asJSON, fmt.Errorf("no token: pass it as an argument or on stdin"))
				return
			}
			raw = string(stdin)
		}

		tok, err := jwt.Parse(raw)
		if err != nil {
			reportJWTError(cmd, asJSON, err)
			return
		}

		var verified *bool
		if secret != "" || publicKeyPath != "" {
			key, err := jwtKey(tok.Algorithm(), secret, secretBase64, publicKeyPath)
			if err != nil {
				reportJWTError(cmd, asJSON, err)
				return
			}
			if err := jwt.Verify(tok, key); err != nil {
				reportJWTError(cmd, asJSON, fmt.Errorf("verification failed: %w", err))
				return
			}
			ok := true
			verified = &ok
		}

		now := time.Now()
		status := jwt.Times(tok.Claims, now)

		headerJSON := prettyJSON(tok.Header)
		claimsJSON := prettyJSON(tok.Claims)

		var text strings.Builder
		fmt.Fprintf(&text, "Algorithm : %s\n", tok.Algorithm())
		switch {
		case verified != nil:
			text.WriteString("Verified  : yes\n")
		case secret == "" && publicKeyPath == "":
			text.WriteString("Verified  : not checked (pass --secret or --public-key)\n")
		}
		expiredFlag := ""
		if status.Expired {
			expiredFlag = "EXPIRED"
		}
		writeTimeClaim(&text, "Expires", status.ExpiresAt, now, expiredFlag)
		if status.NotYetValid {
			writeTimeClaim(&text, "Not before", status.NotBefore, now, "NOT YET VALID")
		} else {
			writeTimeClaim(&text, "Not before", status.NotBefore, now, "")
		}
		writeTimeClaim(&text, "Issued at", status.IssuedAt, now, "")
		fmt.Fprintf(&text, "Header:\n%s\n", headerJSON)
		fmt.Fprintf(&text, "Payload:\n%s\n", claimsJSON)

		copyJSON, _ := json.Marshal(tok.Claims)
		if asJSON {
			extra := struct {
				Header    map[string]any `json:"header"`
				Claims    map[string]any `json:"claims"`
				Algorithm string         `json:"algorithm"`
				Verified  *bool          `json:"verified,omitempty"`
				Time      jwt.TimeStatus `json:"time"`
			}{
				Header:    tok.Header,
				Claims:    tok.Claims,
				Algorithm: tok.Algorithm(),
				Verified:  verified,
				Time:      status,
			}
			_ = writeResult(cmd.OutOrStdout(), "jwt", strings.TrimRight(text.String(), "\n"), string(copyJSON), extra)
			return
		}

		fmt.Fprint(cmd.OutOrStdout(), text.String())
	},
}

func jwtKey(alg, secret string, secretBase64 bool, publicKeyPath string) ([]byte, error) {
	switch {
	case strings.HasPrefix(alg, "HS"):
		if secret == "" {
			return nil, fmt.Errorf("%s token requires --secret", alg)
		}
		if publicKeyPath != "" {
			return nil, fmt.Errorf("HS* tokens use --secret, not --public-key")
		}
		if !secretBase64 {
			return []byte(secret), nil
		}
		if b, err := base64.StdEncoding.DecodeString(secret); err == nil {
			return b, nil
		}
		if b, err := base64.RawStdEncoding.DecodeString(secret); err == nil {
			return b, nil
		}
		return nil, fmt.Errorf("--secret is not valid base64")
	case strings.HasPrefix(alg, "RS"), strings.HasPrefix(alg, "PS"), strings.HasPrefix(alg, "ES"):
		if publicKeyPath == "" {
			return nil, fmt.Errorf("%s token requires --public-key", alg)
		}
		if secret != "" {
			return nil, fmt.Errorf("%s tokens use --public-key, not --secret", alg)
		}
		data, err := os.ReadFile(publicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read public key: %w", err)
		}
		return data, nil
	case alg == "none":
		return nil, fmt.Errorf(`alg "none" has no signature to verify`)
	default:
		return nil, fmt.Errorf("unsupported algorithm %q", alg)
	}
}

func prettyJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func writeTimeClaim(b *strings.Builder, label string, t *time.Time, now time.Time, flag string) {
	if t == nil {
		return
	}
	line := fmt.Sprintf("%-10s: %s (%s)", label, t.UTC().Format(time.RFC3339), relativeTime(*t, now))
	if flag != "" {
		line += " (" + flag + ")"
	}
	fmt.Fprintln(b, line)
}

func relativeTime(t, now time.Time) string {
	d := t.Sub(now)
	if d >= 0 {
		return "in " + humanDuration(d)
	}
	return humanDuration(-d) + " ago"
}

func humanDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		days := int(d.Hours() / 24)
		if days < 365 {
			return fmt.Sprintf("%dd", days)
		}
		return fmt.Sprintf("%dy%dd", days/365, days%365)
	}
}

func reportJWTError(cmd *cobra.Command, asJSON bool, err error) {
	if asJSON {
		_ = writeErrorResult(cmd.OutOrStdout(), "jwt", err)
		return
	}
	fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
}

func init() {
	jwtCmd.Flags().StringP("secret", "s", "", "HMAC secret for HS* tokens")
	jwtCmd.Flags().Bool("secret-base64", false, "Decode --secret as base64")
	jwtCmd.Flags().StringP("public-key", "p", "", "PEM public key, certificate, or private key for RS*/PS*/ES* tokens")
	jwtCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	rootCmd.AddCommand(jwtCmd)
}
