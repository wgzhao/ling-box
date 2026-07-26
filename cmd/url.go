package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/url"
)

var urlCmd = &cobra.Command{
	Use:   "url <string>",
	Short: "URL encoding and decoding tool",
	Long: `Encode or decode URL strings using standard URL encoding.

Examples:
  ling-box url -e 'hello world'
  ling-box url -d 'hello+world'
  ling-box url -e '你好'`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		encode, _ := cmd.Flags().GetBool("encode")
		decode, _ := cmd.Flags().GetBool("decode")

		switch {
		case encode:
			fmt.Println(url.Encode(input))
		case decode:
			fmt.Println(url.Decode(input))
		default:
			fmt.Fprintln(cmd.OutOrStderr(), "Please specify --encode or --decode")
		}
	},
}

func init() {
	urlCmd.Flags().BoolP("encode", "e", false, "Encode the input string")
	urlCmd.Flags().BoolP("decode", "d", false, "Decode the input string")
	rootCmd.AddCommand(urlCmd)
}
