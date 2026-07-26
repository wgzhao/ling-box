package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/convert"
)

var yaml2jsonCmd = &cobra.Command{
	Use:   "yaml2json",
	Short: "Convert YAML to JSON",
	Long: `Convert YAML format data to JSON format.

Reads from a file or stdin and outputs the JSON result.

Examples:
  ling-box yaml2json config.yaml
  cat config.yaml | ling-box yaml2json`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var input []byte
		var err error

		if len(args) > 0 {
			input, err = os.ReadFile(args[0])
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error reading file: %v\n", err)
				return
			}
		} else {
			input, err = readStdin()
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
			if len(input) == 0 {
				fmt.Fprintln(cmd.OutOrStderr(), "Error: no input provided. Pipe data or specify a file.")
				cmd.Help()
				return
			}
		}

		output, err := convert.YAMLToJSON(input)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		fmt.Println(string(output))
	},
}

func init() {
	rootCmd.AddCommand(yaml2jsonCmd)
}
