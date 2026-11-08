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
		asJSON, _ := cmd.Flags().GetBool("json")

		var result string
		switch {
		case encode:
			result = unicode.Encode(input)
		case decode:
			var err error
			result, err = unicode.Decode(input)
			if err != nil {
				if asJSON {
					_ = writeErrorResult(cmd.OutOrStdout(), "unicode", err)
					return
				}
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
		default:
			// Auto-detect: if input looks encoded, decode; otherwise encode
			if unicode.IsEncoded(input) {
				var err error
				result, err = unicode.Decode(input)
				if err != nil {
					if asJSON {
						_ = writeErrorResult(cmd.OutOrStdout(), "unicode", err)
						return
					}
					fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
					return
				}
			} else {
				result = unicode.Encode(input)
			}
		}

		if asJSON {
			_ = writeResult(cmd.OutOrStdout(), "unicode", result, result, nil)
			return
		}
		fmt.Println(result)
	},
}

func init() {
	unicodeCmd.Flags().BoolP("encode", "e", false, "Encode text to unicode escapes")
	unicodeCmd.Flags().BoolP("decode", "d", false, "Decode unicode escapes to text")
	unicodeCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	rootCmd.AddCommand(unicodeCmd)
}
