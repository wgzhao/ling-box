package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/cnnum"
)

var cnnumCmd = &cobra.Command{
	Use:     "cnnum [number]",
	Aliases: []string{"zhnum", "rmb"},
	Short:   "Convert Arabic and Chinese numerals",
	Long: `Convert between Arabic numerals and Chinese numerals.

Two character sets are supported:
  lower  一百二十三   (cnnum default)
  upper  壹佰贰拾叁   (financial form; the rmb alias defaults to this)

The direction is detected automatically: Arabic digits produce Chinese
characters, Chinese characters produce Arabic digits. Integers, decimals,
negatives, thousands separators, and values up to int64 are supported.

Examples:
  lingbox cnnum 123                 # 一百二十三
  lingbox cnnum --style upper 123   # 壹佰贰拾叁
  lingbox rmb 123                   # 壹佰贰拾叁
  lingbox cnnum 壹佰贰拾叁          # 123
  lingbox cnnum 一万亿              # 1000000000000
  lingbox cnnum -12.30              # 负十二点三零
  echo 123 | lingbox cnnum`,
	// Negative numbers start with "-", so parse the arguments by hand.
	DisableFlagParsing: true,
	Args:               cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, rawArgs []string) {
		args, styleFlag, asJSON, help, err := parseCnnumArgs(rawArgs)
		if help {
			_ = cmd.Help()
			return
		}
		if err != nil {
			reportCnnumError(cmd, asJSON, err)
			return
		}

		style := cnnum.Lower
		if cmd.CalledAs() == "rmb" {
			style = cnnum.Upper
		}
		if styleFlag != "" {
			switch strings.ToLower(styleFlag) {
			case "lower":
				style = cnnum.Lower
			case "upper":
				style = cnnum.Upper
			default:
				reportCnnumError(cmd, asJSON, fmt.Errorf("invalid --style %q (supported: lower, upper)", styleFlag))
				return
			}
		}

		input := ""
		if len(args) == 1 {
			input = args[0]
		} else if len(args) > 1 {
			reportCnnumError(cmd, asJSON, fmt.Errorf("expected one number, got %d arguments", len(args)))
			return
		} else {
			stdin, err := readStdin()
			if err != nil {
				reportCnnumError(cmd, asJSON, err)
				return
			}
			if stdin == nil {
				reportCnnumError(cmd, asJSON, fmt.Errorf("no number: pass it as an argument or on stdin"))
				return
			}
			input = string(stdin)
		}

		result, err := cnnum.Convert(input, style)
		if err != nil {
			reportCnnumError(cmd, asJSON, err)
			return
		}

		if asJSON {
			_ = writeResult(cmd.OutOrStdout(), "cnnum", result.Output, result.Output, result)
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), result.Output)
	},
}

func parseCnnumArgs(raw []string) (args []string, style string, asJSON, help bool, err error) {
	for i := 0; i < len(raw); i++ {
		a := raw[i]
		switch {
		case a == "--":
			args = append(args, raw[i+1:]...)
			return
		case a == "-h" || a == "--help":
			help = true
			return
		case a == "--json":
			asJSON = true
		case a == "-s" || a == "--style":
			if i+1 >= len(raw) {
				return nil, "", asJSON, false, fmt.Errorf("%s requires a value", a)
			}
			i++
			style = raw[i]
		case strings.HasPrefix(a, "--style="):
			style = strings.TrimPrefix(a, "--style=")
		case strings.HasPrefix(a, "-s="):
			style = strings.TrimPrefix(a, "-s=")
		default:
			args = append(args, a)
		}
	}
	return
}

func reportCnnumError(cmd *cobra.Command, asJSON bool, err error) {
	if asJSON {
		_ = writeErrorResult(cmd.OutOrStdout(), "cnnum", err)
		return
	}
	fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
}

func init() {
	cnnumCmd.Flags().StringP("style", "s", "", "Chinese numeral style: lower or upper (default: lower; rmb alias: upper)")
	cnnumCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	rootCmd.AddCommand(cnnumCmd)
}
