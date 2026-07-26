package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/unicode"
)

var unicodeCmd = &cobra.Command{
	Use:   "unicode <string>",
	Short: "Unicode encoding and decoding tool",
	Long: `Encode or decode Unicode escape sequences.

Encode converts non-ASCII characters (Chinese, emoji, etc.) to \uXXXX format.
Decode converts \uXXXX sequences back to human-readable text.
Without -e/-d, auto-detects: encodes plain text, decodes if contains \uXXXX.

Examples:
  lingboxunicode -e '你好世界'
  lingboxunicode -d '你好世界'
  lingboxunicode '你好'`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		encode, _ := cmd.Flags().GetBool("encode")
		decode, _ := cmd.Flags().GetBool("decode")

		switch {
		case encode:
			fmt.Println(unicode.Encode(input))
		case decode:
			result, err := unicode.Decode(input)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
			fmt.Println(result)
		default:
			// Auto-detect: if input looks encoded, decode; otherwise encode
			if unicode.IsEncoded(input) {
				result, err := unicode.Decode(input)
				if err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
					return
				}
				fmt.Println(result)
			} else {
				fmt.Println(unicode.Encode(input))
			}
		}
	},
}

func init() {
	unicodeCmd.Flags().BoolP("encode", "e", false, "Encode text to unicode escapes")
	unicodeCmd.Flags().BoolP("decode", "d", false, "Decode unicode escapes to text")
	rootCmd.AddCommand(unicodeCmd)
}
