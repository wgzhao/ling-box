package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/jsonx"
)

var jsonCmd = &cobra.Command{
	Use:   "json",
	Short: "JSON tools: format and verify",
	Long: `JSON formatting and validation.

Subcommands:
  format    Re-indent JSON
  verify    Validate JSON syntax`,
}

var jsonFormatCmd = &cobra.Command{
	Use:   "format [file]",
	Short: "Re-indent JSON",
	Long: `Re-indent JSON input with a consistent layout. Key order is
preserved unless --sort-keys is given; --compact collapses the output
to a single line.

Examples:
  lingbox json format data.json
  cat data.json | lingbox json format --indent 4
  cat data.json | lingbox json format --sort-keys --compact`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		indent, _ := cmd.Flags().GetInt("indent")
		sortKeys, _ := cmd.Flags().GetBool("sort-keys")
		compact, _ := cmd.Flags().GetBool("compact")
		if indent < 0 || indent > 16 {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: --indent must be between 0 and 16\n")
			return
		}
		input, err := readInput(args)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		out, err := jsonx.Format(input, jsonx.FormatOptions{
			Indent:   strings.Repeat(" ", indent),
			SortKeys: sortKeys,
			Compact:  compact,
		})
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}
		fmt.Println(string(out))
	},
}

var jsonVerifyCmd = &cobra.Command{
	Use:   "verify [file]",
	Short: "Validate JSON syntax",
	Long: `Check that the input is well-formed JSON. Reports the first
offending line and column on failure and exits with a non-zero status,
so the command can be used in scripts.

Examples:
  lingbox json verify data.json
  cat data.json | lingbox json verify`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		quiet, _ := cmd.Flags().GetBool("quiet")
		input, err := readInput(args)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			os.Exit(1)
			return
		}
		if err := jsonx.Verify(input); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			os.Exit(1)
			return
		}
		if !quiet {
			fmt.Fprintln(cmd.OutOrStdout(), "valid JSON")
		}
	},
}

func init() {
	jsonFormatCmd.Flags().IntP("indent", "i", 2, "Indentation width in spaces")
	jsonFormatCmd.Flags().BoolP("sort-keys", "s", false, "Sort object keys alphabetically")
	jsonFormatCmd.Flags().BoolP("compact", "c", false, "Output a single line")
	jsonVerifyCmd.Flags().BoolP("quiet", "q", false, "Suppress the success message")
	jsonCmd.AddCommand(jsonFormatCmd)
	jsonCmd.AddCommand(jsonVerifyCmd)
	rootCmd.AddCommand(jsonCmd)
}

// readInput resolves command input: an optional file argument, otherwise
// stdin. It must be non-empty.
func readInput(args []string) ([]byte, error) {
	if len(args) > 0 {
		input, err := os.ReadFile(args[0])
		if err != nil {
			return nil, fmt.Errorf("read file: %v", err)
		}
		return input, nil
	}
	input, err := readStdin()
	if err != nil {
		return nil, err
	}
	if len(input) == 0 {
		return nil, fmt.Errorf("no input provided. Pipe data or specify a file")
	}
	return input, nil
}
