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
		runLauncherQuery(cmd)
	},
}

var launcherQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Print launcher-agnostic JSON for a query",
	Long: `Print the same items as "launcher alfred" without Alfred-specific
wording. This is the contract used by the uTools adapter.

Long-running tools (ssl, ipgeo) return a non-zero rerun value; poll the same
query until rerun is zero.`,
	Run: func(cmd *cobra.Command, args []string) {
		runLauncherQuery(cmd)
	},
}

var launcherWorkCmd = &cobra.Command{
	Use:    "work",
	Short:  "Run a background launcher job",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		jobPath, _ := cmd.Flags().GetString("job")
		if jobPath == "" {
			return
		}
		job, err := launcher.ReadJob(jobPath)
		if err != nil {
			_ = launcher.CompleteJob(jobPath, launcher.JobResult{Error: err.Error()})
			return
		}
		timeout := time.Duration(job.Timeout) * time.Second
		if timeout <= 0 {
			timeout = 60 * time.Second
		}
		out, err := execLingboxTimeout(job.Argv, job.Stdin, timeout)
		res := launcher.JobResult{OK: err == nil, Output: out}
		if err != nil {
			res.Error = err.Error()
		}
		_ = launcher.CompleteJob(jobPath, res)
	},
}

func runLauncherQuery(cmd *cobra.Command) {
	query, _ := cmd.Flags().GetString("query")
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetEscapeHTML(false)

	// A panic inside a Script Filter would leave the launcher with no JSON
	// at all, so turn it into an item the user can read.
	defer func() {
		if r := recover(); r != nil {
			_ = enc.Encode(launcher.Response{Items: []launcher.Item{{
				Title:    "ling-box internal error",
				Subtitle: fmt.Sprint(r),
				Valid:    false,
			}}})
		}
	}()

	_ = enc.Encode(buildLauncherResponse(query))
}

func buildLauncherResponse(query string) launcher.Response {
	q := launcher.Parse(query)
	if q.Tool == nil {
		return launcher.Response{Items: launcher.Suggestions(query)}
	}
	if q.Tool.LongRunning {
		store := launcher.NewJobStore()
		store.Cleanup(10 * time.Minute)
		items, rerun := buildAsync(q, store)
		return launcher.Response{Items: items, Rerun: rerun}
	}
	return launcher.Response{Items: launcher.Build(query, execLingbox)}
}

// buildAsync returns a running item and a poll interval on the first call,
// then the completed result once the worker has written it.
func buildAsync(q launcher.Query, store launcher.JobStore) ([]launcher.Item, float64) {
	if q.Tool.RequiresInput && q.Rest == "" {
		return []launcher.Item{launcher.HintItem(q.Tool, q.Alias)}, 0
	}

	key := store.Key(q.Tool.ID + "\x00" + q.Rest)

	if res, ok := store.Load(key); ok {
		store.Remove(key)
		if res.OK {
			return []launcher.Item{launcher.ResultItem(q.Tool, res.Output)}, 0
		}
		return []launcher.Item{launcher.ErrorItem(q.Tool.ID, errors.New(res.Error))}, 0
	}

	if age, running := store.LockAge(key); running {
		if age < q.Tool.Timeout()+30*time.Second {
			return []launcher.Item{runningItem(q.Tool, q.Rest)}, 0.5
		}
		// The worker died or hung; start over.
		store.Abort(key)
	}

	argv, stdin := q.ArgvAndStdin()
	job := launcher.Job{
		Tool:    q.Tool.ID,
		Argv:    argv,
		Stdin:   stdin,
		Timeout: int(q.Tool.Timeout().Seconds()),
	}
	jobPath, err := store.Begin(key, job)
	if err != nil {
		// Another invocation won the race; poll it instead of reporting an
		// error the user cannot act on.
		if _, running := store.LockAge(key); running {
			return []launcher.Item{runningItem(q.Tool, q.Rest)}, 0.5
		}
		return []launcher.Item{launcher.ErrorItem(q.Tool.ID, fmt.Errorf("start job: %w", err))}, 0
	}

	exe, err := os.Executable()
	if err != nil {
		store.Abort(key)
		return []launcher.Item{launcher.ErrorItem(q.Tool.ID, err)}, 0
	}
	if err := startDetachedWorker(exe, jobPath); err != nil {
		store.Abort(key)
		return []launcher.Item{launcher.ErrorItem(q.Tool.ID, fmt.Errorf("start worker: %w", err))}, 0
	}
	return []launcher.Item{runningItem(q.Tool, q.Rest)}, 0.5
}

func runningItem(t *launcher.Tool, input string) launcher.Item {
	label := strings.TrimSpace(input)
	if len([]rune(label)) > 48 {
		label = string([]rune(label)[:45]) + "..."
	}
	title := t.ID
	if label != "" {
		title += ": " + label
	}
	return launcher.Item{
		UID:      "running:" + t.ID,
		Title:    title,
		Subtitle: "Running · results will appear here",
		Valid:    false,
	}
}

// startDetachedWorker starts the hidden worker without inheriting Alfred's
// stdio, so the Script Filter can exit immediately.
func startDetachedWorker(exe, jobPath string) error {
	c := exec.Command(exe, "launcher", "work", "--job", jobPath)
	c.Env = cleanLauncherEnv()
	c.SysProcAttr = detachedProcAttr()
	if err := c.Start(); err != nil {
		return err
	}
	return c.Process.Release()
}

// execLingbox runs the current binary as a child process. Reusing the real
// CLI keeps the palette honest: it sees exactly what a terminal user sees,
// and there is no second argument parser to drift.
func execLingbox(argv []string, stdin string) (string, error) {
	return execLingboxTimeout(argv, stdin, 3*time.Second)
}

func execLingboxTimeout(argv []string, stdin string, timeout time.Duration) (string, error) {
	return runLingbox(argv, stdin, timeout, true)
}

// execLingboxFile runs a command that may legitimately write a file and
// print nothing, such as `dbf export -o` or `convert -o`.
func execLingboxFile(argv []string, timeout time.Duration) error {
	_, err := runLingbox(argv, "", timeout, false)
	return err
}

func runLingbox(argv []string, stdin string, timeout time.Duration, requireOutput bool) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate lingbox: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
			return "", fmt.Errorf("timed out after %s", timeout)
		}
		msg := strings.TrimSpace(errOut.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}

	// Several tools print errors to stderr and exit 0 (a CLI convention this
	// adapter has to live with). For result-producing commands, empty stdout
	// is the signal that something went wrong; file-producing commands are
	// allowed to be silent.
	if requireOutput && out.Len() == 0 {
		msg := strings.TrimSpace(errOut.String())
		if msg == "" {
			msg = "no output"
		}
		return "", errors.New(msg)
	}
	// File-producing commands are allowed to be silent, but a message on
	// stderr still means the file was not written (the CLI convention is to
	// print the error and exit 0).
	if !requireOutput {
		if msg := strings.TrimSpace(errOut.String()); msg != "" {
			return "", errors.New(msg)
		}
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
	launcherQueryCmd.Flags().String("query", "", "The launcher query")
	launcherWorkCmd.Flags().String("job", "", "The job file")
	launcherCmd.AddCommand(launcherAlfredCmd)
	launcherCmd.AddCommand(launcherQueryCmd)
	launcherCmd.AddCommand(launcherWorkCmd)
	rootCmd.AddCommand(launcherCmd)
}
