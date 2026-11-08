package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/base64"
)

var base64Cmd = &cobra.Command{
	Use:   "base64 <string>",
	Short: "Base64 encoding and decoding tool",
	Long: `Encode or decode Base64 strings. Supports standard and URL-safe Base64.

Examples:
  lingboxbase64 -e 'Hello World'
  lingboxbase64 -d 'SGVsbG8gV29ybGQ='
  lingboxbase64 -u -e 'test+/string'`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		encode, _ := cmd.Flags().GetBool("encode")
		decode, _ := cmd.Flags().GetBool("decode")
		urlSafe, _ := cmd.Flags().GetBool("url-safe")
		asJSON, _ := cmd.Flags().GetBool("json")

		var result string
		switch {
		case encode:
			result = base64.Encode(input, urlSafe)
		case decode:
			var err error
			result, err = base64.Decode(input, urlSafe)
			if err != nil {
				if asJSON {
					_ = writeErrorResult(cmd.OutOrStdout(), "base64", err)
					return
				}
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
		default:
			err := errors.New("please specify --encode or --decode")
			if asJSON {
				_ = writeErrorResult(cmd.OutOrStdout(), "base64", err)
				return
			}
			fmt.Fprintln(cmd.OutOrStderr(), "Please specify --encode or --decode")
			return
		}

		if asJSON {
			_ = writeResult(cmd.OutOrStdout(), "base64", result, result, nil)
			return
		}
		fmt.Println(result)
	},
}

func init() {
	base64Cmd.Flags().BoolP("encode", "e", false, "Encode the input string")
	base64Cmd.Flags().BoolP("decode", "d", false, "Decode the input string")
	base64Cmd.Flags().BoolP("url-safe", "u", false, "Use URL-safe Base64 encoding")
	base64Cmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	rootCmd.AddCommand(base64Cmd)
}
