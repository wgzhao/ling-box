package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options configure Install and Uninstall.
type Options struct {
	// Shell is "bash", "zsh" or "fish". Empty or "auto" detects it from $SHELL.
	Shell string
	// RC overrides the rc file to edit; empty uses the shell's default.
	RC string
	// DryRun computes the result without touching the file.
	DryRun bool
	// Env looks up environment variables; nil means os.Getenv.
	Env func(string) string
}

// Result describes what Install or Uninstall did, or would do with DryRun.
type Result struct {
	// Shell is the resolved shell name.
	Shell string
	// RC is the resolved rc file path.
	RC string
	// Created reports that the rc file did not exist before this run.
	Created bool
	// Changed reports that the rc file content was (or would be) modified.
	Changed bool
	// Content is the resulting rc file; filled on DryRun only.
	Content string
}

func (o Options) env() func(string) string {
	if o.Env != nil {
		return o.Env
	}
	return os.Getenv
}

func (o Options) resolve() (shellName, rc string, err error) {
	env := o.env()
	shellName = strings.ToLower(strings.TrimSpace(o.Shell))
	if shellName == "" || shellName == "auto" {
		shellName, err = Detect(env)
		if err != nil {
			return "", "", err
		}
	}
	if !IsSupported(shellName) {
		return "", "", fmt.Errorf("unsupported shell %q; supported: %s", shellName, strings.Join(Supported, ", "))
	}
	rc = strings.TrimSpace(o.RC)
	if rc == "" {
		rc, err = RCPath(shellName, env)
		return shellName, rc, err
	}
	rc, err = expandHome(rc, env)
	return shellName, rc, err
}

// Install writes or updates the managed block in the user's rc file.
func Install(o Options) (Result, error) {
	shellName, rc, err := o.resolve()
	if err != nil {
		return Result{}, err
	}

	content, mode, created, err := readRC(rc)
	if err != nil {
		return Result{}, err
	}
	block, err := Block(shellName, newlineOf(content))
	if err != nil {
		return Result{}, err
	}
	updated, changed, err := Apply(content, block)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", rc, err)
	}

	res := Result{Shell: shellName, RC: rc, Created: created, Changed: changed}
	if o.DryRun {
		res.Content = updated
		return res, nil
	}
	if changed {
		if err := writeRC(rc, []byte(updated), mode); err != nil {
			return res, err
		}
	}
	return res, nil
}

// Uninstall removes every managed block from the user's rc file.
func Uninstall(o Options) (Result, error) {
	shellName, rc, err := o.resolve()
	if err != nil {
		return Result{}, err
	}

	content, mode, created, err := readRC(rc)
	if err != nil {
		return Result{}, err
	}
	if created {
		return Result{Shell: shellName, RC: rc}, nil
	}
	updated, changed, err := Remove(content)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", rc, err)
	}

	res := Result{Shell: shellName, RC: rc, Changed: changed}
	if o.DryRun {
		res.Content = updated
		return res, nil
	}
	if changed {
		if err := writeRC(rc, []byte(updated), mode); err != nil {
			return res, err
		}
	}
	return res, nil
}

func readRC(path string) (content string, mode os.FileMode, created bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0o644, true, nil
		}
		return "", 0, false, fmt.Errorf("read %s: %w", path, err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		return "", 0, false, fmt.Errorf("stat %s: %w", path, err)
	}
	return string(data), fi.Mode().Perm(), false, nil
}

// writeRC replaces the file atomically, preserving its permissions. A
// symlinked rc file (common in dotfile setups) is written through, leaving
// the symlink itself intact.
func writeRC(path string, data []byte, mode os.FileMode) error {
	target := path
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			target = resolved
		}
	}
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".lingbox-rc-*")
	if err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", target, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	if err := os.Chmod(tmpName, mode.Perm()); err != nil {
		return fmt.Errorf("chmod %s: %w", target, err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	return nil
}

// expandHome expands a leading "~" in a user-supplied path.
func expandHome(path string, env Env) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home := env("HOME")
	if home == "" {
		return "", fmt.Errorf("cannot expand %q: $HOME is not set", path)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}
