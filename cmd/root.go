package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ling-box",
	Short: "玲珑盒 - A collection of useful utility tools",
	Long: `ling-box (玲珑盒) is a cross-platform CLI toolbox for developers.

It provides handy utilities for:
- URL encoding/decoding
- Base64 encoding/decoding (including URL-safe mode)
- BCrypt password hashing and verification
- QR code generation
- Secure password generation`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
