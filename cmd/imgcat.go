package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/imgcat"
)

var imgcatCmd = &cobra.Command{
	Use:   "imgcat [flags] [image-file ...]",
	Short: "Display images in the terminal",
	Long: `Render images directly in the terminal.

Default renderer is iTerm2 (OSC 1337 protocol), which produces lossless
output and is supported by iTerm2, WezTerm, Warp, kaku, kitty (compat
mode), VS Code terminal, and many others.

For terminals that need an alternative:
  -r halfblock   ANSI true-color with ▀ block characters (universal)
  -r kitty       Kitty native graphics protocol
  -r ascii       Grayscale ASCII art (any terminal)

When multiple image files are given, enter interactive browse mode:
one image at a time, navigate with arrow keys or space (next) /
arrow keys (previous), q to quit. Files that can't be rendered are
silently skipped.

Glob patterns (*, ?, [...]) in file arguments are expanded automatically.
This is useful when the shell doesn't expand wildcards, e.g. quoted patterns.

Reads from a file path or from standard input (pipe).`,
	Example: `  lingbox imgcat photo.jpg
  lingbox imgcat photo.jpg -w 60
  lingbox imgcat photo.jpg --renderer iterm2
  lingbox imgcat photo.jpg --renderer halfblock
  lingbox imgcat photo.jpg --renderer ascii
  cat photo.png | lingbox imgcat
  lingbox imgcat photo1.jpg photo2.jpg photo3.png
  lingbox imgcat "875*.png"
  lingbox imgcat photo1.jpg "screenshot-*.png"`,
	Run: func(cmd *cobra.Command, args []string) {
		width, _ := cmd.Flags().GetInt("width")
		renderer, _ := cmd.Flags().GetString("renderer")

		opts := imgcat.Options{
			Width:    width,
			Renderer: imgcat.Renderer(renderer),
		}

		// Expand glob patterns (e.g. "875*.png") in positional args.
		args = expandGlobs(args)

		// Multiple files: interactive browse (invalid files skipped silently).
		if len(args) > 1 {
			if err := imgcat.Browse(cmd.OutOrStdout(), args, opts); err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error browsing images: %v\n", err)
			}
			return
		}

		// Single file or stdin: existing behavior.
		var data []byte
		var err error

		if len(args) > 0 {
			data, err = os.ReadFile(args[0])
		} else {
			data, err = readStdin()
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error reading stdin: %v\n", err)
				return
			}
		}
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error reading input: %v\n", err)
			return
		}
		if len(data) == 0 {
			fmt.Fprintln(cmd.OutOrStderr(), "Error: no image data provided")
			return
		}

		if err := imgcat.Display(cmd.OutOrStdout(), data, opts); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error displaying image: %v\n", err)
		}
	},
}

// expandGlobs expands glob patterns (*, ?, [...]) in args using filepath.Glob.
// Non-pattern arguments pass through unchanged. If a pattern matches nothing,
// the argument is left verbatim so Browse can try it (and skip if it fails).
func expandGlobs(args []string) []string {
	var expanded []string
	for _, arg := range args {
		if strings.ContainsAny(arg, "*?[") {
			matches, err := filepath.Glob(arg)
			if err != nil || len(matches) == 0 {
				expanded = append(expanded, arg)
				continue
			}
			expanded = append(expanded, matches...)
		} else {
			expanded = append(expanded, arg)
		}
	}
	return expanded
}

func init() {
	imgcatCmd.Flags().IntP("width", "w", 0, "Output width in character columns (default: auto-detect)")
	imgcatCmd.Flags().StringP("renderer", "r", "auto", "Renderer: auto, halfblock, iterm2, kitty, ascii")
	rootCmd.AddCommand(imgcatCmd)
}
