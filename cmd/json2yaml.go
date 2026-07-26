package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/convert"
)

var json2yamlCmd = &cobra.Command{
	Use:   "json2yaml",
	Short: "Convert JSON to YAML",
	Long: `Convert JSON format data to YAML format.

Reads from a file or stdin and outputs the YAML result.

Examples:
  ling-box json2yaml data.json
  cat data.json | ling-box json2yaml`,
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

		output, err := convert.JSONToYAML(input)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		fmt.Println(string(output))
	},
}

func init() {
	rootCmd.AddCommand(json2yamlCmd)
}
