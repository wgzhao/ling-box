package cmd

import (
	"fmt"
	"math"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/datecalc"
)

var dateCmd = &cobra.Command{
	Use:   "date",
	Short: "Date calculation tool",
	Long: `Calculate dates: add/subtract days or find the difference between two dates.

Subcommands:
  add   Add or subtract days from a date
  diff  Calculate the difference between two dates

Examples:
  ling-box date add 2026-01-01 10      # Add 10 days
  ling-box date add 2026-01-01 -10     # Subtract 10 days
  ling-box date diff 2026-01-01 2026-01-11
  ling-box date diff "2026-01-01 12:00:00" "2026-01-02 14:30:00"`,
}

var dateAddCmd = &cobra.Command{
	Use:                "add",
	Short:              "Add or subtract days from a date",
	Long:               `Add positive days or subtract negative days from a given date.\n\nDate format: YYYY-MM-DD`,
	DisableFlagParsing: true,
	Args:               cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
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
	Use:   "diff",
	Short: "Calculate the difference between two dates",
	Long:  `Calculate the difference between two dates or datetimes.\n\nDate format: YYYY-MM-DD or YYYY-MM-DD HH:MM:SS`,
	Args:  cobra.ExactArgs(2),
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
