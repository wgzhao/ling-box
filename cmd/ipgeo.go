package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/qqwry"
	"golang.org/x/term"
)

var (
	ipgeoDBPath string
	ipgeoUpdate bool
	ipgeoJSON   bool
)

var ipgeoCmd = &cobra.Command{
	Use:   "ipgeo [options] <ip> [ip ...]",
	Short: "IP geolocation lookup (qqwry.dat)",
	Long: `ipgeo looks up the region and ISP of IPv4 addresses using the
qqwry.dat database (纯真 IP 库).

The database is downloaded on first use (about 27 MB) into the user
cache directory and reused offline afterwards:
  macOS:   ~/Library/Caches/ling-box/qqwry.dat
  Linux:   ~/.cache/ling-box/qqwry.dat
  Windows: %LocalAppData%\ling-box\qqwry.dat
Use --update to force a database refresh, or --db to point at a custom
database file.

Database source: https://github.com/metowolf/qqwry.dat (updated weekly)

Examples:
  lingbox ipgeo 8.8.8.8
  lingbox ipgeo 114.114.114.114 223.5.5.5
  lingbox ipgeo --json 8.8.8.8`,
	Args: cobra.MinimumNArgs(1),
	RunE: runIPGeo,
}

func init() {
	ipgeoCmd.Flags().StringVarP(&ipgeoDBPath, "db", "d", "", "database file path (default: qqwry.dat in the user cache directory)")
	ipgeoCmd.Flags().BoolVarP(&ipgeoUpdate, "update", "u", false, "force re-download of the database")
	ipgeoCmd.Flags().BoolVarP(&ipgeoJSON, "json", "j", false, "output as JSON")
	rootCmd.AddCommand(ipgeoCmd)
}

// geoResult pairs a queried IP with its record or the error that
// prevented the query.
type geoResult struct {
	IP  string
	Rec qqwry.Record
	Err error
}

func runIPGeo(cmd *cobra.Command, args []string) error {
	path := ipgeoDBPath
	if path == "" {
		var err error
		path, err = qqwry.DefaultPath()
		if err != nil {
			return err
		}
	}
	interactive := term.IsTerminal(int(os.Stderr.Fd()))
	_, err := qqwry.Ensure(path, ipgeoUpdate, nil, cmd.ErrOrStderr(), interactive)
	if err != nil {
		if _, statErr := os.Stat(path); statErr == nil {
			// keep using the existing database, warn about the failed update
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: database update failed: %v\n", err)
		} else {
			return fmt.Errorf("database download failed: %w\n(download qqwry.dat manually and use --db to specify its path)", err)
		}
	}
	db, err := qqwry.Open(path)
	if err != nil {
		return err
	}

	results := make([]geoResult, 0, len(args))
	for _, ip := range args {
		rec, err := db.QueryStr(ip)
		results = append(results, geoResult{IP: ip, Rec: rec, Err: err})
	}
	if ipgeoJSON {
		return renderGeoJSON(cmd, results)
	}
	return renderGeoTable(cmd, results)
}

func renderGeoJSON(cmd *cobra.Command, results []geoResult) error {
	type geoEntry struct {
		IP       string `json:"ip"`
		Country  string `json:"country"`
		Area     string `json:"area"`
		Location string `json:"location"`
	}
	entries := make([]geoEntry, 0, len(results))
	failed := 0
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", r.IP, r.Err)
			failed++
			continue
		}
		entries = append(entries, geoEntry{
			IP:       r.IP,
			Country:  r.Rec.Country,
			Area:     r.Rec.Area,
			Location: r.Rec.Location(),
		})
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	if err := enc.Encode(entries); err != nil {
		return err
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d ip(s) could not be resolved", failed, len(results))
	}
	return nil
}

func renderGeoTable(cmd *cobra.Command, results []geoResult) error {
	failed := 0
	var rows []string
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", r.IP, r.Err)
			failed++
			continue
		}
		rows = append(rows, fmt.Sprintf("%s\t%s", r.IP, r.Rec.Location()))
	}
	if len(rows) > 0 {
		out := cmd.OutOrStdout()
		tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "IP\tLocation")
		for _, row := range rows {
			fmt.Fprintln(tw, row)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d ip(s) could not be resolved", failed, len(results))
	}
	return nil
}
