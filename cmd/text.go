package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/textx"
)

var textCmd = &cobra.Command{
	Use:     "text <operation> [input...]",
	Aliases: []string{"str", "case"},
	Short:   "Transform text: casing, escapes, width, stats",
	Long: `Transform text without shell one-liners full of sed and awk.

Operations:
  camel, pascal, snake, kebab, constant, title, lower, upper, slug
  html-encode, html-decode
  json-escape, json-unescape
  fullwidth, halfwidth
  stats

Input comes from the remaining arguments (joined with spaces) or stdin.

Examples:
  lingbox text snake "Hello World"        # hello_world
  lingbox text camel hello_world          # helloWorld
  lingbox text kebab "XMLHttpRequest"     # xml-http-request
  lingbox text slug "Hello, World!"       # hello-world
  lingbox text json-unescape 'hello\nworld'
  lingbox text stats "Hello 世界"
  echo 'Hello World' | lingbox text snake`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		op := strings.ToLower(strings.TrimSpace(args[0]))
		asJSON, _ := cmd.Flags().GetBool("json")

		var input string
		if len(args) > 1 {
			input = strings.Join(args[1:], " ")
		} else {
			stdin, err := readStdin()
			if err != nil {
				reportTextError(cmd, asJSON, err)
				return
			}
			if stdin == nil {
				reportTextError(cmd, asJSON, fmt.Errorf("no input: pass text as arguments or on stdin"))
				return
			}
			input = string(stdin)
		}

		if op == "stats" {
			stats := textx.Analyze(input)
			text := fmt.Sprintf("Characters : %d\nBytes      : %d\nWords      : %d\nLines      : %d\nCJK        : %d\nReading    : %d min",
				stats.Characters, stats.Bytes, stats.Words, stats.Lines, stats.CJK, stats.ReadingMinutes)
			if asJSON {
				_ = writeResult(cmd.OutOrStdout(), "text", text, text, stats)
				return
			}
			fmt.Fprintln(cmd.OutOrStdout(), text)
			return
		}

		out, err := textx.Apply(op, input)
		if err != nil {
			reportTextError(cmd, asJSON, err)
			return
		}

		if asJSON {
			_ = writeResult(cmd.OutOrStdout(), "text", out, out, nil)
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), out)
	},
}

func reportTextError(cmd *cobra.Command, asJSON bool, err error) {
	if asJSON {
		_ = writeErrorResult(cmd.OutOrStdout(), "text", err)
		return
	}
	fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
}

func init() {
	textCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	rootCmd.AddCommand(textCmd)
}
