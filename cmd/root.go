package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is overridden at build time via
// -ldflags "-X github.com/wgzhao/ling-box/cmd.version=<ver>".
var version = "dev"

// helpTemplate places the author and repository info right after the
// command description, before the usage block.
const helpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}Author: wgzhao
Repository: https://github.com/wgzhao/ling-box

{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

func init() {
	rootCmd.SetHelpTemplate(helpTemplate)
}

var rootCmd = &cobra.Command{
	Use:     "lingbox",
	Version: version,
	Short:   "玲珑盒 - A collection of useful utility tools",
	Long:    "lingbox (玲珑盒) is a cross-platform CLI toolbox for developers.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// readStdin reads all data from standard input.
func readStdin() ([]byte, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, nil
	}
	input := make([]byte, 0)
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			input = append(input, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return input, nil
}
