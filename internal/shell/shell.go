// Package shell installs ling-box's shell integration into the user's rc
// file: an `lb` alias and the generated tab-completion script.
//
// The integration is written as a marker-delimited block. Installing again
// (for example after an upgrade) rewrites that block in place instead of
// appending a second copy, and duplicate blocks left behind by older runs are
// collapsed into one. Everything outside the markers is left untouched.
package shell

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Supported shell names.
const (
	Bash = "bash"
	Zsh  = "zsh"
	Fish = "fish"
)

// Supported lists the shells Install can configure.
var Supported = []string{Bash, Zsh, Fish}

// Markers delimit the managed block in the rc file.
const (
	StartMarker = "# >>> lingbox shell integration >>>"
	EndMarker   = "# <<< lingbox shell integration <<<"
)

// Env looks up an environment variable; os.Getenv matches this signature.
type Env func(string) string

// IsSupported reports whether name is a shell this package knows about.
func IsSupported(name string) bool {
	for _, s := range Supported {
		if s == name {
			return true
		}
	}
	return false
}

// Detect returns the user's login shell from $SHELL.
func Detect(env Env) (string, error) {
	raw := strings.TrimSpace(env("SHELL"))
	if raw == "" {
		return "", fmt.Errorf("$SHELL is not set; pass --shell %s", strings.Join(Supported, "|"))
	}
	name := strings.TrimPrefix(filepath.Base(raw), "-") // login shells: "-bash"
	if !IsSupported(name) {
		return "", fmt.Errorf("unsupported shell %q; supported: %s (use --shell to choose)",
			name, strings.Join(Supported, ", "))
	}
	return name, nil
}

// RCPath returns the rc file that shell reads at startup.
func RCPath(shell string, env Env) (string, error) {
	home := strings.TrimSpace(env("HOME"))
	if home == "" {
		return "", errors.New("$HOME is not set")
	}
	switch shell {
	case Bash:
		bashrc := filepath.Join(home, ".bashrc")
		// macOS Terminal starts login shells, which read .bash_profile and
		// not .bashrc. Follow whichever file the user already has.
		if !fileExists(bashrc) && fileExists(filepath.Join(home, ".bash_profile")) {
			return filepath.Join(home, ".bash_profile"), nil
		}
		return bashrc, nil
	case Zsh:
		dir := strings.TrimSpace(env("ZDOTDIR"))
		if dir == "" {
			dir = home
		}
		return filepath.Join(dir, ".zshrc"), nil
	case Fish:
		dir := strings.TrimSpace(env("XDG_CONFIG_HOME"))
		if dir == "" {
			dir = filepath.Join(home, ".config")
		}
		return filepath.Join(dir, "fish", "config.fish"), nil
	}
	return "", fmt.Errorf("unsupported shell %q", shell)
}

// Block renders the managed snippet for shell using newline ("\n" or "\r\n")
// as the line separator.
func Block(shell, newline string) (string, error) {
	var body string
	switch shell {
	case Bash:
		body = "alias lb='lingbox'\n" +
			"if command -v lingbox >/dev/null 2>&1; then\n" +
			"    source <(lingbox completion bash)\n" +
			"fi\n"
	case Zsh:
		// compdef only exists once compinit has run; without this guard a
		// bare zsh would print "command not found: compdef" on every start.
		body = "alias lb='lingbox'\n" +
			"if command -v lingbox >/dev/null 2>&1 && (( $+functions[compdef] )); then\n" +
			"    source <(lingbox completion zsh)\n" +
			"fi\n"
	case Fish:
		body = "alias lb lingbox\n" +
			"if command -sq lingbox\n" +
			"    lingbox completion fish | source\n" +
			"end\n"
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
	const header = "# Managed by `lingbox shell install`; re-run it to update. Local edits are lost.\n"
	block := StartMarker + "\n" + header + body + EndMarker + "\n"
	if newline == "\r\n" {
		block = strings.ReplaceAll(block, "\n", "\r\n")
	}
	return block, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
