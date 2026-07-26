package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/qrcode"
)

var qrcodeCmd = &cobra.Command{
	Use:   "qrcode",
	Short: "QR Code generation tool",
	Long:  `Generate QR codes as PNG, JPG, or GIF images from text input.`,
	Args:  cobra.ExactArgs(1),
	Example: `  ling-box qrcode "https://example.com"
  ling-box qrcode "Hello World" -o mycode.png -s 500
  ling-box qrcode "Test" -o mycode.jpg -f JPG`,
	Run: func(cmd *cobra.Command, args []string) {
		text := args[0]
		output, _ := cmd.Flags().GetString("output")
		size, _ := cmd.Flags().GetInt("size")
		format, _ := cmd.Flags().GetString("format")

		if err := qrcode.Generate(text, output, size, format); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error generating QR code: %v\n", err)
			return
		}
		fmt.Printf("QR code generated successfully: %s\n", output)
	},
}

func init() {
	qrcodeCmd.Flags().StringP("output", "o", "qrcode.png", "Output file path (default: qrcode.png)")
	qrcodeCmd.Flags().IntP("size", "s", 300, "Size of the QR code in pixels (default: 300)")
	qrcodeCmd.Flags().StringP("format", "f", "PNG", "Image format: PNG, JPG, GIF (default: PNG)")
	rootCmd.AddCommand(qrcodeCmd)
}
