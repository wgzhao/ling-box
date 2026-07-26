package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/convert"
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "YAML/JSON format conversion tool",
	Long: `Convert between YAML and JSON formats.

Reads from a file or stdin, auto-detects the input format,
and outputs the converted result.

Examples:
  ling-box convert data.yaml
  ling-box convert data.json
  cat data.yaml | ling-box convert
  echo '{"key": "value"}' | ling-box convert`,
	Run: func(cmd *cobra.Command, args []string) {
		var input []byte
		var err error

		if len(args) > 0 {
			// Read from file
			input, err = os.ReadFile(args[0])
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error reading file: %v\n", err)
				return
			}
		} else {
			// Read from stdin
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) != 0 {
				fmt.Fprintln(cmd.OutOrStderr(), "Error: no input provided. Pipe data or specify a file.")
				cmd.Help()
				return
			}
			input = make([]byte, 0)
			buf := make([]byte, 4096)
			for {
				n, readErr := os.Stdin.Read(buf)
				if n > 0 {
					input = append(input, buf[:n]...)
				}
				if readErr != nil {
					break
				}
			}
			if len(input) == 0 {
				fmt.Fprintln(cmd.OutOrStderr(), "Error: empty input")
				return
			}
		}

		output, from, to, err := convert.ConvertAuto(input)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		fmt.Fprintf(cmd.ErrOrStderr(), "# converted from %s to %s\n", from, to)
		fmt.Println(string(output))
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)
}
