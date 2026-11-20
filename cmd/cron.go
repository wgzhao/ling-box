package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/cronx"
)

var cronCmd = &cobra.Command{
	Use:     "cron <expression>",
	Aliases: []string{"crontab", "schedule"},
	Short:   "Explain a cron expression and show its next runs",
	Long: `Parse a cron expression, explain every field, and list upcoming run times.

Accepts standard 5-field expressions, 6-field expressions with a leading
seconds field, and macros: @yearly, @annually, @monthly, @weekly, @daily,
@midnight, @hourly. Month and weekday names (JAN-DEC, SUN-SAT) are supported;
day-of-month and day-of-week use the usual OR semantics when both are
restricted.

Examples:
  lingbox cron '*/5 * * * *'
  lingbox cron '0 9 * * MON-FRI' -n 10
  lingbox cron '30 * * * * *' --tz UTC
  lingbox cron '@daily'
  lingbox cron '0 0 1 * *' --from 2026-01-01T00:00:00Z -n 3

Quote the expression so the shell passes it as one argument.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		count, _ := cmd.Flags().GetInt("count")
		fromRaw, _ := cmd.Flags().GetString("from")
		tzName, _ := cmd.Flags().GetString("tz")
		asJSON, _ := cmd.Flags().GetBool("json")

		if count <= 0 {
			reportCronError(cmd, asJSON, fmt.Errorf("--count must be positive"))
			return
		}

		loc := time.Local
		if tzName != "" {
			var err error
			loc, err = time.LoadLocation(tzName)
			if err != nil {
				reportCronError(cmd, asJSON, fmt.Errorf("unknown timezone %q: %w", tzName, err))
				return
			}
		}

		from := time.Now()
		if fromRaw != "" {
			parsed, err := parseCronFrom(fromRaw)
			if err != nil {
				reportCronError(cmd, asJSON, err)
				return
			}
			from = parsed
		}

		sched, err := cronx.Parse(args[0], loc)
		if err != nil {
			reportCronError(cmd, asJSON, err)
			return
		}
		runs, err := sched.Next(from, count)
		if err != nil {
			reportCronError(cmd, asJSON, err)
			return
		}

		times := make([]string, len(runs))
		for i, run := range runs {
			times[i] = run.Format(time.RFC3339)
		}

		mode := "5 fields (minute-level)"
		if sched.Seconds {
			mode = "6 fields (second-level)"
		}
		var b strings.Builder
		fmt.Fprintf(&b, "Expression  : %s\n", sched.Expr)
		fmt.Fprintf(&b, "Mode        : %s\n", mode)
		for _, line := range sched.Explain() {
			b.WriteString(line + "\n")
		}
		fmt.Fprintf(&b, "Next run    : %s\n", times[0])
		b.WriteString("Upcoming    :\n")
		for _, t := range times {
			b.WriteString("  " + t + "\n")
		}
		text := strings.TrimRight(b.String(), "\n")

		if asJSON {
			extra := struct {
				Expression string   `json:"expression"`
				Seconds    bool     `json:"seconds"`
				Fields     []string `json:"fields"`
				NextRuns   []string `json:"next_runs"`
			}{
				Expression: sched.Expr,
				Seconds:    sched.Seconds,
				Fields:     sched.Explain(),
				NextRuns:   times,
			}
			_ = writeResult(cmd.OutOrStdout(), "cron", text, times[0], extra)
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), text)
	},
}

func parseCronFrom(s string) (time.Time, error) {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(n, 0), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid --from value %q (use Unix seconds or RFC3339)", s)
}

func reportCronError(cmd *cobra.Command, asJSON bool, err error) {
	if asJSON {
		_ = writeErrorResult(cmd.OutOrStdout(), "cron", err)
		return
	}
	fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
}

func init() {
	cronCmd.Flags().IntP("count", "n", 5, "Number of upcoming runs to list")
	cronCmd.Flags().String("from", "", "Start time (Unix seconds or RFC3339, default: now)")
	cronCmd.Flags().String("tz", "", "Timezone for scheduling and output (default: local)")
	cronCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	rootCmd.AddCommand(cronCmd)
}
