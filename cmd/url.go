package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/url"
)

var urlCmd = &cobra.Command{
	Use:   "url <string>",
	Short: "URL encoding and decoding tool",
	Long: `Encode or decode URL strings using standard URL encoding.

Examples:
  lingboxurl -e 'hello world'
  lingboxurl -d 'hello+world'
  lingboxurl -e '你好'`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		encode, _ := cmd.Flags().GetBool("encode")
		decode, _ := cmd.Flags().GetBool("decode")
		asJSON, _ := cmd.Flags().GetBool("json")

		var result string
		switch {
		case encode:
			result = url.Encode(input)
		case decode:
			result = url.Decode(input)
		default:
			err := errors.New("please specify --encode or --decode")
			if asJSON {
				_ = writeErrorResult(cmd.OutOrStdout(), "url", err)
				return
			}
			fmt.Fprintln(cmd.OutOrStderr(), "Please specify --encode or --decode")
			return
		}

		if asJSON {
			_ = writeResult(cmd.OutOrStdout(), "url", result, result, nil)
			return
		}
		fmt.Println(result)
	},
}

func init() {
	urlCmd.Flags().BoolP("encode", "e", false, "Encode the input string")
	urlCmd.Flags().BoolP("decode", "d", false, "Decode the input string")
	urlCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	rootCmd.AddCommand(urlCmd)
}
