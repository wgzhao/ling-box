package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/idcard"
)

var idcardCmd = &cobra.Command{
	Use:   "idcard",
	Short: "Chinese ID card validation and generation",
	Long: `Validate Chinese resident identity card numbers, or generate synthetic
numbers for form testing.

Subcommands:
  validate  Check a 15- or 18-digit ID (checksum, region, birth date, gender)
  generate  Create valid test numbers with a chosen region, birth date, gender

Generated numbers are synthetic and intended for testing only. They may
coincidentally match a real person's number; do not use them as a real
identity.`,
}

var idcardValidateCmd = &cobra.Command{
	Use:     "validate <number>",
	Aliases: []string{"check", "verify"},
	Short:   "Validate a Chinese ID card number",
	Long: `Check a 15- or 18-digit Chinese ID card number.

Validation covers the format, birth date, gender, checksum, and region code.
Unknown region codes are warnings by default because older IDs may use
historical divisions; use --strict to reject them.

Examples:
  lingbox idcard validate 11010519491231002X
  lingbox idcard validate 110105491231002
  lingbox idcard validate --strict 11010519491231002X
  lingbox idcard validate --json 11010519491231002X`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		strict, _ := cmd.Flags().GetBool("strict")
		asJSON, _ := cmd.Flags().GetBool("json")

		info := idcard.Validate(args[0], idcard.Options{StrictRegion: strict})
		text := formatIDCardInfo(info)
		copyValue := info.Number
		if !info.Valid || copyValue == "" {
			copyValue = strings.TrimSpace(info.Input)
		}

		if asJSON {
			_ = writeResult(cmd.OutOrStdout(), "idcard", text, copyValue, info)
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), text)
		if !info.Valid {
			os.Exit(1)
		}
	},
}

var idcardGenerateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "Generate synthetic Chinese ID card numbers",
	Long: `Generate Chinese ID card numbers with valid checksums for testing.

The default region is Beijing Chaoyang (110105) and the default birth date is
1990-01-01. The sequence digits are random, with the last digit matching the
requested gender.

Examples:
  lingbox idcard generate
  lingbox idcard generate --region 310104 --birth 1985-06-15 --gender female
  lingbox idcard generate -n 10 --gender any
  lingbox idcard generate --json`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		region, _ := cmd.Flags().GetString("region")
		birthRaw, _ := cmd.Flags().GetString("birth")
		gender, _ := cmd.Flags().GetString("gender")
		count, _ := cmd.Flags().GetInt("count")
		asJSON, _ := cmd.Flags().GetBool("json")

		var birth time.Time
		if birthRaw != "" {
			var err error
			birth, err = time.Parse("2006-01-02", birthRaw)
			if err != nil {
				reportIDCardError(cmd, asJSON, fmt.Errorf("invalid birth date %q (expected YYYY-MM-DD)", birthRaw))
				return
			}
		}

		items, err := idcard.Generate(idcard.GenerateOptions{
			Region: region,
			Birth:  birth,
			Gender: gender,
			Count:  count,
		})
		if err != nil {
			reportIDCardError(cmd, asJSON, err)
			return
		}

		numbers := make([]string, len(items))
		for i, item := range items {
			numbers[i] = item.Number
		}
		text := strings.Join(numbers, "\n")

		if asJSON {
			extra := struct {
				Count int                `json:"count"`
				Items []idcard.Generated `json:"items"`
			}{Count: len(items), Items: items}
			_ = writeResult(cmd.OutOrStdout(), "idcard", text, text, extra)
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), text)
	},
}

func formatIDCardInfo(info idcard.Info) string {
	valid := "no"
	if info.Valid {
		valid = "yes"
	}
	lines := []string{"Valid        : " + valid}
	if info.Format != "" {
		lines = append(lines, "Format       : "+info.Format)
	}
	if info.Number != "" {
		lines = append(lines, "Number       : "+info.Number)
	}
	if info.RegionCode != "" {
		region := info.RegionName
		if region == "" {
			region = "unknown"
		}
		lines = append(lines, fmt.Sprintf("Region       : %s (%s)", region, info.RegionCode))
	}
	if info.BirthDate != "" {
		lines = append(lines, "Birth date   : "+info.BirthDate)
	}
	if info.Gender != "" {
		lines = append(lines, "Gender       : "+info.Gender)
	}
	if info.Sequence != "" {
		lines = append(lines, "Sequence     : "+info.Sequence)
	}
	if info.CheckDigit != "" {
		checksum := "ok"
		if info.ExpectedCheckDigit != "" && info.CheckDigit != info.ExpectedCheckDigit {
			checksum = fmt.Sprintf("mismatch: expected %s, got %s", info.ExpectedCheckDigit, info.CheckDigit)
		}
		lines = append(lines, "Checksum     : "+checksum)
	}
	if len(info.Errors) > 0 {
		lines = append(lines, "Problems     :")
		for _, e := range info.Errors {
			lines = append(lines, "  - "+e)
		}
	}
	if len(info.Warnings) > 0 {
		lines = append(lines, "Warnings     :")
		for _, w := range info.Warnings {
			lines = append(lines, "  - "+w)
		}
	}
	return strings.Join(lines, "\n")
}

func reportIDCardError(cmd *cobra.Command, asJSON bool, err error) {
	if asJSON {
		_ = writeErrorResult(cmd.OutOrStdout(), "idcard", err)
		return
	}
	fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
}

func init() {
	idcardValidateCmd.Flags().Bool("strict", false, "Treat unknown region codes as invalid")
	idcardValidateCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")

	idcardGenerateCmd.Flags().StringP("region", "r", idcard.DefaultRegion, "6-digit county-level region code")
	idcardGenerateCmd.Flags().StringP("birth", "b", "", "Birth date YYYY-MM-DD (default: 1990-01-01)")
	idcardGenerateCmd.Flags().StringP("gender", "g", "any", "Gender: male, female, or any")
	idcardGenerateCmd.Flags().IntP("count", "n", 1, "Number of IDs to generate")
	idcardGenerateCmd.Flags().Bool("json", false, "Output a machine-readable JSON result envelope")

	idcardCmd.AddCommand(idcardValidateCmd)
	idcardCmd.AddCommand(idcardGenerateCmd)
	rootCmd.AddCommand(idcardCmd)
}
