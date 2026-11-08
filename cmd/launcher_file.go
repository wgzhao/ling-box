package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wgzhao/ling-box/internal/launcher"
)

var launcherFileCmd = &cobra.Command{
	Use:   "file [--] <path>...",
	Short: "List ling-box actions for selected files",
	Long: `List the actions that apply to the selected files. This is the
Script Filter behind the Alfred File Action and can be called directly to
debug it:

  lingbox launcher file -- report.dbf data.json`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetEscapeHTML(false)
		_ = enc.Encode(launcher.Response{Items: launcher.FileItems(args)})
	},
}

var launcherActionCmd = &cobra.Command{
	Use:    "action",
	Short:  "Run a file action",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		spec, _ := cmd.Flags().GetString("spec")
		msg, err := runFileAction(spec)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "ling-box: %v\n", err)
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), msg)
	},
}

func runFileAction(specJSON string) (string, error) {
	var spec launcher.ActionSpec
	if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
		return "", fmt.Errorf("invalid action: %w", err)
	}
	if spec.Path == "" {
		return "", errors.New("missing file path")
	}
	info, err := os.Stat(spec.Path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.New("directories are not supported")
	}

	switch spec.Action {
	case "dbf-info":
		out, err := execLingboxTimeout([]string{"dbf", "info", spec.Path}, "", 30*time.Second)
		if err != nil {
			return "", err
		}
		if err := copyText(out); err == nil {
			return "DBF info copied to clipboard", nil
		}
		return out, nil

	case "dbf-export":
		if spec.Format != "json" && spec.Format != "csv" {
			return "", fmt.Errorf("unsupported export format %q", spec.Format)
		}
		outPath := replaceExt(spec.Path, "."+spec.Format)
		args := []string{"dbf", "export", spec.Path, "-o", outPath}
		if spec.Format == "json" {
			args = append(args, "--json")
		}
		if err := execLingboxFile(args, 60*time.Second); err != nil {
			return "", err
		}
		reveal(outPath)
		return "Exported to " + outPath, nil

	case "convert":
		switch spec.Format {
		case "json", "yaml", "csv", "markdown":
		default:
			return "", fmt.Errorf("unsupported convert format %q", spec.Format)
		}
		outPath := replaceExt(spec.Path, targetExt(spec.Format))
		args := []string{"convert", "-i", spec.Path, "-o", outPath, "-t", spec.Format}
		if err := execLingboxFile(args, 60*time.Second); err != nil {
			return "", err
		}
		reveal(outPath)
		return "Converted to " + outPath, nil

	case "imgcat":
		return openInTerminal(spec.Path)
	}
	return "", fmt.Errorf("unknown action %q", spec.Action)
}

func replaceExt(path, newExt string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + newExt
}

func targetExt(format string) string {
	switch format {
	case "markdown":
		return ".md"
	case "yaml":
		return ".yaml"
	default:
		return "." + format
	}
}

// copyText puts text on the clipboard. The action is primarily an Alfred
// (macOS) feature; Linux support is a best-effort convenience.
func copyText(text string) error {
	if os.Getenv("LINGBOX_NO_GUI") != "" {
		return errors.New("clipboard disabled by LINGBOX_NO_GUI")
	}
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("pbcopy")
	case "linux":
		switch {
		case hasCommand("wl-copy"):
			c = exec.Command("wl-copy")
		case hasCommand("xclip"):
			c = exec.Command("xclip", "-selection", "clipboard")
		case hasCommand("xsel"):
			c = exec.Command("xsel", "--clipboard", "--input")
		default:
			return errors.New("no clipboard tool found")
		}
	default:
		return errors.New("clipboard not supported on this platform")
	}
	c.Stdin = strings.NewReader(text)
	return c.Run()
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// reveal shows the output file in the system file manager (macOS only).
func reveal(path string) {
	if runtime.GOOS != "darwin" || os.Getenv("LINGBOX_NO_GUI") != "" {
		return
	}
	_ = exec.Command("open", "-R", path).Run()
}

// openInTerminal launches Terminal with `lingbox imgcat`. Terminal.app does
// not understand the iTerm2 protocol, so force the portable half-block
// renderer.
func openInTerminal(path string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	command := shellQuote(exe) + " imgcat -r halfblock " + shellQuote(path)
	if runtime.GOOS != "darwin" || os.Getenv("LINGBOX_NO_GUI") != "" {
		return "Run: " + command, nil
	}

	f, err := os.CreateTemp("", "lingbox-imgcat-*.command")
	if err != nil {
		return "", err
	}
	name := f.Name()
	defer f.Close()
	if _, err := fmt.Fprintf(f, "#!/bin/sh\nexec %s\n", command); err != nil {
		return "", err
	}
	if err := os.Chmod(name, 0o755); err != nil {
		return "", err
	}
	if err := exec.Command("open", "-a", "Terminal", name).Run(); err != nil {
		return "", err
	}
	return "Opened in Terminal", nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func init() {
	launcherActionCmd.Flags().String("spec", "", "Action spec JSON")
	launcherCmd.AddCommand(launcherFileCmd)
	launcherCmd.AddCommand(launcherActionCmd)
}
