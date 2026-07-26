package cmd

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/datecalc"
)

var dateCmd = &cobra.Command{
	Use:   "date",
	Short: "Date calculation tool",
	Long: strings.TrimSpace(`
Calculate dates: add/subtract days or find the difference between two dates.

Subcommands:
  add   Add or subtract days from a date
  diff  Calculate the difference between two dates
`),
}

var dateAddCmd = &cobra.Command{
	Use:                "add <date> <days>",
	Short:              "Add or subtract days from a date",
	Long: `Add positive days or subtract negative days from a given date.

Use "--" before negative numbers.

Examples:
  lingboxdate add 2026-01-01 10
  lingboxdate add 2026-01-01 -- -30`,
	DisableFlagParsing: true,
	Args:               cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Handle --help manually since DisableFlagParsing disables built-in help
		for _, a := range args {
			if a == "--help" || a == "-h" {
				cmd.Help()
				return
			}
		}

		dateStr := args[0]
		days, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: invalid number of days %q\n", args[1])
			return
		}

		result, err := datecalc.AddDays(dateStr, days)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		sign := "+"
		if days < 0 {
			sign = ""
		}
		fmt.Printf("%s %s%d days = %s\n", dateStr, sign, days, result)
	},
}

var dateDiffCmd = &cobra.Command{
	Use:   "diff <date1> <date2>",
	Short: "Calculate the difference between two dates",
	Long: strings.TrimSpace(`
Calculate the difference between two dates or datetimes.

Formats: YYYY-MM-DD or YYYY-MM-DD HH:MM:SS

Examples:
  lingboxdate diff 2026-01-01 2026-07-26
  lingboxdate diff "2026-01-01 12:00:00" "2026-01-02 14:30:00"
`),
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		result, err := datecalc.Diff(args[0], args[1])
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
			return
		}

		absDays := int64(math.Abs(float64(result.Days)))
		absMins := int64(math.Abs(float64(result.Minutes)))
		absSecs := int64(math.Abs(float64(result.Seconds)))

		fmt.Printf("From  : %s\n", result.StartDate)
		fmt.Printf("To    : %s\n", result.EndDate)
		fmt.Printf("Days  : %d\n", absDays)
		fmt.Printf("Mins  : %d\n", absMins)
		fmt.Printf("Secs  : %d\n", absSecs)
	},
}

func init() {
	dateCmd.AddCommand(dateAddCmd)
	dateCmd.AddCommand(dateDiffCmd)
	rootCmd.AddCommand(dateCmd)
}
