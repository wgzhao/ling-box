package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/color"
)

var colorCmd = &cobra.Command{
	Use:   "color",
	Short: "Color code conversion tool",
	Long: `Convert between color formats: Hex, RGB, HSL, and named colors.

Accepts input in any common format and outputs all other representations.

Examples:
  lingboxcolor "#FF0000"
  lingboxcolor "rgb(255, 0, 0)"
  lingboxcolor "hsl(0, 100%, 50%)"
  lingboxcolor "red"
  lingboxcolor "dark gray"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		raw, _ := cmd.Flags().GetBool("raw")
		asJSON, _ := cmd.Flags().GetBool("json")
		input := args[0]

		r, err := color.Convert(input)
		if err != nil {
			if asJSON {
				_ = writeErrorResult(cmd.OutOrStdout(), "color", err)
				return
			}
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		if asJSON {
			text := fmt.Sprintf("%s  %s  %s", r.Hex, r.RGB, r.HSL)
			if r.Name != "" {
				text += "  (" + r.Name + ")"
			}
			_ = writeResult(cmd.OutOrStdout(), "color", text, r.Hex, r)
			return
		}

		if raw {
			data, _ := json.MarshalIndent(r, "", "  ")
			fmt.Println(string(data))
			return
		}

		fmt.Printf("Input : %s\n", r.Input)
		fmt.Printf("Format: %s\n", r.Format)
		fmt.Printf("Hex   : %s\n", r.Hex)
		fmt.Printf("RGB   : %s\n", r.RGB)
		fmt.Printf("HSL   : %s\n", r.HSL)
		if r.Name != "" {
			fmt.Printf("Name  : %s\n", r.Name)
		}
	},
}

func init() {
	colorCmd.Flags().Bool("raw", false, "Output as raw JSON")
	colorCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	rootCmd.AddCommand(colorCmd)
}
