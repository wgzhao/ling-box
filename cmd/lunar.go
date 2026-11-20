package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/lunar"
)

var dateLunarCmd = &cobra.Command{
	Use:     "lunar [solar-date]",
	Aliases: []string{"nongli"},
	Short:   "Chinese lunar calendar converter",
	Long: `Convert between Gregorian (solar) and Chinese lunar dates.

With no argument, today's solar date is converted. Use --lunar to convert a
lunar date back to solar, --leap for a leap month, and --terms to list the 24
solar terms of a year. Dates use YYYY-MM-DD; years 1900-2100 are supported.

Examples:
  lingbox date lunar 2025-01-29
  lingbox date lunar --lunar 2025-01-01
  lingbox date lunar --lunar 2025-06-01 --leap
  lingbox date lunar --terms --year 2025
  lingbox date lunar --json 2025-01-29`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		lunarFlag, _ := cmd.Flags().GetBool("lunar")
		leap, _ := cmd.Flags().GetBool("leap")
		termsFlag, _ := cmd.Flags().GetBool("terms")
		year, _ := cmd.Flags().GetInt("year")
		asJSON, _ := cmd.Flags().GetBool("json")

		if termsFlag {
			if len(args) > 0 || lunarFlag || leap {
				reportLunarError(cmd, asJSON, fmt.Errorf("--terms cannot be combined with a date, --lunar, or --leap"))
				return
			}
			if year == 0 {
				year = time.Now().Year()
			}
			terms, err := lunar.Terms(year)
			if err != nil {
				reportLunarError(cmd, asJSON, err)
				return
			}
			lines := make([]string, 0, len(terms)+1)
			lines = append(lines, fmt.Sprintf("Year        : %d", year))
			for _, term := range terms {
				lines = append(lines, term.Date+"  "+term.Name)
			}
			text := strings.Join(lines, "\n")
			if asJSON {
				extra := struct {
					Year  int          `json:"year"`
					Terms []lunar.Term `json:"terms"`
				}{Year: year, Terms: terms}
				_ = writeResult(cmd.OutOrStdout(), "lunar", text, text, extra)
				return
			}
			fmt.Fprintln(cmd.OutOrStdout(), text)
			return
		}

		if year != 0 {
			reportLunarError(cmd, asJSON, fmt.Errorf("--year requires --terms"))
			return
		}

		var (
			info      lunar.DateInfo
			copyValue string
			err       error
		)

		if lunarFlag {
			if len(args) != 1 {
				reportLunarError(cmd, asJSON, fmt.Errorf("--lunar requires a date (YYYY-MM-DD)"))
				return
			}
			y, m, d, parseErr := parseYMD(args[0])
			if parseErr != nil {
				reportLunarError(cmd, asJSON, parseErr)
				return
			}
			info, err = lunar.FromLunar(y, m, d, leap)
			copyValue = info.Solar
		} else {
			if leap {
				reportLunarError(cmd, asJSON, fmt.Errorf("--leap requires --lunar"))
				return
			}
			var y, m, d int
			if len(args) == 1 {
				y, m, d, err = parseYMD(args[0])
			} else {
				now := time.Now()
				y, m, d = now.Year(), int(now.Month()), now.Day()
			}
			if err == nil {
				info, err = lunar.FromSolar(y, m, d)
				copyValue = info.Lunar
			}
		}
		if err != nil {
			reportLunarError(cmd, asJSON, err)
			return
		}

		text := formatLunarInfo(info)
		if asJSON {
			_ = writeResult(cmd.OutOrStdout(), "lunar", text, copyValue, info)
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), text)
	},
}

func formatLunarInfo(info lunar.DateInfo) string {
	lines := []string{
		fmt.Sprintf("Solar        : %s (%s)", info.Solar, info.SolarWeekday),
		fmt.Sprintf("Lunar        : %s", info.Lunar),
		fmt.Sprintf("Zodiac       : %s", info.Zodiac),
		fmt.Sprintf("GanZhi year  : %s", info.GanZhiYear),
		fmt.Sprintf("GanZhi month : %s", info.GanZhiMonth),
		fmt.Sprintf("GanZhi day   : %s", info.GanZhiDay),
	}
	if info.SolarTerm != "" {
		lines = append(lines, fmt.Sprintf("Solar term   : %s", info.SolarTerm))
	}
	return strings.Join(lines, "\n")
}

func parseYMD(s string) (int, int, int, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", s)
	}
	return t.Year(), int(t.Month()), t.Day(), nil
}

func reportLunarError(cmd *cobra.Command, asJSON bool, err error) {
	if asJSON {
		_ = writeErrorResult(cmd.OutOrStdout(), "lunar", err)
		return
	}
	fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
}

func init() {
	dateLunarCmd.Flags().Bool("lunar", false, "Treat the date as lunar and convert it to solar")
	dateLunarCmd.Flags().Bool("leap", false, "Treat the --lunar month as a leap month")
	dateLunarCmd.Flags().Bool("terms", false, "List the 24 solar terms")
	dateLunarCmd.Flags().Int("year", 0, "Year for --terms (default: current year)")
	dateLunarCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")
	dateCmd.AddCommand(dateLunarCmd)
}
