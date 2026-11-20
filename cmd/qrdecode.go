package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/qrdecode"
)

var qrdecodeCmd = &cobra.Command{
	Use:     "qrdecode [image]",
	Aliases: []string{"qrread", "qr-decode"},
	Short:   "Decode a QR code image",
	Long: `Extract the text payload from a QR code image.

Reads PNG, JPEG, or GIF from a file or stdin and prints the decoded text.

Examples:
  lingbox qrdecode qrcode.png
  lingbox qrdecode screenshot.jpg
  cat qrcode.png | lingbox qrdecode`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		asJSON, _ := cmd.Flags().GetBool("json")

		var (
			result qrdecode.Result
			err    error
		)
		if len(args) == 1 {
			result, err = qrdecode.DecodeFile(args[0])
		} else {
			data, readErr := readStdin()
			if readErr != nil {
				err = readErr
			} else if data == nil {
				err = fmt.Errorf("no image: pass a file or pipe image data on stdin")
			} else {
				result, err = qrdecode.DecodeBytes(data)
			}
		}
		if err != nil {
			if asJSON {
				_ = writeErrorResult(cmd.OutOrStdout(), "qrdecode", err)
				return
			}
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		if asJSON {
			extra := struct {
				Format string `json:"format"`
			}{Format: result.Format}
			_ = writeResult(cmd.OutOrStdout(), "qrdecode", result.Text, result.Text, extra)
			return
		}

		fmt.Fprintln(cmd.OutOrStdout(), result.Text)
	},
}

func init() {
	qrdecodeCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	rootCmd.AddCommand(qrdecodeCmd)
}
