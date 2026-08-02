package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/imgcat"
)

var imgcatCmd = &cobra.Command{
	Use:   "imgcat <image-file>",
	Short: "Display images in the terminal",
	Long: `Render images directly in the terminal.

Default renderer is iTerm2 (OSC 1337 protocol), which produces lossless
output and is supported by iTerm2, WezTerm, Warp, kaku, kitty (compat
mode), VS Code terminal, and many others.

For terminals that need an alternative:
  -r halfblock   ANSI true-color with ▀ block characters (universal)
  -r kitty       Kitty native graphics protocol
  -r ascii       Grayscale ASCII art (any terminal)

Reads from a file path or from standard input (pipe).`,
	Args: cobra.MaximumNArgs(1),
	Example: `  lingbox imgcat photo.jpg
  lingbox imgcat photo.png -w 60
  lingbox imgcat photo.jpg --renderer iterm2
  lingbox imgcat photo.jpg --renderer halfblock
  lingbox imgcat photo.jpg --renderer ascii
  cat photo.png | lingbox imgcat`,
	Run: func(cmd *cobra.Command, args []string) {
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

		width, _ := cmd.Flags().GetInt("width")
		renderer, _ := cmd.Flags().GetString("renderer")

		opts := imgcat.Options{
			Width:    width,
			Renderer: imgcat.Renderer(renderer),
		}

		if err := imgcat.Display(cmd.OutOrStdout(), data, opts); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error displaying image: %v\n", err)
		}
	},
}

func init() {
	imgcatCmd.Flags().IntP("width", "w", 0, "Output width in character columns (default: auto-detect)")
	imgcatCmd.Flags().StringP("renderer", "r", "auto", "Renderer: auto, halfblock, iterm2, kitty, ascii")
	rootCmd.AddCommand(imgcatCmd)
}
