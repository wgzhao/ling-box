package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/launcher"
)

var launcherCmd = &cobra.Command{
	Use:   "launcher",
	Short: "Launcher integrations (Alfred, uTools, ...)",
}

var launcherAlfredCmd = &cobra.Command{
	Use:   "alfred",
	Short: "Print Alfred Script Filter JSON for a query",
	Long: `Print Alfred Script Filter JSON for a query.

The Alfred workflow calls this on every keystroke:
  lingbox launcher alfred --query "<query>"

Errors are reported as items rather than as a non-zero exit so Alfred can
always render something.`,
	Run: func(cmd *cobra.Command, args []string) {
		query, _ := cmd.Flags().GetString("query")
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetEscapeHTML(false)

		// A panic inside a Script Filter would leave Alfred with no JSON at
		// all, so turn it into an item the user can read.
		defer func() {
			if r := recover(); r != nil {
				_ = enc.Encode(launcher.Response{Items: []launcher.Item{{
					Title:    "ling-box internal error",
					Subtitle: fmt.Sprint(r),
					Valid:    false,
				}}})
			}
		}()

		_ = enc.Encode(launcher.Response{Items: launcher.Build(query, execLingbox)})
	},
}

// execLingbox runs the current binary as a child process. Reusing the real
// CLI keeps the palette honest: it sees exactly what a terminal user sees,
// and there is no second argument parser to drift.
func execLingbox(argv []string, stdin string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate lingbox: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c := exec.CommandContext(ctx, exe, argv...)
	if stdin != "" {
		c.Stdin = strings.NewReader(stdin)
	}
	c.Env = cleanLauncherEnv()

	var out, errOut bytes.Buffer
	c.Stdout = &out
	c.Stderr = &errOut

	if err := c.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", errors.New("timed out after 3s")
		}
		msg := strings.TrimSpace(errOut.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}

	// Several tools print errors to stderr and exit 0 (a CLI convention this
	// adapter has to live with). Empty stdout is the signal that something
	// went wrong.
	if out.Len() == 0 {
		msg := strings.TrimSpace(errOut.String())
		if msg == "" {
			msg = "no output"
		}
		return "", errors.New(msg)
	}
	return out.String(), nil
}

// cleanLauncherEnv removes colour/terminal settings that change a tool's
// output, then sets them deterministically for the child.
func cleanLauncherEnv() []string {
	overridden := map[string]bool{"NO_COLOR": true, "CLICOLOR": true, "TERM": true}
	env := make([]string, 0, len(os.Environ())+3)
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if !overridden[k] {
			env = append(env, kv)
		}
	}
	return append(env, "NO_COLOR=1", "CLICOLOR=0", "TERM=dumb")
}

func init() {
	launcherAlfredCmd.Flags().String("query", "", "The Alfred query")
	launcherCmd.AddCommand(launcherAlfredCmd)
	rootCmd.AddCommand(launcherCmd)
}
