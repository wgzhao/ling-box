package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/shell"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Set up shell integration (lb alias, tab completion)",
	Long: `Write ling-box's shell integration into your shell's rc file.

The integration adds an "lb" alias for lingbox and loads lingbox's tab
completion script. It is stored between marker comments and rewritten in
place on every run, so upgrading never adds a second copy.`,
}

var shellInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Add the lb alias and tab completion to your shell's rc file",
	Long: `Add an "lb" alias for lingbox and load tab completion into the rc file of
your login shell ($SHELL).

The snippet is delimited by marker comments. Running this again after an
upgrade updates the snippet in place and removes any duplicate copies.

Use --shell to override shell detection and --rc to target a specific file.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		shellName, _ := cmd.Flags().GetString("shell")
		rc, _ := cmd.Flags().GetString("rc")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		res, err := shell.Install(shell.Options{Shell: shellName, RC: rc, DryRun: dryRun})
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()

		if dryRun {
			fmt.Fprintf(out, "Target: %s (%s)\n", res.RC, res.Shell)
			fmt.Fprintf(out, "Action: %s\n\n", actionWord(res))
			fmt.Fprint(out, res.Content)
			return nil
		}

		switch {
		case res.Created:
			fmt.Fprintf(out, "Added shell integration to %s (%s): lb alias, tab completion.\n", res.RC, res.Shell)
		case res.Changed:
			fmt.Fprintf(out, "Updated shell integration in %s (%s): lb alias, tab completion.\n", res.RC, res.Shell)
		default:
			fmt.Fprintf(out, "Shell integration in %s is already up to date.\n", res.RC)
		}
		if res.Changed {
			fmt.Fprintf(out, "Restart your shell or run: source %s\n", res.RC)
		}
		return nil
	},
}

var shellUninstallCmd = &cobra.Command{
	Use:           "uninstall",
	Short:         "Remove the lb alias and tab completion from your shell's rc file",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		shellName, _ := cmd.Flags().GetString("shell")
		rc, _ := cmd.Flags().GetString("rc")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		res, err := shell.Uninstall(shell.Options{Shell: shellName, RC: rc, DryRun: dryRun})
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()

		if dryRun {
			fmt.Fprintf(out, "Target: %s (%s)\n", res.RC, res.Shell)
			fmt.Fprintf(out, "Action: %s\n\n", actionWord(res))
			fmt.Fprint(out, res.Content)
			return nil
		}
		if res.Changed {
			fmt.Fprintf(out, "Removed shell integration from %s.\n", res.RC)
		} else {
			fmt.Fprintf(out, "No shell integration found in %s.\n", res.RC)
		}
		return nil
	},
}

func actionWord(res shell.Result) string {
	switch {
	case res.Created:
		return "create"
	case res.Changed:
		return "update"
	default:
		return "no change"
	}
}

func init() {
	for _, c := range []*cobra.Command{shellInstallCmd, shellUninstallCmd} {
		c.Flags().String("shell", "auto", "Shell to configure: auto, bash, zsh or fish")
		c.Flags().String("rc", "", "RC file to edit (default: the shell's own startup file)")
		c.Flags().Bool("dry-run", false, "Print the resulting rc file without writing it")
		_ = c.RegisterFlagCompletionFunc("shell", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return append([]string{"auto"}, shell.Supported...), cobra.ShellCompDirectiveNoFileComp
		})
		_ = c.RegisterFlagCompletionFunc("rc", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		})
	}
	shellCmd.AddCommand(shellInstallCmd, shellUninstallCmd)
	rootCmd.AddCommand(shellCmd)
}
