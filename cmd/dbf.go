package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/dbf"
)

// maxCellWidth caps view cells so long memo values do not explode the
// table; longer values are truncated with an ellipsis.
const maxCellWidth = 60

var dbfCmd = &cobra.Command{
	Use:   "dbf",
	Short: "dBase/FoxPro DBF table reader",
	Long: `Read dBase/FoxPro database table files (.dbf), including the common
field types (C/N/F/D/L/M/I/T/Y/B) and memo files (.fpt/.dbt).

Subcommands:
  info    Show the file header, encoding, and fields
  view    Browse records as an aligned table
  export  Export records as CSV or JSON

Text encoding is auto-detected from the language driver ID, defaulting
to GBK when unset; override with -e. Tables with the encryption flag
set are refused. Microsoft Access (.mdb/.accdb) is a different format
and is not supported.

Examples:
  lingbox dbf info customers.dbf
  lingbox dbf view customers.dbf -n 10
  lingbox dbf view customers.dbf -e gbk --include-deleted
  lingbox dbf export customers.dbf -o customers.csv
  lingbox dbf export customers.dbf --json`,
	Args: cobra.NoArgs,
}

var dbfInfoCmd = &cobra.Command{
	Use:   "info <file>",
	Short: "Show the DBF file header and fields",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		enc, _ := cmd.Flags().GetString("encoding")
		tbl, err := dbf.OpenWithEncoding(args[0], enc)
		if err != nil {
			return err
		}
		defer tbl.Close()
		h := tbl.Header
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "File:    %s\n", args[0])
		fmt.Fprintf(out, "Version: %s (0x%02X)\n", h.VersionName, h.Version)
		if h.LastUpdate != "" {
			fmt.Fprintf(out, "Updated: %s\n", h.LastUpdate)
		}
		fmt.Fprintf(out, "Records: %d\n", h.RecordCount)
		fmt.Fprintf(out, "Header:  %d bytes\n", h.HeaderSize)
		fmt.Fprintf(out, "Record:  %d bytes\n", h.RecordSize)
		if h.CodePage == "gbk" && h.LanguageID != 0x4D && h.LanguageID != 0x7A {
			fmt.Fprintf(out, "Encoding: %s (language driver 0x%02X, unknown, default)\n", h.CodePage, h.LanguageID)
		} else {
			fmt.Fprintf(out, "Encoding: %s (language driver 0x%02X)\n", h.CodePage, h.LanguageID)
		}
		if h.HasMemo {
			if h.MemoFile != "" {
				fmt.Fprintf(out, "Memo:    %s\n", h.MemoFile)
			} else {
				fmt.Fprintln(out, "Memo:    M fields present, but no .fpt/.dbt file found")
			}
		}

		fmt.Fprintln(out)
		fmt.Fprintf(out, "Fields (%d):\n", len(h.Fields))
		fmt.Fprintln(out, "Name            Type  Length  Dec  Description")
		for _, f := range h.Fields {
			fmt.Fprintf(out, "%-16s %-4c %5d  %-3d %s\n",
				f.Name, f.Type, f.Length, f.Decimal, dbf.FieldTypeName(f.Type))
		}
		return nil
	},
}

var dbfViewCmd = &cobra.Command{
	Use:   "view <file>",
	Short: "Browse records as an aligned table",
	Long: `Display records as a table, with wide (CJK) characters counted as
two columns for alignment.

Records marked as deleted are hidden by default; --include-deleted
shows them with a leading *. Cells are capped at 60 columns, with
longer values (e.g. memos) truncated.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		enc, _ := cmd.Flags().GetString("encoding")
		limit, _ := cmd.Flags().GetInt("limit")
		inclDeleted, _ := cmd.Flags().GetBool("include-deleted")
		noMemo, _ := cmd.Flags().GetBool("no-memo")

		tbl, err := dbf.OpenWithEncoding(args[0], enc)
		if err != nil {
			return err
		}
		defer tbl.Close()
		recs, err := tbl.Rows(!noMemo)
		if err != nil {
			return err
		}
		h := tbl.Header

		var rows []dbf.Record
		memoErrs := 0
		for _, r := range recs {
			if r.Deleted && !inclDeleted {
				continue
			}
			if r.MemoErr != "" {
				memoErrs++
			}
			rows = append(rows, r)
		}
		if limit > 0 && len(rows) > limit {
			rows = rows[:limit]
		}

		out := cmd.OutOrStdout()
		widths := make([]int, len(h.Fields))
		for i, f := range h.Fields {
			widths[i] = displayWidth(f.Name)
		}
		for _, r := range rows {
			for i, v := range r.Values {
				if w := displayWidth(v); w > widths[i] {
					widths[i] = w
				}
			}
		}
		for i := range widths {
			if widths[i] > maxCellWidth {
				widths[i] = maxCellWidth
			}
		}

		for i, f := range h.Fields {
			fmt.Fprint(out, padRight(f.Name, widths[i]+1))
		}
		fmt.Fprintln(out)
		for i := range widths {
			fmt.Fprint(out, strings.Repeat("-", widths[i]+1))
		}
		fmt.Fprintln(out)
		for _, r := range rows {
			for i, v := range r.Values {
				w := widths[i] + 1
				if r.Deleted && i == 0 {
					// The delete marker fills the first column's gap.
					fmt.Fprint(out, "*")
					w = widths[i]
				}
				fmt.Fprint(out, padRight(truncate(v, widths[i]), w))
			}
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "%d record(s)\n", len(rows))
		if memoErrs > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "note: %d record(s) have unreadable memo fields (missing .fpt/.dbt?)\n", memoErrs)
		}
		return nil
	},
}

var dbfExportCmd = &cobra.Command{
	Use:   "export <file>",
	Short: "Export records as CSV or JSON",
	Long: `Export the records. The default output is CSV (first row is the
field names); --json outputs typed JSON (numeric fields as numbers,
logical fields as booleans, empty values as null).

Deleted records are skipped by default; use --include-deleted to
include them. CSV goes to the -o file, or stdout when -o is omitted
or "-".`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		enc, _ := cmd.Flags().GetString("encoding")
		output, _ := cmd.Flags().GetString("output")
		asJSON, _ := cmd.Flags().GetBool("json")
		inclDeleted, _ := cmd.Flags().GetBool("include-deleted")

		tbl, err := dbf.OpenWithEncoding(args[0], enc)
		if err != nil {
			return err
		}
		defer tbl.Close()
		recs, err := tbl.Rows(true)
		if err != nil {
			return err
		}
		h := tbl.Header
		var rows []dbf.Record
		for _, r := range recs {
			if !r.Deleted || inclDeleted {
				rows = append(rows, r)
			}
		}

		var data []byte
		if asJSON {
			list := make([]map[string]interface{}, 0, len(rows))
			for _, r := range rows {
				m := make(map[string]interface{}, len(h.Fields))
				for i, f := range h.Fields {
					m[f.Name] = dbf.JSONValue(f, r.Values[i])
				}
				list = append(list, m)
			}
			data, err = json.MarshalIndent(list, "", "  ")
			if err != nil {
				return err
			}
			data = append(data, '\n')
		} else {
			var buf strings.Builder
			w := csv.NewWriter(&buf)
			names := make([]string, len(h.Fields))
			for i, f := range h.Fields {
				names[i] = f.Name
			}
			if err := w.Write(names); err != nil {
				return err
			}
			for _, r := range rows {
				if err := w.Write(r.Values); err != nil {
					return err
				}
			}
			w.Flush()
			if err := w.Error(); err != nil {
				return err
			}
			data = []byte(buf.String())
		}

		if output == "" || output == "-" {
			_, err = cmd.OutOrStdout().Write(data)
			return err
		}
		return os.WriteFile(output, data, 0o644)
	},
}

// truncate shortens s so its display width fits width, appending an
// ellipsis when cut.
func truncate(s string, width int) string {
	if displayWidth(s) <= width {
		return s
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := displayWidth(string(r))
		if w+rw > width-1 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + "…"
}

func init() {
	encHelp := "text encoding (" + dbf.EncodingNames() + "; default auto-detect from the language driver)"
	dbfInfoCmd.Flags().StringP("encoding", "e", "auto", encHelp)
	dbfViewCmd.Flags().StringP("encoding", "e", "auto", encHelp)
	dbfViewCmd.Flags().IntP("limit", "n", 0, "show at most N records (0 = all)")
	dbfViewCmd.Flags().Bool("include-deleted", false, "include deleted records (marked with *)")
	dbfViewCmd.Flags().Bool("no-memo", false, "show raw memo pointers instead of resolving")
	dbfExportCmd.Flags().StringP("encoding", "e", "auto", encHelp)
	dbfExportCmd.Flags().StringP("output", "o", "", "output file (default or -: stdout)")
	dbfExportCmd.Flags().Bool("json", false, "output JSON instead of CSV")
	dbfExportCmd.Flags().Bool("include-deleted", false, "include deleted records")
	dbfCmd.AddCommand(dbfInfoCmd)
	dbfCmd.AddCommand(dbfViewCmd)
	dbfCmd.AddCommand(dbfExportCmd)
	rootCmd.AddCommand(dbfCmd)
}
