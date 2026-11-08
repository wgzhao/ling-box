package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/password"
)

var passwordCmd = &cobra.Command{
	Use:   "password",
	Short: "Secure password generation tool",
	Long: `Generate secure random passwords with customizable options.

By default, the characters |, !, [, ], $, and backtick are excluded from
special characters to avoid encoding issues in certain business systems.
Use --exclude-chars "" to include all special characters.

Examples:
  lingbox password
  lingbox password -l 24
  lingbox password -c 5
  lingbox password -d
  lingbox password -u
  lingbox password -n
  lingbox password --exclude-chars ""       # include all special chars
  lingbox password --exclude-chars "@#&"    # exclude @, #, &`,
	Run: func(cmd *cobra.Command, args []string) {
		length, _ := cmd.Flags().GetInt("length")
		noSpecial, _ := cmd.Flags().GetBool("no-special")
		uppercaseOnly, _ := cmd.Flags().GetBool("uppercase-only")
		digitsOnly, _ := cmd.Flags().GetBool("digits-only")
		count, _ := cmd.Flags().GetInt("count")
		excludeChars, _ := cmd.Flags().GetString("exclude-chars")
		asJSON, _ := cmd.Flags().GetBool("json")

		opts := password.Options{
			Length:         length,
			IncludeSpecial: !noSpecial,
			UppercaseOnly:  uppercaseOnly,
			DigitsOnly:     digitsOnly,
			ExcludeChars:   excludeChars,
		}

		pwds := make([]string, 0, max(count, 0))
		for range count {
			pwd, err := password.Generate(opts)
			if err != nil {
				if asJSON {
					_ = writeErrorResult(cmd.OutOrStdout(), "password", err)
					return
				}
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
			pwds = append(pwds, pwd)
		}

		if asJSON {
			text := strings.Join(pwds, "\n")
			_ = writeResult(cmd.OutOrStdout(), "password", text, text, nil)
			return
		}

		for _, pwd := range pwds {
			fmt.Println(pwd)
		}
	},
}

func init() {
	passwordCmd.Flags().IntP("length", "l", 16, "Password length (default: 16)")
	passwordCmd.Flags().BoolP("no-special", "n", false, "Exclude special characters")
	passwordCmd.Flags().BoolP("uppercase-only", "u", false, "Use only uppercase letters")
	passwordCmd.Flags().BoolP("digits-only", "d", false, "Use only digits")
	passwordCmd.Flags().IntP("count", "c", 1, "Number of passwords to generate (default: 1)")
	passwordCmd.Flags().StringP("exclude-chars", "x", "|![]$()`", "Characters to exclude from generated password (default: |![]$`)")
	passwordCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	rootCmd.AddCommand(passwordCmd)
}
