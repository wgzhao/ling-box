package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/baseconv"
)

var baseCmd = &cobra.Command{
	Use:   "base",
	Short: "Number base conversion tool",
	Long: `Convert numbers between binary, octal, decimal, and hexadecimal.

Outputs all representations for the given input.
Use --from to specify the input base (default: auto-detect).

Auto-detection rules:
  0x...  → hex
  0b...  → binary
  0...   → octal (if all digits 0-7)
  ABC    → hex (if contains A-F)
  123    → decimal

Examples:
  ling-box base 255
  ling-box base FF --from hex
  ling-box base "0xFF"
  ling-box base 1010 -f bin`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		fromBaseStr, _ := cmd.Flags().GetString("from")

		var fromBase baseconv.Base
		var err error

		if fromBaseStr != "" {
			fromBase, err = baseconv.ParseBase(fromBaseStr)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
		} else {
			fromBase, err = baseconv.AutoDetect(input)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
		}

		result, err := baseconv.Convert(input, fromBase)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		fmt.Printf("Input : %s (%s)\n", result.Input, baseName(fromBase))
		fmt.Printf("Bin   : %s\n", result.Binary)
		fmt.Printf("Oct   : %s\n", result.Octal)
		fmt.Printf("Dec   : %s\n", result.Decimal)
		fmt.Printf("Hex   : %s\n", result.Hex)
	},
}

func baseName(b baseconv.Base) string {
	switch b {
	case baseconv.Bin:
		return "binary"
	case baseconv.Oct:
		return "octal"
	case baseconv.Dec:
		return "decimal"
	case baseconv.Hex:
		return "hexadecimal"
	default:
		return fmt.Sprintf("base-%d", b)
	}
}

func init() {
	baseCmd.Flags().StringP("from", "f", "", "Input base: bin, oct, dec, hex (default: auto-detect)")
	rootCmd.AddCommand(baseCmd)
}
