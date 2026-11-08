// Package e2e exercises the built lingbox binary end to end.
//
// The launcher contract is the observable interface between ling-box and
// any launcher (Alfred, uTools, Raycast, ...):
//
//   - lingbox tools --json        lists the supported tools
//   - <tool> ... --json           returns a result envelope
//   - lingbox launcher alfred     speaks Alfred's Script Filter JSON
//
// The tests build the real binary, drive it as a user would, and write the
// raw outputs to test/e2e/artifacts so a failure can be reproduced from the
// artifact alone.
package e2e

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

var (
	binPath      string
	repoRoot     string
	artifactsDir string
	report       strings.Builder
)

func TestMain(m *testing.M) {
	var err error
	repoRoot, err = filepath.Abs("../..")
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve repo root: %v\n", err)
		os.Exit(1)
	}

	tmp, err := os.MkdirTemp("", "lingbox-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "temp dir: %v\n", err)
		os.Exit(1)
	}

	binName := "lingbox"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath = filepath.Join(tmp, binName)

	build := exec.Command("go", "build", "-trimpath", "-o", binPath, ".")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build lingbox: %v\n%s\n", err, out)
		os.Exit(1)
	}

	artifactsDir = filepath.Join(repoRoot, "test", "e2e", "artifacts")
	if err := os.RemoveAll(artifactsDir); err != nil {
		fmt.Fprintf(os.Stderr, "clean artifacts: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create artifacts: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	report.WriteString("\nartifact directory: " + artifactsDir + "\n")
	if err := os.WriteFile(filepath.Join(artifactsDir, "report.txt"), []byte(report.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write report: %v\n", err)
	}
	os.RemoveAll(tmp)
	os.Exit(code)
}

func record(t *testing.T, name, detail string) {
	t.Helper()
	fmt.Fprintf(&report, "PASS %-32s %s\n", name, detail)
}

func saveArtifact(t *testing.T, name, ext, data string) string {
	t.Helper()
	p := filepath.Join(artifactsDir, name+ext)
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatalf("write artifact %s: %v", p, err)
	}
	return p
}

// runBin runs the built binary and returns stdout, stderr and the exit code.
func runBin(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "TERM=dumb", "LINGBOX_NO_GUI=1")
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("run %v: %v", args, err)
		}
	}
	return out.String(), errOut.String(), code
}

type envelope struct {
	Schema int             `json:"schema"`
	OK     bool            `json:"ok"`
	Tool   string          `json:"tool"`
	Text   string          `json:"text"`
	Copy   string          `json:"copy"`
	Error  string          `json:"error"`
	Extra  json.RawMessage `json:"extra"`
}

type alfredText struct {
	Copy      string `json:"copy"`
	Largetype string `json:"largetype"`
}

type alfredItem struct {
	Title        string     `json:"title"`
	Subtitle     string     `json:"subtitle"`
	Arg          string     `json:"arg"`
	Autocomplete string     `json:"autocomplete"`
	Valid        bool       `json:"valid"`
	Text         alfredText `json:"text"`
}

type alfredResponse struct {
	Items []alfredItem `json:"items"`
	Rerun float64      `json:"rerun"`
}

func parseEnvelope(t *testing.T, name, out string) envelope {
	t.Helper()
	saveArtifact(t, name, ".json", out)
	var e envelope
	if err := json.Unmarshal([]byte(out), &e); err != nil {
		t.Fatalf("%s: invalid JSON envelope: %v\n%s", name, err, out)
	}
	return e
}

func parseAlfred(t *testing.T, name, out string) alfredResponse {
	t.Helper()
	saveArtifact(t, name, ".json", out)
	var r alfredResponse
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("%s: invalid Alfred JSON: %v\n%s", name, err, out)
	}
	if len(r.Items) == 0 {
		t.Fatalf("%s: no items\n%s", name, out)
	}
	return r
}

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestToolsJSON(t *testing.T) {
	out, errOut, code := runBin(t, "", "tools", "--json")
	saveArtifact(t, "tools", ".json", out)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errOut)
	}
	if errOut != "" {
		t.Fatalf("unexpected stderr: %s", errOut)
	}

	var doc struct {
		Schema int `json:"schema"`
		Tools  []struct {
			ID            string   `json:"id"`
			Aliases       []string `json:"aliases"`
			Summary       string   `json:"summary"`
			Example       string   `json:"example"`
			Input         string   `json:"input"`
			Output        string   `json:"output"`
			RequiresInput bool     `json:"requires_input"`
			LongRunning   bool     `json:"long_running"`
		} `json:"tools"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if doc.Schema != 1 {
		t.Fatalf("schema = %d, want 1", doc.Schema)
	}

	byID := map[string]int{}
	for i, tool := range doc.Tools {
		byID[tool.ID] = i
	}
	for _, id := range []string{"uuid", "password", "url", "base64", "unicode", "ipcalc", "color", "plate", "date", "json", "ssl", "ipgeo"} {
		if _, ok := byID[id]; !ok {
			t.Errorf("tool %q missing from tools --json", id)
		}
	}
	for _, id := range []string{"ssl", "ipgeo"} {
		if !doc.Tools[byID[id]].LongRunning {
			t.Errorf("%s should be marked long_running", id)
		}
	}

	ip := doc.Tools[byID["ipcalc"]]
	if ip.Input != "fields" || !ip.RequiresInput {
		t.Errorf("ipcalc metadata = input %q requires_input %v", ip.Input, ip.RequiresInput)
	}
	if !contains(ip.Aliases, "ip") {
		t.Errorf("ipcalc aliases = %v, want ip", ip.Aliases)
	}
	if !contains(doc.Tools[byID["plate"]].Aliases, "车牌") {
		t.Errorf("plate aliases = %v, want 车牌", doc.Tools[byID["plate"]].Aliases)
	}
	if doc.Tools[byID["uuid"]].Input != "none" {
		t.Errorf("uuid input = %q, want none", doc.Tools[byID["uuid"]].Input)
	}
	if doc.Tools[byID["json"]].Input != "stdin" {
		t.Errorf("json input = %q, want stdin", doc.Tools[byID["json"]].Input)
	}

	plain, _, code := runBin(t, "", "tools")
	if code != 0 || !strings.Contains(plain, "ipcalc") {
		t.Errorf("tools (plain) = %q, exit %d", plain, code)
	}

	record(t, "tools --json", fmt.Sprintf("%d tools", len(doc.Tools)))
}

func TestResultEnvelopes(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCopy string
		contains []string
		check    func(t *testing.T, e envelope)
	}{
		{
			name: "uuid",
			args: []string{"uuid", "--json"},
			check: func(t *testing.T, e envelope) {
				if !uuidV4.MatchString(e.Copy) {
					t.Errorf("copy %q is not a v4 UUID", e.Copy)
				}
			},
		},
		{
			name: "password",
			args: []string{"password", "--json", "-l", "24"},
			check: func(t *testing.T, e envelope) {
				if len(e.Copy) != 24 {
					t.Errorf("password length = %d, want 24", len(e.Copy))
				}
				if strings.ContainsAny(e.Copy, "|![]$()`") {
					t.Errorf("password contains excluded characters: %q", e.Copy)
				}
			},
		},
		{
			name:     "url",
			args:     []string{"url", "-e", "--json", "--", "hello world"},
			wantCopy: "hello+world",
		},
		{
			name:     "base64",
			args:     []string{"base64", "-e", "--json", "--", "Hello World"},
			wantCopy: "SGVsbG8gV29ybGQ=",
		},
		{
			name:     "unicode",
			args:     []string{"unicode", "--json", "--", "你好"},
			wantCopy: `\u4F60\u597D`,
		},
		{
			name:     "ipcalc",
			args:     []string{"ipcalc", "--json", "10.0.0.0/8"},
			contains: []string{"10.0.0.0", "255.0.0.0"},
		},
		{
			name:     "color",
			args:     []string{"color", "--json", "--", "#ff0000"},
			wantCopy: "#FF0000",
			contains: []string{"rgb(255, 0, 0)"},
		},
		{
			name:     "plate",
			args:     []string{"plate", "--json", "--", "湘"},
			contains: []string{"湖南省"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, code := runBin(t, "", tt.args...)
			if code != 0 {
				t.Fatalf("exit %d, stderr: %s", code, errOut)
			}
			e := parseEnvelope(t, tt.name, out)
			if e.Schema != 1 || !e.OK {
				t.Fatalf("envelope = %+v", e)
			}
			if e.Tool != tt.name {
				t.Errorf("tool = %q, want %q", e.Tool, tt.name)
			}
			if e.Text == "" || e.Copy == "" {
				t.Fatalf("empty text/copy: %+v", e)
			}
			if tt.wantCopy != "" && e.Copy != tt.wantCopy {
				t.Errorf("copy = %q, want %q", e.Copy, tt.wantCopy)
			}
			for _, want := range tt.contains {
				if !strings.Contains(e.Text, want) {
					t.Errorf("text missing %q:\n%s", want, e.Text)
				}
			}
			if strings.ContainsAny(e.Text, "\x1b") || strings.ContainsAny(e.Copy, "\x1b") {
				t.Errorf("ANSI escapes leaked into machine output")
			}
			if tt.check != nil {
				tt.check(t, e)
			}
			record(t, tt.name+" --json", "ok")
		})
	}
}

func TestResultEnvelopeErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"color-invalid", []string{"color", "--json", "--", "not-a-color"}},
		{"plate-invalid", []string{"plate", "--json", "--", "不存在"}},
		{"url-no-mode", []string{"url", "--json", "--", "x"}},
		{"base64-decode", []string{"base64", "-d", "--json", "--", "!!!"}},
		{"unicode-decode", []string{"unicode", "-d", "--json", "--", `\uZZZZ`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, _ := runBin(t, "", tt.args...)
			e := parseEnvelope(t, tt.name, out)
			if e.OK || e.Error == "" {
				t.Fatalf("want error envelope, got %+v", e)
			}
			record(t, tt.name, e.Error)
		})
	}
}

func TestAlfredDiscoverability(t *testing.T) {
	out, errOut, code := runBin(t, "", "launcher", "alfred", "--query", "")
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errOut)
	}
	r := parseAlfred(t, "alfred-empty", out)
	if len(r.Items) < 12 {
		t.Fatalf("got %d items, want at least 12", len(r.Items))
	}
	ip := findItem(r.Items, func(i alfredItem) bool { return strings.HasPrefix(i.Title, "ipcalc") })
	if ip == nil {
		t.Fatalf("no ipcalc suggestion: %+v", r.Items)
	}
	if ip.Valid {
		t.Errorf("suggestion should not be valid: %+v", ip)
	}
	if ip.Autocomplete != "ipcalc " {
		t.Errorf("autocomplete = %q, want %q", ip.Autocomplete, "ipcalc ")
	}

	out, _, _ = runBin(t, "", "launcher", "alfred", "--query", "i")
	r = parseAlfred(t, "alfred-prefix-i", out)
	if findItem(r.Items, func(i alfredItem) bool { return strings.HasPrefix(i.Title, "ipcalc") }) == nil {
		t.Errorf("query i did not suggest ipcalc: %+v", r.Items)
	}

	out, _, _ = runBin(t, "", "launcher", "alfred", "--query", "zzz")
	r = parseAlfred(t, "alfred-unknown", out)
	if r.Items[0].Valid || !strings.Contains(strings.ToLower(r.Items[0].Title), "no matching") {
		t.Errorf("unknown query item = %+v", r.Items[0])
	}

	// A query that starts with a dash must not be parsed as a flag.
	out, _, code = runBin(t, "", "launcher", "alfred", "--query", "-abc")
	if code != 0 {
		t.Fatalf("leading-dash query exit %d", code)
	}
	r = parseAlfred(t, "alfred-leading-dash", out)
	if r.Items[0].Valid {
		t.Errorf("leading-dash query item = %+v", r.Items[0])
	}

	// Async tools still show an input hint instead of starting a job.
	out, _, _ = runBin(t, "", "launcher", "alfred", "--query", "geo")
	r = parseAlfred(t, "alfred-hint-geo", out)
	if r.Items[0].Valid || r.Items[0].Autocomplete != "geo " {
		t.Errorf("geo hint item = %+v", r.Items[0])
	}

	record(t, "alfred discoverability", "list, prefix, unknown, leading dash, async hint")
}

func TestAlfredRuns(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantArg  string
		contains []string
		check    func(t *testing.T, i alfredItem)
	}{
		{
			name:  "uuid",
			query: "uuid",
			check: func(t *testing.T, i alfredItem) {
				if !uuidV4.MatchString(i.Arg) {
					t.Errorf("arg %q is not a v4 UUID", i.Arg)
				}
			},
		},
		{
			name:  "password",
			query: "password",
			check: func(t *testing.T, i alfredItem) {
				if len(i.Arg) != 16 {
					t.Errorf("password length = %d, want 16", len(i.Arg))
				}
			},
		},
		{name: "url-spaces", query: "url hello world", wantArg: "hello+world"},
		{name: "base64-alias", query: "b64 Hello World", wantArg: "SGVsbG8gV29ybGQ="},
		{name: "unicode", query: "uni 你好", wantArg: `\u4F60\u597D`},
		{name: "url-dash", query: "url -a b", wantArg: "-a+b"},
		{name: "ip-cidr", query: "ip 10.0.0.0/8", contains: []string{"10.0.0.0", "255.0.0.0"}, check: func(t *testing.T, i alfredItem) {
			if !strings.Contains(i.Title, "Network:") {
				t.Errorf("title = %q, want the network line", i.Title)
			}
		}},
		{name: "ip-fields", query: "ip 10.0.0.1 255.255.0.0", contains: []string{"255.255.0.0"}},
		{name: "color", query: "color #00ff00", wantArg: "#00FF00"},
		{name: "plate", query: "plate 湘", contains: []string{"湖南省"}},
		{name: "plate-cn-alias", query: "车牌 湘", contains: []string{"湖南省"}},
		{name: "date", query: "date add 2026-01-01 10", contains: []string{"2026-01-11"}},
		{name: "date-diff", query: "date diff 2026-01-01 2026-07-26", contains: []string{"Days"}},
		{name: "json-stdin", query: `json {"a":1,"b":[2,3]}`, contains: []string{`"a": 1`, `"b": [`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, code := runBin(t, "", "launcher", "alfred", "--query", tt.query)
			if code != 0 {
				t.Fatalf("exit %d, stderr: %s", code, errOut)
			}
			r := parseAlfred(t, "alfred-run-"+tt.name, out)
			if len(r.Items) != 1 {
				t.Fatalf("got %d items, want 1: %+v", len(r.Items), r.Items)
			}
			item := r.Items[0]
			if !item.Valid {
				t.Fatalf("item is not valid: %+v", item)
			}
			if item.Text.Copy != item.Arg || item.Text.Largetype == "" {
				t.Errorf("text fields not wired: %+v", item)
			}
			if tt.wantArg != "" && item.Arg != tt.wantArg {
				t.Errorf("arg = %q, want %q", item.Arg, tt.wantArg)
			}
			for _, want := range tt.contains {
				if !strings.Contains(item.Arg, want) {
					t.Errorf("arg missing %q:\n%s", want, item.Arg)
				}
			}
			if tt.check != nil {
				tt.check(t, item)
			}
			record(t, "alfred "+tt.name, "ok")
		})
	}
}

func TestLauncherQuery(t *testing.T) {
	out, errOut, code := runBin(t, "", "launcher", "query", "--query", "uuid")
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errOut)
	}
	r := parseAlfred(t, "launcher-query-uuid", out)
	if !uuidV4.MatchString(r.Items[0].Arg) {
		t.Errorf("launcher query arg = %q", r.Items[0].Arg)
	}
	record(t, "launcher query", "uuid")
}

func cleanJobStore(t *testing.T) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(os.TempDir(), "lingbox-launcher")); err != nil {
		t.Fatalf("clean job store: %v", err)
	}
}

func TestAlfredAsync(t *testing.T) {
	cleanJobStore(t)
	srv := httptest.NewUnstartedServer(http.NotFoundHandler())
	// The scanner probes old protocol versions on purpose; keep the test
	// output free of expected handshake errors.
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.StartTLS()
	defer srv.Close()

	target := strings.TrimPrefix(srv.URL, "https://")
	query := "ssl " + target

	out, errOut, code := runBin(t, "", "launcher", "alfred", "--query", query)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errOut)
	}
	r := parseAlfred(t, "alfred-async-first", out)
	if r.Items[0].Valid {
		t.Fatalf("first response should be a running item: %+v", r.Items[0])
	}
	if r.Rerun <= 0 {
		t.Errorf("first response rerun = %v, want > 0", r.Rerun)
	}

	deadline := time.Now().Add(30 * time.Second)
	for i := 1; time.Now().Before(deadline); i++ {
		time.Sleep(250 * time.Millisecond)
		out, errOut, code = runBin(t, "", "launcher", "alfred", "--query", query)
		if code != 0 {
			t.Fatalf("poll %d exit %d, stderr: %s", i, code, errOut)
		}
		r = parseAlfred(t, fmt.Sprintf("alfred-async-%02d", i), out)
		item := r.Items[0]
		if item.Valid {
			if !strings.Contains(item.Arg, target) {
				t.Errorf("result does not mention %s:\n%s", target, item.Arg)
			}
			if strings.Contains(item.Arg, "\x1b") {
				t.Errorf("ANSI escape leaked into the async result")
			}
			if r.Rerun != 0 {
				t.Errorf("rerun after completion = %v, want 0", r.Rerun)
			}
			record(t, "alfred async ssl", "completed")
			return
		}
		if strings.Contains(item.Title, "Error") {
			t.Fatalf("ssl scan failed: %+v", item)
		}
	}
	t.Fatal("timed out waiting for the ssl result")
}

func TestAlfredAsyncFailure(t *testing.T) {
	cleanJobStore(t)
	query := "ssl 127.0.0.1:1"

	out, errOut, code := runBin(t, "", "launcher", "alfred", "--query", query)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errOut)
	}
	r := parseAlfred(t, "alfred-async-fail-first", out)
	if r.Items[0].Valid {
		t.Fatalf("first response should not be a result: %+v", r.Items[0])
	}

	deadline := time.Now().Add(20 * time.Second)
	for i := 1; time.Now().Before(deadline); i++ {
		time.Sleep(250 * time.Millisecond)
		out, errOut, code = runBin(t, "", "launcher", "alfred", "--query", query)
		if code != 0 {
			t.Fatalf("poll %d exit %d, stderr: %s", i, code, errOut)
		}
		r = parseAlfred(t, fmt.Sprintf("alfred-async-fail-%02d", i), out)
		item := r.Items[0]
		if strings.Contains(item.Title, "Error") {
			if item.Valid {
				t.Errorf("error item should not be valid")
			}
			record(t, "alfred async ssl failure", item.Subtitle)
			return
		}
		if item.Valid {
			t.Fatalf("expected an error, got a result: %+v", item)
		}
	}
	t.Fatal("timed out waiting for the ssl failure")
}

func TestAlfredHintsAndErrors(t *testing.T) {
	t.Run("hint", func(t *testing.T) {
		out, _, code := runBin(t, "", "launcher", "alfred", "--query", "ip")
		if code != 0 {
			t.Fatalf("exit %d", code)
		}
		r := parseAlfred(t, "alfred-hint-ip", out)
		if len(r.Items) != 1 {
			t.Fatalf("got %d items", len(r.Items))
		}
		item := r.Items[0]
		if item.Valid {
			t.Errorf("hint should not be valid: %+v", item)
		}
		if item.Autocomplete != "ip " {
			t.Errorf("autocomplete = %q, want %q", item.Autocomplete, "ip ")
		}
		record(t, "alfred hint", "ip")
	})

	tests := []struct {
		name  string
		query string
	}{
		{"bad-ip", "ip nope"},
		{"bad-color", "color not-a-color"},
		{"bad-date", "date add 2026-01-01"},
		{"bad-json", "json not-json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, code := runBin(t, "", "launcher", "alfred", "--query", tt.query)
			if code != 0 {
				t.Fatalf("launcher must exit 0 so Alfred can render the error: exit %d, stderr %s", code, errOut)
			}
			r := parseAlfred(t, "alfred-error-"+tt.name, out)
			item := r.Items[0]
			if item.Valid {
				t.Fatalf("error item should not be valid: %+v", item)
			}
			if !strings.Contains(strings.ToLower(item.Title+item.Subtitle), "error") {
				t.Errorf("error not explained: %+v", item)
			}
			if strings.Contains(item.Subtitle, "Usage:") || strings.Contains(item.Subtitle, "\n") {
				t.Errorf("error subtitle should be one concise line: %q", item.Subtitle)
			}
			record(t, "alfred error "+tt.name, item.Subtitle)
		})
	}
}

func TestAlfredWorkflowPackage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("build.sh needs a POSIX shell")
	}
	if _, err := exec.LookPath("zip"); err != nil {
		t.Skip("zip not installed")
	}
	build := exec.Command("/bin/sh", "contrib/alfred/build.sh")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build workflow: %v\n%s", err, out)
	}
	zipPath := filepath.Join(repoRoot, "contrib", "alfred", "dist", "lingbox.alfredworkflow")
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("workflow not built: %v", err)
	}

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open workflow: %v", err)
	}
	defer zr.Close()

	names := map[string]bool{}
	var plist string
	for _, f := range zr.File {
		names[f.Name] = true
		if f.Name == "info.plist" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open info.plist: %v", err)
			}
			var buf bytes.Buffer
			buf.ReadFrom(rc)
			rc.Close()
			plist = buf.String()
		}
	}
	for _, want := range []string{"info.plist", "lib.sh", "script-filter.sh", "file-filter.sh", "run-action.sh"} {
		if !names[want] {
			t.Errorf("workflow missing %s (has %v)", want, names)
		}
	}
	for _, want := range []string{
		"alfred.workflow.input.scriptfilter",
		"alfred.workflow.output.clipboard",
		"alfred.workflow.trigger.action",
		"alfred.workflow.action.script",
		"alfred.workflow.output.notification",
		"script-filter.sh",
		"file-filter.sh",
		"run-action.sh",
		`"$@"`,
		"<key>keyword</key>",
		"<string>lb</string>",
	} {
		if !strings.Contains(plist, want) {
			t.Errorf("info.plist missing %q", want)
		}
	}

	// Keep a copy of the built workflow next to the test outputs.
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactsDir, "lingbox.alfredworkflow"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	record(t, "alfred workflow package", fmt.Sprintf("%d files", len(zr.File)))
}

func TestScriptFilter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("script-filter.sh needs a POSIX shell")
	}
	script := filepath.Join(repoRoot, "contrib", "alfred", "script-filter.sh")
	isolated := func() map[string]string {
		return map[string]string{
			"LINGBOX_BIN": "",
			"HOME":        t.TempDir(),
			"PATH":        "/usr/bin:/bin",
		}
	}
	run := func(scriptPath string, env map[string]string, args ...string) (string, string, int) {
		cmd := exec.Command("/bin/sh", append([]string{scriptPath}, args...)...)
		cmd.Dir = repoRoot
		cmd.Env = envWith(env)
		var out, errOut bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errOut
		err := cmd.Run()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		return out.String(), errOut.String(), code
	}

	// 1. Explicit override.
	out, errOut, code := run(script, map[string]string{"LINGBOX_BIN": binPath}, "uuid")
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errOut)
	}
	r := parseAlfred(t, "script-filter-uuid", out)
	if !uuidV4.MatchString(r.Items[0].Arg) {
		t.Errorf("script filter arg = %q", r.Items[0].Arg)
	}

	// 2. Alfred may pass the query split into words; the script must rejoin it.
	out, errOut, code = run(script, map[string]string{"LINGBOX_BIN": binPath}, "url", "hello", "world")
	if code != 0 {
		t.Fatalf("multi-arg exit %d, stderr: %s", code, errOut)
	}
	r = parseAlfred(t, "script-filter-multi-arg", out)
	if r.Items[0].Arg != "hello+world" {
		t.Errorf("multi-arg arg = %q, want hello+world", r.Items[0].Arg)
	}

	// 3. A bad explicit override must say so, not pretend ling-box is missing.
	out, _, code = run(script, map[string]string{"LINGBOX_BIN": "/nonexistent/lingbox"}, "uuid")
	if code != 0 {
		t.Fatalf("bad override should still emit JSON: exit %d", code)
	}
	r = parseAlfred(t, "script-filter-bad-override", out)
	if !strings.Contains(r.Items[0].Title, "LINGBOX_BIN") {
		t.Errorf("bad override item = %+v", r.Items[0])
	}

	// 4. A bundled bin/lingbox wins, without relying on PATH or the login
	// shell. This is the branch that fixes Alfred's limited PATH.
	bundleDir := t.TempDir()
	bundledScript := filepath.Join(bundleDir, "script-filter.sh")
	copyFile(t, script, bundledScript, 0o755)
	copyFile(t, filepath.Join(repoRoot, "contrib", "alfred", "lib.sh"), filepath.Join(bundleDir, "lib.sh"), 0o644)
	if err := os.MkdirAll(filepath.Join(bundleDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	copyFile(t, binPath, filepath.Join(bundleDir, "bin", "lingbox"), 0o755)

	out, errOut, code = run(bundledScript, isolated(), "uuid")
	if code != 0 {
		t.Fatalf("bundled exit %d, stderr: %s", code, errOut)
	}
	r = parseAlfred(t, "script-filter-bundled", out)
	if !uuidV4.MatchString(r.Items[0].Arg) {
		t.Errorf("bundled arg = %q", r.Items[0].Arg)
	}

	// 5. An old binary that does not know the launcher command must produce
	// a readable item instead of empty output.
	oldDir := t.TempDir()
	oldScript := filepath.Join(oldDir, "script-filter.sh")
	copyFile(t, script, oldScript, 0o755)
	copyFile(t, filepath.Join(repoRoot, "contrib", "alfred", "lib.sh"), filepath.Join(oldDir, "lib.sh"), 0o644)
	if err := os.MkdirAll(filepath.Join(oldDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldBin := filepath.Join(oldDir, "bin", "lingbox")
	if err := os.WriteFile(oldBin, []byte("#!/bin/sh\necho 'unknown command' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, _, code = run(oldScript, isolated(), "uuid")
	if code != 0 {
		t.Fatalf("old binary should still emit JSON: exit %d", code)
	}
	r = parseAlfred(t, "script-filter-old-binary", out)
	if !strings.Contains(strings.ToLower(r.Items[0].Title), "could not run") {
		t.Errorf("old binary item = %+v", r.Items[0])
	}

	record(t, "script-filter.sh", "override, multi-arg, bundled, bad paths")
}

func envWith(overrides map[string]string) []string {
	env := make([]string, 0, len(os.Environ())+len(overrides))
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if _, ok := overrides[k]; !ok {
			env = append(env, kv)
		}
	}
	for k, v := range overrides {
		env = append(env, k+"="+v)
	}
	return env
}

func copyFile(t *testing.T, src, dst string, mode os.FileMode) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func findItem(items []alfredItem, pred func(alfredItem) bool) *alfredItem {
	for i := range items {
		if pred(items[i]) {
			return &items[i]
		}
	}
	return nil
}
