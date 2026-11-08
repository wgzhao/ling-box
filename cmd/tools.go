package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/launcher"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List the tools exposed to launchers",
	Long: `List the tools that launcher integrations (Alfred, uTools, ...) can run.

Use --json to get the stable, machine-readable form consumed by launchers.`,
	Run: func(cmd *cobra.Command, args []string) {
		asJSON, _ := cmd.Flags().GetBool("json")
		tools := launcher.Tools()

		if asJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetEscapeHTML(false)
			_ = enc.Encode(struct {
				Schema int             `json:"schema"`
				Tools  []launcher.Tool `json:"tools"`
			}{Schema: 1, Tools: tools})
			return
		}

		for _, t := range tools {
			fmt.Fprintf(cmd.OutOrStdout(), "%-10s %s\n", t.ID, t.Summary)
		}
	},
}

func init() {
	toolsCmd.Flags().Bool("json", false, "Output the machine-readable tool list")
	rootCmd.AddCommand(toolsCmd)
}
