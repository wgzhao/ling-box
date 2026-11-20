package e2e

// End-to-end coverage for `lingbox shell install`, the feature that adds an
// `lb` alias and tab completion to the user's rc file.
//
// The tests drive the real binary against throwaway HOME directories and then
// run a real bash (and zsh/fish when available) over the resulting rc file, so
// the assertions cover the whole chain: rc editing, shell startup, alias
// resolution and the completion protocol. Every scenario leaves its inputs and
// outputs in test/e2e/artifacts so a failure replays from the artifacts alone.

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	shellStart = "# >>> lingbox shell integration >>>"
	shellEnd   = "# <<< lingbox shell integration <<<"
)

type shellFixture struct {
	home string
	env  map[string]string
	rc   string
}

// newShellFixture builds an isolated HOME with a copy of the binary on PATH,
// so the rc snippet's `command -v lingbox` guard succeeds in a real shell.
func newShellFixture(t *testing.T) *shellFixture {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell rc integration is Unix-only")
	}
	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read built binary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "lingbox"), data, 0o755); err != nil {
		t.Fatalf("copy binary: %v", err)
	}
	return &shellFixture{
		home: home,
		rc:   filepath.Join(home, ".bashrc"),
		env: map[string]string{
			"HOME":            home,
			"PATH":            binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
			"SHELL":           "/bin/bash",
			"XDG_CONFIG_HOME": filepath.Join(home, ".config"),
			"ZDOTDIR":         "",
		},
	}
}

func mergeEnv(overrides map[string]string) []string {
	env := make([]string, 0, len(os.Environ())+len(overrides))
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if _, ok := overrides[key]; ok {
			continue
		}
		env = append(env, kv)
	}
	for k, v := range overrides {
		env = append(env, k+"="+v)
	}
	return env
}

// runProcess runs an arbitrary command with the given env overrides.
func runProcess(t *testing.T, name string, env map[string]string, args ...string) (string, string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(mergeEnv(env), "NO_COLOR=1", "TERM=dumb", "LINGBOX_NO_GUI=1")
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("run %s %v: %v", name, args, err)
		}
	}
	return out.String(), errOut.String(), code
}

func (f *shellFixture) run(t *testing.T, extra map[string]string, args ...string) (string, string, int) {
	t.Helper()
	env := make(map[string]string, len(f.env)+len(extra))
	for k, v := range f.env {
		env[k] = v
	}
	for k, v := range extra {
		env[k] = v
	}
	return runProcess(t, binPath, env, args...)
}

func (f *shellFixture) writeRC(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(f.rc, []byte(content), 0o644); err != nil {
		t.Fatalf("write rc: %v", err)
	}
}

func (f *shellFixture) readRC(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(f.rc)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	return string(data)
}

func assertSingleBlock(t *testing.T, rc string) {
	t.Helper()
	if got := strings.Count(rc, shellStart); got != 1 {
		t.Fatalf("start marker count = %d, want 1\n%s", got, rc)
	}
	if got := strings.Count(rc, shellEnd); got != 1 {
		t.Fatalf("end marker count = %d, want 1\n%s", got, rc)
	}
}

const userRC = "export EDITOR=vim\nalias ll='ls -l'\n"

func TestShellInstallIdempotent(t *testing.T) {
	f := newShellFixture(t)
	f.writeRC(t, userRC)

	out, errOut, code := f.run(t, nil, "shell", "install")
	if code != 0 {
		t.Fatalf("install exit %d, stderr: %s", code, errOut)
	}
	if !strings.Contains(out, "shell integration") {
		t.Fatalf("install stdout missing notice: %q", out)
	}
	first := f.readRC(t)
	saveArtifact(t, "shell-bashrc-installed", ".sh", first)

	assertSingleBlock(t, first)
	for _, want := range []string{
		userRC,
		"alias lb='lingbox'",
		"source <(lingbox completion bash)",
	} {
		if !strings.Contains(first, want) {
			t.Fatalf("rc missing %q\n%s", want, first)
		}
	}
	if !strings.HasPrefix(first, strings.TrimSuffix(userRC, "\n")) {
		t.Fatalf("user configuration was not preserved at the top:\n%s", first)
	}

	// A second run (an upgrade) must not change a byte.
	out, errOut, code = f.run(t, nil, "shell", "install")
	if code != 0 {
		t.Fatalf("reinstall exit %d, stderr: %s", code, errOut)
	}
	if !strings.Contains(out, "already up to date") {
		t.Fatalf("reinstall stdout = %q, want up-to-date notice", out)
	}
	if second := f.readRC(t); second != first {
		t.Fatalf("reinstall changed the rc file:\n--- before\n%s\n--- after\n%s", first, second)
	}
	record(t, "shell install idempotent", filepath.Base(f.rc))
}

func TestShellUpgradeReplacesBlockInPlace(t *testing.T) {
	f := newShellFixture(t)
	legacy := strings.Join([]string{
		"# user top",
		"export EDITOR=vim",
		"",
		shellStart,
		"alias lb='lingbox'",
		"source <(lingbox completion bash 2>/dev/null)",
		shellEnd,
		"",
		"# user middle",
		"export FOO=bar",
		shellStart,
		"alias lb=old-lingbox",
		shellEnd,
		"# user bottom",
		"",
	}, "\n")
	f.writeRC(t, legacy)

	// --dry-run previews the rewrite without touching the file.
	out, errOut, code := f.run(t, nil, "shell", "install", "--dry-run")
	if code != 0 {
		t.Fatalf("dry-run exit %d, stderr: %s", code, errOut)
	}
	if got := f.readRC(t); got != legacy {
		t.Fatalf("dry-run modified the rc file:\n%s", got)
	}
	if !strings.Contains(out, "Managed by") || strings.Contains(out, "old-lingbox") {
		t.Fatalf("dry-run does not preview the replacement:\n%s", out)
	}
	saveArtifact(t, "shell-dryrun", ".txt", out)

	_, errOut, code = f.run(t, nil, "shell", "install")
	if code != 0 {
		t.Fatalf("install exit %d, stderr: %s", code, errOut)
	}
	got := f.readRC(t)
	saveArtifact(t, "shell-bashrc-upgraded", ".sh", got)

	assertSingleBlock(t, got)
	for _, stale := range []string{"old-lingbox", "2>/dev/null"} {
		if strings.Contains(got, stale) {
			t.Fatalf("stale block content %q survived:\n%s", stale, got)
		}
	}
	// The surviving block stays where the first one was: not appended at EOF.
	top := strings.Index(got, "# user top")
	start := strings.Index(got, shellStart)
	middle := strings.Index(got, "# user middle")
	bottom := strings.Index(got, "# user bottom")
	if !(top < start && start < middle && middle < bottom) {
		t.Fatalf("block moved out of place: top=%d start=%d middle=%d bottom=%d\n%s",
			top, start, middle, bottom, got)
	}
	if !strings.Contains(got, "alias lb='lingbox'") {
		t.Fatalf("new block missing alias:\n%s", got)
	}

	// Running again after the in-place upgrade is a no-op.
	before := got
	if _, _, code := f.run(t, nil, "shell", "install"); code != 0 {
		t.Fatalf("post-upgrade install exit %d", code)
	}
	if after := f.readRC(t); after != before {
		t.Fatalf("upgrade rerun changed the rc file")
	}
	record(t, "shell upgrade in place", "dedup + position preserved")
}

func TestShellBashSession(t *testing.T) {
	f := newShellFixture(t)
	f.writeRC(t, userRC)
	if _, errOut, code := f.run(t, nil, "shell", "install"); code != 0 {
		t.Fatalf("install exit %d, stderr: %s", code, errOut)
	}

	// Simulate a real tab completion. The shim stands in for the
	// bash-completion package's _get_comp_words_by_ref helper, which is not
	// installed on every system; the generated script falls back to it.
	const script = `
_get_comp_words_by_ref() { words=("${COMP_WORDS[@]}"); cword=$COMP_CWORD; cur=${COMP_WORDS[COMP_CWORD]}; prev=${COMP_WORDS[COMP_CWORD-1]}; }
printf 'TYPE:%s\n' "$(type -t lb)"
printf 'ALIAS:%s\n' "$(alias lb)"
printf 'COMPLETE:%s\n' "$(complete -p lingbox)"
COMP_WORDS=(lingbox uu); COMP_CWORD=1
__start_lingbox 2>/dev/null
printf 'REPLY:%s\n' "${COMPREPLY[@]}"
lb --version
`
	out, errOut, code := runProcess(t, "bash", f.env, "--rcfile", f.rc, "-i", "-c", script)
	transcript := "$ bash --rcfile " + f.rc + " -i -c '<session script>'\n" + out + "--- stderr ---\n" + errOut
	saveArtifact(t, "shell-bash-session", ".txt", transcript)
	if code != 0 {
		t.Fatalf("bash session exit %d\n%s", code, transcript)
	}
	for _, want := range []string{
		"TYPE:alias",
		"ALIAS:alias lb='lingbox'",
		"COMPLETE:complete -o default -F __start_lingbox lingbox",
		"REPLY:uuid",
		"lingbox version",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("bash session missing %q:\n%s", want, transcript)
		}
	}
	record(t, "shell bash session", "alias lb + completion(uu -> uuid)")

	// The same checks for zsh/fish when they happen to be installed.
	if _, err := exec.LookPath("zsh"); err == nil {
		zshRC := filepath.Join(f.home, ".zshrc")
		if _, errOut, code := f.run(t, map[string]string{"SHELL": "/usr/bin/zsh"}, "shell", "install"); code != 0 {
			t.Fatalf("zsh install exit %d, stderr: %s", code, errOut)
		}
		zshScript := `compdef() { print -r -- "COMPDEF:$*" }
source ` + zshRC + `
alias lb`
		out, errOut, code := runProcess(t, "zsh", f.env, "-f", "-c", zshScript)
		saveArtifact(t, "shell-zsh-session", ".txt", out+"--- stderr ---\n"+errOut)
		if code != 0 || !strings.Contains(out, "COMPDEF:_lingbox lingbox") || !strings.Contains(out, "lb='lingbox'") {
			t.Fatalf("zsh session exit %d:\n%s\n%s", code, out, errOut)
		}
		record(t, "shell zsh session", "compdef registered")
	}
	if _, err := exec.LookPath("fish"); err == nil {
		fishRC := filepath.Join(f.home, ".config", "fish", "config.fish")
		if _, errOut, code := f.run(t, map[string]string{"SHELL": "/usr/bin/fish"}, "shell", "install"); code != 0 {
			t.Fatalf("fish install exit %d, stderr: %s", code, errOut)
		}
		fishScript := "source " + fishRC + "; functions -q lb; and echo ALIAS_OK; complete -c lingbox"
		out, errOut, code := runProcess(t, "fish", f.env, "-c", fishScript)
		saveArtifact(t, "shell-fish-session", ".txt", out+"--- stderr ---\n"+errOut)
		if code != 0 || !strings.Contains(out, "ALIAS_OK") || strings.TrimSpace(out) == "" {
			t.Fatalf("fish session exit %d:\n%s\n%s", code, out, errOut)
		}
		record(t, "shell fish session", "alias + completions")
	}
}

func TestShellUninstall(t *testing.T) {
	f := newShellFixture(t)
	f.writeRC(t, userRC)
	if _, errOut, code := f.run(t, nil, "shell", "install"); code != 0 {
		t.Fatalf("install exit %d, stderr: %s", code, errOut)
	}

	out, errOut, code := f.run(t, nil, "shell", "uninstall")
	if code != 0 {
		t.Fatalf("uninstall exit %d, stderr: %s", code, errOut)
	}
	if !strings.Contains(out, "Removed") {
		t.Fatalf("uninstall stdout = %q", out)
	}
	got := f.readRC(t)
	saveArtifact(t, "shell-bashrc-uninstalled", ".sh", got)
	if strings.Contains(got, shellStart) || strings.Contains(got, "alias lb") {
		t.Fatalf("uninstall left integration behind:\n%s", got)
	}
	if !strings.Contains(got, "export EDITOR=vim") {
		t.Fatalf("uninstall removed user configuration:\n%s", got)
	}

	// Uninstalling again is a no-op, not an error.
	out, _, code = f.run(t, nil, "shell", "uninstall")
	if code != 0 || !strings.Contains(out, "No shell integration") {
		t.Fatalf("second uninstall exit %d stdout %q", code, out)
	}
	record(t, "shell uninstall", "idempotent")
}

func TestShellDetection(t *testing.T) {
	f := newShellFixture(t)

	// SHELL drives the default target file for each supported shell.
	cases := []struct {
		shell string
		path  string
		want  string
	}{
		{"zsh", filepath.Join(f.home, ".zshrc"), "lingbox completion zsh"},
		{"fish", filepath.Join(f.home, ".config", "fish", "config.fish"), "lingbox completion fish"},
	}
	for _, tc := range cases {
		if _, errOut, code := f.run(t, map[string]string{"SHELL": "/usr/bin/" + tc.shell}, "shell", "install"); code != 0 {
			t.Fatalf("%s install exit %d, stderr: %s", tc.shell, code, errOut)
		}
		data, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatalf("%s rc not created at %s: %v", tc.shell, tc.path, err)
		}
		if !strings.Contains(string(data), tc.want) {
			t.Fatalf("%s rc missing %q:\n%s", tc.shell, tc.want, data)
		}
		saveArtifact(t, "shell-rc-"+tc.shell, ".txt", string(data))
	}

	// ZDOTDIR wins over HOME for zsh.
	zdot := filepath.Join(f.home, "zdotdir")
	if _, errOut, code := f.run(t, map[string]string{"SHELL": "/usr/bin/zsh", "ZDOTDIR": zdot}, "shell", "install"); code != 0 {
		t.Fatalf("zsh ZDOTDIR install exit %d, stderr: %s", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(zdot, ".zshrc")); err != nil {
		t.Fatalf("ZDOTDIR rc missing: %v", err)
	}

	// --rc takes precedence and expands a leading ~.
	if _, errOut, code := f.run(t, nil, "shell", "install", "--shell", "bash", "--rc", "~/custom.rc"); code != 0 {
		t.Fatalf("--rc install exit %d, stderr: %s", code, errOut)
	}
	if data, err := os.ReadFile(filepath.Join(f.home, "custom.rc")); err != nil || !strings.Contains(string(data), "alias lb='lingbox'") {
		t.Fatalf("--rc target not written: %v\n%s", err, data)
	}

	// An unrecognised shell fails loudly and writes nothing.
	out, errOut, code := f.run(t, map[string]string{"SHELL": "/bin/tcsh"}, "shell", "install")
	if code == 0 {
		t.Fatalf("tcsh install unexpectedly succeeded:\n%s", out)
	}
	if !strings.Contains(errOut, "unsupported shell") {
		t.Fatalf("tcsh error = %q", errOut)
	}
	if _, err := os.Stat(filepath.Join(f.home, ".tcshrc")); err == nil {
		t.Fatalf("unsupported shell created a file")
	}
	record(t, "shell detection", "zsh/fish/ZDOTDIR/--rc/unsupported")
}

func TestShellPreservesSymlinkedRC(t *testing.T) {
	f := newShellFixture(t)
	dotfiles := filepath.Join(f.home, "dotfiles")
	if err := os.MkdirAll(dotfiles, 0o755); err != nil {
		t.Fatalf("mkdir dotfiles: %v", err)
	}
	target := filepath.Join(dotfiles, "bashrc")
	if err := os.WriteFile(target, []byte(userRC), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, f.rc); err != nil {
		t.Fatalf("symlink rc: %v", err)
	}

	if _, errOut, code := f.run(t, nil, "shell", "install"); code != 0 {
		t.Fatalf("install exit %d, stderr: %s", code, errOut)
	}
	fi, err := os.Lstat(f.rc)
	if err != nil {
		t.Fatalf("lstat rc: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("rc is no longer a symlink (mode=%v)", fi.Mode())
	}
	data, err := os.ReadFile(target)
	if err != nil || !strings.Contains(string(data), "alias lb='lingbox'") {
		t.Fatalf("symlink target not updated: %v\n%s", err, data)
	}
	saveArtifact(t, "shell-symlink-target", ".sh", string(data))
	record(t, "shell symlink", "writes through, link preserved")
}
