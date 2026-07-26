package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/bcrypt"
)

var bcryptCmd = &cobra.Command{
	Use:   "bcrypt <password> [hash]",
	Short: "BCrypt password hashing tool",
	Long: `Generate bcrypt password hashes or verify a password against a hash.

To generate a hash, use --generate with a password.
To verify, use --verify with a password and a hash.

Examples:
  lingboxbcrypt -g mypassword
  lingboxbcrypt -v mypassword '$2a$12$...'`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		generate, _ := cmd.Flags().GetBool("generate")
		verify, _ := cmd.Flags().GetBool("verify")
		cost, _ := cmd.Flags().GetInt("cost")

		switch {
		case generate:
			result, err := bcrypt.Hash(input, cost)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
			fmt.Println(result)
		case verify:
			if len(args) < 2 {
				fmt.Fprintln(cmd.OutOrStderr(), "Error: hash is required for verification")
				return
			}
			match, err := bcrypt.Verify(input, args[1])
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
			if match {
				fmt.Println("Password matches!")
			} else {
				fmt.Println("Password does not match.")
			}
		default:
			fmt.Fprintln(cmd.OutOrStderr(), "Please specify --generate or --verify")
		}
	},
}

func init() {
	bcryptCmd.Flags().BoolP("generate", "g", false, "Generate a bcrypt hash")
	bcryptCmd.Flags().BoolP("verify", "v", false, "Verify a password against a hash")
	bcryptCmd.Flags().IntP("cost", "c", 12, "Cost factor for bcrypt (default: 12)")
	rootCmd.AddCommand(bcryptCmd)
}
