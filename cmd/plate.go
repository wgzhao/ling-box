package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/plate"
	"golang.org/x/term"
)

var plateCmd = &cobra.Command{
	Use:   "plate [province]",
	Short: "车牌归属地查询",
	Long: `查询中国各省份车牌代码及其归属地区。

指定省份时,支持简称 (湘)、全称 (湖南省) 或不带行政后缀的名称 (湖南):
  lingbox plate 湘
  lingbox plate 湖南省
  lingbox plate 湖南

不指定省份时,分页打印全国 31 个省级行政区的车牌归属地:
  lingbox plate
按 Enter 查看下一页,q 退出;输出重定向到文件或管道时自动全部打印。
每页显示的省份数量可通过 --page-size 调整。`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pageSize, _ := cmd.Flags().GetInt("page-size")
		if pageSize < 1 {
			pageSize = 1
		}
		out := cmd.OutOrStdout()

		if len(args) == 1 {
			p, err := plate.Find(args[0])
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				return
			}
			printProvince(out, p)
			return
		}

		all := plate.All()
		// Interactive paging only when stdin is a terminal; piped or
		// redirected output prints everything in one pass.
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			for _, p := range all {
				printProvince(out, p)
			}
			return
		}
		pagedPrint(out, bufio.NewReader(os.Stdin), all, pageSize)
	},
}

func init() {
	plateCmd.Flags().IntP("page-size", "n", 5, "每个分页显示的省份数量")
	rootCmd.AddCommand(plateCmd)
}

// printProvince renders one province's plates:
//
//	湖南省 (湘)
//	  湘A  长沙市
//	  湘B  株洲市
func printProvince(w io.Writer, p *plate.Province) {
	fmt.Fprintf(w, "%s (%s)\n", p.Name, p.Short)
	for _, c := range p.Codes {
		fmt.Fprintf(w, "  %s%s\n", padRight(c.Code, 8), strings.Join(c.Districts, "、"))
	}
	fmt.Fprintln(w)
}

// pagedPrint prints provinces pageSize at a time, pausing between pages
// for Enter (next) or q (quit). The prompt goes to stderr so stdout stays
// a clean listing.
func pagedPrint(w io.Writer, in *bufio.Reader, provinces []*plate.Province, pageSize int) {
	total := len(provinces)
	pages := (total + pageSize - 1) / pageSize
	for page := 0; page < pages; page++ {
		start := page * pageSize
		end := start + pageSize
		if end > total {
			end = total
		}
		for _, p := range provinces[start:end] {
			printProvince(w, p)
		}
		if page == pages-1 {
			return
		}
		fmt.Fprintf(os.Stderr, "-- 第 %d/%d 页 (按 Enter 查看下一页, q 退出) --\n", page+1, pages)
		if !waitNext(in) {
			fmt.Fprintln(os.Stderr) // newline so the shell prompt starts fresh
			return
		}
	}
}

// waitNext waits for a single keypress: Enter advances the page, q quits.
// The terminal is switched to raw mode so q takes effect immediately
// without pressing Enter (less-style); line reading is the fallback when
// raw mode is unavailable.
func waitNext(in *bufio.Reader) bool {
	tty := os.Stdin
	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		line, _ := in.ReadString('\n')
		t := strings.TrimSpace(line)
		return t != "q" && t != "Q"
	}
	defer term.Restore(int(tty.Fd()), oldState)
	b := make([]byte, 1)
	if _, err := in.Read(b); err != nil {
		return false
	}
	// Any key advances; q/Q quits, and Ctrl-C (0x03) aborts since raw
	// mode intercepts the signal.
	return b[0] != 'q' && b[0] != 'Q' && b[0] != 0x03
}

// padRight pads s with spaces to the given display width. CJK characters
// count as two columns, so 湘A (3 runes) aligns with 粤Z as 5 columns.
func padRight(s string, width int) string {
	if w := displayWidth(s); w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-displayWidth(s))
}

// displayWidth returns the terminal display width of s: CJK characters
// count as two columns, everything else as one.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}
