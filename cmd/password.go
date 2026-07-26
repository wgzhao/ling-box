package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/password"
)

var passwordCmd = &cobra.Command{
	Use:   "password",
	Short: "Secure password generation tool",
	Long: `Generate secure random passwords with customizable options.

Examples:
  ling-box password
  ling-box password -l 24
  ling-box password -c 5
  ling-box password -d
  ling-box password -u
  ling-box password -n`,
	Run: func(cmd *cobra.Command, args []string) {
		length, _ := cmd.Flags().GetInt("length")
		noSpecial, _ := cmd.Flags().GetBool("no-special")
		uppercaseOnly, _ := cmd.Flags().GetBool("uppercase-only")
		digitsOnly, _ := cmd.Flags().GetBool("digits-only")
		count, _ := cmd.Flags().GetInt("count")

		opts := password.Options{
			Length:         length,
			IncludeSpecial: !noSpecial,
			UppercaseOnly:  uppercaseOnly,
			DigitsOnly:     digitsOnly,
		}

		for i := 0; i < count; i++ {
			pwd, err := password.Generate(opts)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
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
	rootCmd.AddCommand(passwordCmd)
}
