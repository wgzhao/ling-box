package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/base64"
)

var base64Cmd = &cobra.Command{
	Use:   "base64",
	Short: "Base64 encoding and decoding tool",
	Long:  `Encode or decode Base64 strings. Supports standard and URL-safe Base64.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		encode, _ := cmd.Flags().GetBool("encode")
		decode, _ := cmd.Flags().GetBool("decode")
		urlSafe, _ := cmd.Flags().GetBool("url-safe")

		switch {
		case encode:
			fmt.Println(base64.Encode(input, urlSafe))
		case decode:
			result, err := base64.Decode(input, urlSafe)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
			fmt.Println(result)
		default:
			fmt.Fprintln(cmd.OutOrStderr(), "Please specify --encode or --decode")
		}
	},
}

func init() {
	base64Cmd.Flags().BoolP("encode", "e", false, "Encode the input string")
	base64Cmd.Flags().BoolP("decode", "d", false, "Decode the input string")
	base64Cmd.Flags().BoolP("url-safe", "u", false, "Use URL-safe Base64 encoding")
	rootCmd.AddCommand(base64Cmd)
}
