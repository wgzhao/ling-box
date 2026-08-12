package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/wgzhao/ling-box/internal/imgcat"
	"github.com/wgzhao/ling-box/internal/pdf"
)

var (
	pdfWidth    int
	pdfRenderer string
	pdfPage     int
	pdfDPI      float64
)

var pdfCmd = &cobra.Command{
	Use:   "pdf [flags] [file]",
	Short: "Display PDF files in the terminal",
	Long: `Render and display PDF files in the terminal using the same renderers
as imgcat (iTerm2 inline images, Kitty graphics protocol, half-block
ANSI art, or plain ASCII).

Without the --page flag, pdf enters interactive browse mode where you
can flip through pages with arrow keys:
  Space / Enter / Right / Down  — next page
  Left / Up                     — previous page
  q / Ctrl-C                    — quit`,
	Example: `  lingbox pdf document.pdf           # interactive page browsing
  lingbox pdf -p 1 document.pdf        # display first page only
  lingbox pdf -w 80 -r ascii doc.pdf   # 80-column ASCII art
  cat document.pdf | lingbox pdf       # read PDF from stdin (single page)`,
	Args: cobra.MaximumNArgs(1),
	Run:  runPDF,
}

func init() {
	pdfCmd.Flags().IntVarP(&pdfWidth, "width", "w", 0, "Output width in character columns (default: auto-detect)")
	pdfCmd.Flags().StringVarP(&pdfRenderer, "renderer", "r", "auto", "Renderer: auto, halfblock, iterm2, kitty, ascii")
	pdfCmd.Flags().IntVarP(&pdfPage, "page", "p", 0, "Specific page to render (1-indexed, default: interactive browse)")
	pdfCmd.Flags().Float64Var(&pdfDPI, "dpi", 150, "Render resolution in DPI")

	rootCmd.AddCommand(pdfCmd)
}

func runPDF(cmd *cobra.Command, args []string) {
	var data []byte
	var name string

	if len(args) == 1 {
		var err error
		data, err = os.ReadFile(args[0])
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error reading file: %v\n", err)
			return
		}
		name = filepath.Base(args[0])
	} else {
		var err error
		data, err = readStdin()
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error reading stdin: %v\n", err)
			return
		}
		if data == nil {
			cmd.Help()
			return
		}
		name = "stdin"
	}

	doc, err := pdf.OpenFromBytes(data)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "Error opening PDF: %v\n", err)
		return
	}
	defer doc.Close()

	opts := imgcat.Options{
		Width:    pdfWidth,
		Renderer: imgcat.Renderer(pdfRenderer),
	}

	// Single-page mode: render specified page and exit.
	if pdfPage > 0 {
		pageIdx := pdfPage - 1 // convert 1-indexed to 0-indexed
		if pageIdx >= doc.NumPages() {
			fmt.Fprintf(cmd.OutOrStderr(), "Error: page %d out of range (document has %d pages)\n",
				pdfPage, doc.NumPages())
			return
		}
		png, err := doc.RenderPage(pageIdx, pdfDPI)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error rendering page %d: %v\n", pdfPage, err)
			return
		}
		if err := imgcat.Display(cmd.OutOrStdout(), png, opts); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error displaying page: %v\n", err)
		}
		return
	}

	// Interactive browse mode.
	if err := browsePDF(cmd.OutOrStdout(), doc, name, opts); err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "Error browsing PDF: %v\n", err)
	}
}

// browsePDF renders each page interactively with keyboard navigation.
func browsePDF(w io.Writer, doc *pdf.Document, name string, opts imgcat.Options) error {
	total := doc.NumPages()

	tty, err := openPDFTTY()
	if err != nil {
		return err
	}
	defer tty.Close()

	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		return fmt.Errorf("set raw terminal mode: %w", err)
	}
	defer term.Restore(int(tty.Fd()), oldState)

	return browsePDFLoop(w, tty, doc, name, total, opts)
}

// openPDFTTY opens /dev/tty for keyboard input, falling back to stdin.
func openPDFTTY() (*os.File, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		return tty, nil
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return os.Stdin, nil
	}
	return nil, fmt.Errorf("open /dev/tty for keyboard input: %w", err)
}

// browsePDFLoop is the main interactive loop.
func browsePDFLoop(w io.Writer, r io.Reader, doc *pdf.Document, name string, total int, opts imgcat.Options) error {
	idx := 0 // 0-indexed current page

	for {
		// Clear screen.
		fmt.Fprint(w, "\x1b[2J\x1b[H")

		// Render current page.
		png, err := doc.RenderPage(idx, pdfDPI)
		if err != nil {
			fmt.Fprintf(w, "Error rendering page %d: %v\r\n", idx+1, err)
		} else {
			if err := imgcat.Display(w, png, opts); err != nil {
				fmt.Fprintf(w, "Error displaying page %d: %v\r\n", idx+1, err)
			}
		}

		// Status line.
		fmt.Fprintf(w, "\nPage %d/%d — %s  (← → arrows, q to quit)\r\n", idx+1, total, name)

		// Read navigation key.
		key := make([]byte, 3)
		n, err := r.Read(key)
		if err != nil {
			return nil
		}

		switch key[0] {
		case 'q', 'Q', '\x03':
			return nil
		case ' ', '\r', '\n':
			idx = (idx + 1) % total
		case '\x1b':
			if n >= 3 && key[1] == '[' {
				switch key[2] {
				case 'C', 'B': // right, down
					idx = (idx + 1) % total
				case 'D', 'A': // left, up
					idx = (idx - 1 + total) % total
				}
			}
		}
	}
}
