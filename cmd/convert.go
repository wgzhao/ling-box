package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/convert"
)

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert between JSON, YAML, CSV, and Markdown",
	Long: `Convert between JSON, YAML, CSV, and Markdown (pandoc-style
-i/-o interface). The input format is read from the -i file extension
(or detected from stdin); the target format comes from -t, or is
guessed from the -o file extension when -t is omitted.

Input formats (from -i): json, yaml/yml, csv.
Output formats (from -t or -o): json, yaml, csv, markdown.

CSV and Markdown output require an array of objects (header from their
keys) or, for Markdown only, a scalar array (bullet list) or a
top-level object (key/value table). Nested objects and arrays inside
cells are encoded as compact JSON strings. CSV input uses the first
row as the header.

Examples:
  lingbox convert -i data.json -o data.yaml
  lingbox convert -i data.csv -t markdown
  lingbox convert -i data.json -o out.md
  cat data.json | lingbox convert -t yaml`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		in, _ := cmd.Flags().GetString("input")
		out, _ := cmd.Flags().GetString("output")
		to, _ := cmd.Flags().GetString("to")

		// Read input: -i file (or "-" for stdin), else stdin. The
		// format comes from the extension, or detection for stdin.
		var input []byte
		var from string
		var err error
		if in == "" || in == "-" {
			input, err = readStdin()
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
			if len(input) == 0 {
				fmt.Fprintln(cmd.OutOrStderr(), "Error: no input provided. Use -i to specify a file or pipe data in.")
				return
			}
			from, err = convert.DetectInputFormat(input)
		} else {
			input, err = os.ReadFile(in)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: read file: %v\n", err)
				return
			}
			from, err = formatFromSuffix(in)
		}
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		// Target format: -t wins, else the -o extension, else error
		// (a stdout target needs an explicit -t).
		target := to
		if target == "" {
			if out != "" && out != "-" {
				target, err = formatFromSuffix(out)
				if err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "Error: %v (specify -t to pick the target format)\n", err)
					return
				}
			} else {
				fmt.Fprintln(cmd.OutOrStderr(), "Error: no target format. Specify -t or use -o with a recognizable extension.")
				return
			}
		}

		data, err := convert.Parse(from, input)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}
		output, err := convert.Render(target, data)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		// Write output: -o file (or "-" for stdout), else stdout.
		if out != "" && out != "-" {
			if err := os.WriteFile(out, output, 0o644); err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: write file: %v\n", err)
				return
			}
			return
		}
		// CSV keeps its own line structure; a trailing newline here
		// would produce a blank line.
		if target == "csv" {
			fmt.Print(string(output))
		} else {
			fmt.Println(string(output))
		}
	},
}

func init() {
	convertCmd.Flags().StringP("input", "i", "", "Input file (default: stdin)")
	convertCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	convertCmd.Flags().StringP("to", "t", "", "Target format: json, yaml, csv, or markdown (default: guessed from -o)")
	rootCmd.AddCommand(convertCmd)
}

// formatFromSuffix guesses an input or output format from a file
// extension.
func formatFromSuffix(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "json", nil
	case ".yaml", ".yml":
		return "yaml", nil
	case ".csv":
		return "csv", nil
	case ".md", ".markdown":
		return "markdown", nil
	default:
		return "", fmt.Errorf("cannot guess format from extension %q", filepath.Ext(path))
	}
}
