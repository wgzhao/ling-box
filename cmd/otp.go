package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/otp"
)

var otpCmd = &cobra.Command{
	Use:     "otp [secret|otpauth-uri]",
	Aliases: []string{"totp", "hotp"},
	Short:   "Generate TOTP/HOTP one-time passwords",
	Long: `Generate time-based (TOTP) or counter-based (HOTP) one-time passwords.

The argument is a base32 secret or a full otpauth:// URI. Lowercase and
space-separated secrets are accepted.

Examples:
  lingbox otp JBSWY3DPEHPK3PXP
  lingbox otp --digits 8 --period 60 JBSWY3DPEHPK3PXP
  lingbox otp --at 59 --digits 8 GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ
  lingbox otp --counter 5 JBSWY3DPEHPK3PXP
  lingbox otp 'otpauth://totp/ACME:alice?secret=JBSWY3DPEHPK3PXP&issuer=ACME'
  cat secret.txt | lingbox otp`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		digits, _ := cmd.Flags().GetInt("digits")
		period, _ := cmd.Flags().GetInt("period")
		algorithm, _ := cmd.Flags().GetString("algorithm")
		counter, _ := cmd.Flags().GetInt64("counter")
		atRaw, _ := cmd.Flags().GetString("at")
		asJSON, _ := cmd.Flags().GetBool("json")

		input := ""
		if len(args) == 1 {
			input = strings.TrimSpace(args[0])
		} else {
			stdin, err := readStdin()
			if err != nil {
				reportOTPError(cmd, asJSON, err)
				return
			}
			if stdin == nil {
				reportOTPError(cmd, asJSON, fmt.Errorf("no secret: pass one as an argument or on stdin"))
				return
			}
			input = strings.TrimSpace(string(stdin))
		}

		cfg := otp.Config{Digits: digits, Period: period, Algorithm: algorithm}
		isHOTP := counter >= 0

		if strings.HasPrefix(strings.ToLower(input), "otpauth://") {
			uri, err := otp.ParseURI(input)
			if err != nil {
				reportOTPError(cmd, asJSON, err)
				return
			}
			if !cmd.Flags().Changed("digits") {
				cfg.Digits = uri.Digits
			}
			if !cmd.Flags().Changed("period") {
				cfg.Period = uri.Period
			}
			if !cmd.Flags().Changed("algorithm") {
				cfg.Algorithm = uri.Algorithm
			}
			if uri.Type == "hotp" && !cmd.Flags().Changed("counter") {
				counter = int64(uri.Counter)
				isHOTP = true
			}
			input = uri.Secret
		}

		normalized, err := cfg.Normalize()
		if err != nil {
			reportOTPError(cmd, asJSON, err)
			return
		}
		cfg = normalized

		if isHOTP {
			code, err := otp.HOTP(input, uint64(counter), cfg)
			if err != nil {
				reportOTPError(cmd, asJSON, err)
				return
			}
			text := fmt.Sprintf("Code      : %s\nAlgorithm : %s\nDigits    : %d\nCounter   : %d",
				code, cfg.Algorithm, cfg.Digits, counter)
			if asJSON {
				extra := struct {
					Code      string `json:"code"`
					Algorithm string `json:"algorithm"`
					Digits    int    `json:"digits"`
					Counter   uint64 `json:"counter"`
				}{code, cfg.Algorithm, cfg.Digits, uint64(counter)}
				_ = writeResult(cmd.OutOrStdout(), "otp", text, code, extra)
				return
			}
			fmt.Fprintln(cmd.OutOrStdout(), text)
			return
		}

		at := time.Now()
		if atRaw != "" {
			parsed, err := parseOTPTime(atRaw)
			if err != nil {
				reportOTPError(cmd, asJSON, err)
				return
			}
			at = parsed
		}

		result, err := otp.TOTP(input, at, cfg)
		if err != nil {
			reportOTPError(cmd, asJSON, err)
			return
		}

		text := fmt.Sprintf("Code      : %s\nAlgorithm : %s\nDigits    : %d\nPeriod    : %ds (%ds remaining)\nAt        : %s",
			result.Code, result.Algorithm, result.Digits, result.Period,
			result.Remaining, result.At.UTC().Format(time.RFC3339))

		if asJSON {
			_ = writeResult(cmd.OutOrStdout(), "otp", text, result.Code, result)
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), text)
	},
}

func parseOTPTime(s string) (time.Time, error) {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(n, 0), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid --at value %q (use Unix seconds or RFC3339)", s)
}

func reportOTPError(cmd *cobra.Command, asJSON bool, err error) {
	if asJSON {
		_ = writeErrorResult(cmd.OutOrStdout(), "otp", err)
		return
	}
	fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
}

func init() {
	otpCmd.Flags().IntP("digits", "d", 6, "Number of digits (4-10)")
	otpCmd.Flags().Int("period", 30, "TOTP period in seconds")
	otpCmd.Flags().StringP("algorithm", "a", "SHA1", "Hash algorithm: SHA1, SHA256, SHA512")
	otpCmd.Flags().Int64P("counter", "c", -1, "HOTP counter (switches to HOTP mode)")
	otpCmd.Flags().String("at", "", "TOTP time as Unix seconds or RFC3339 (default: now)")
	otpCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	rootCmd.AddCommand(otpCmd)
}
