package webserver

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServeMessage(t *testing.T) {
	tests := []struct {
		bind string
		port int
		want string
	}{
		{"", 8000, "Serving HTTP on 0.0.0.0 port 8000 (http://0.0.0.0:8000/) ..."},
		{"127.0.0.1", 8778, "Serving HTTP on 127.0.0.1 port 8778 (http://127.0.0.1:8778/) ..."},
		{"::1", 8080, "Serving HTTP on ::1 port 8080 (http://[::1]:8080/) ..."},
	}
	for _, tt := range tests {
		if got := serveMessage(tt.bind, tt.port); got != tt.want {
			t.Errorf("serveMessage(%q, %d) = %q, want %q", tt.bind, tt.port, got, tt.want)
		}
	}
}

func TestLogLine(t *testing.T) {
	tm := time.Date(2026, 8, 25, 20, 59, 3, 0, time.UTC)
	got := logLine("127.0.0.1", "GET / HTTP/1.1", 200, 12, tm)
	want := `127.0.0.1 - - [25/Aug/2026 20:59:03] "GET / HTTP/1.1" 200 12`
	if got != want {
		t.Errorf("logLine = %q, want %q", got, want)
	}
}

func TestTranslatePath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		urlPath string
		want    string
	}{
		{"/", root},
		{"/a/b", filepath.Join(root, "a", "b")},
		{"/a//b", filepath.Join(root, "a", "b")},
		{"/./a/./b", filepath.Join(root, "a", "b")},
		{"/a/../b", filepath.Join(root, "b")}, // normalized like posixpath.normpath
		{"/../../etc/passwd", filepath.Join(root, "etc", "passwd")},
	}
	for _, tt := range tests {
		got, ok := translatePath(root, tt.urlPath)
		if !ok {
			t.Errorf("translatePath(%q) refused", tt.urlPath)
			continue
		}
		if got != tt.want {
			t.Errorf("translatePath(%q) = %q, want %q", tt.urlPath, got, tt.want)
		}
	}
}

// newServer starts a test server over dir.
func newServer(t *testing.T, dir string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(Handler(dir))
}

func get(t *testing.T, client *http.Client, url string) (int, http.Header, string) {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, string(body)
}

func TestHandler(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("hello.txt", "hello world")
	write("sub/index.html", "<h1>sub index</h1>")
	write("sub/plain.txt", "plain")
	write("space name.txt", "spaced")
	write("a<b>.txt", "escaped")
	write("a.txt", "lower")
	write("B.txt", "upper")

	ts := newServer(t, dir)
	client := ts.Client()

	// Directory listing.
	code, hdr, body := get(t, client, ts.URL+"/")
	if code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", code)
	}
	if ct := hdr.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("listing Content-Type = %q", ct)
	}
	for _, want := range []string{
		"<!DOCTYPE HTML>",
		"<title>Directory listing for /</title>",
		"<h1>Directory listing for /</h1>",
		`<li><a href="hello.txt">hello.txt</a></li>`,
		`<li><a href="sub/">sub/</a></li>`,
		`<li><a href="a.txt">a.txt</a></li>`,
		`<li><a href="space%20name.txt">space name.txt</a></li>`,
		`<li><a href="a%3Cb%3E.txt">a&lt;b&gt;.txt</a></li>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("listing missing %q", want)
		}
	}
	// Case-insensitive sort: a.txt before B.txt.
	if i, j := strings.Index(body, "a.txt"), strings.Index(body, "B.txt"); i < 0 || j < 0 || i > j {
		t.Errorf("listing not sorted case-insensitively: %q", body)
	}

	// Plain file with MIME type.
	code, hdr, body = get(t, client, ts.URL+"/hello.txt")
	if code != http.StatusOK || body != "hello world" {
		t.Errorf("GET /hello.txt = %d %q", code, body)
	}
	if ct := hdr.Get("Content-Type"); ct != "text/plain" {
		t.Errorf("hello.txt Content-Type = %q", ct)
	}

	// Percent-encoded path.
	if _, _, body := get(t, client, ts.URL+"/space%20name.txt"); body != "spaced" {
		t.Errorf("encoded path = %q", body)
	}

	// index.html is served for a directory.
	if _, _, body := get(t, client, ts.URL+"/sub/"); !strings.Contains(body, "<h1>sub index</h1>") {
		t.Errorf("index.html not served: %q", body)
	}

	// Directory without trailing slash redirects (301, no body). The
	// default client follows redirects, so use one that doesn't.
	noRedirect := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := noRedirect.Get(ts.URL + "/sub")
	if err != nil {
		t.Fatal(err)
	}
	redirectBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("GET /sub = %d, want 301", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); !strings.HasSuffix(loc, "/sub/") {
		t.Errorf("Location = %q", loc)
	}
	if len(redirectBody) != 0 {
		t.Errorf("301 response has a body: %q", redirectBody)
	}

	// 404 for missing files, with Python's error page.
	code, _, body = get(t, client, ts.URL+"/nope")
	if code != http.StatusNotFound {
		t.Errorf("GET /nope = %d, want 404", code)
	}
	for _, want := range []string{
		"<p>Error code: 404</p>",
		"<p>Message: File not found.</p>",
		"<p>Error code explanation: 404 - Nothing matches the given URI.</p>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("404 page missing %q", want)
		}
	}

	// Path traversal is blocked.
	code, _, _ = get(t, client, ts.URL+"/../../etc/passwd")
	if code != http.StatusNotFound {
		t.Errorf("traversal = %d, want 404", code)
	}

	// Unsupported methods get Python's 501.
	resp2, err := client.Post(ts.URL+"/", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotImplemented || !strings.Contains(string(body2), "Unsupported method ('POST')") {
		t.Errorf("POST = %d %q", resp2.StatusCode, body2)
	}

	// HEAD returns headers without a body.
	resp3, err := client.Head(ts.URL + "/hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK || resp3.Header.Get("Content-Type") != "text/plain" {
		t.Errorf("HEAD = %d %q", resp3.StatusCode, resp3.Header.Get("Content-Type"))
	}
}

func TestLogging(t *testing.T) {
	var buf bytes.Buffer
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(logging(Handler(dir), &buf))
	defer ts.Close()

	if _, _, body := get(t, ts.Client(), ts.URL+"/nope"); !strings.Contains(body, "File not found.") {
		t.Fatal("unexpected 404 body")
	}
	if _, _, body := get(t, ts.Client(), ts.URL+"/f.txt"); body != "x" {
		t.Fatal("unexpected file body")
	}

	got := buf.String()
	for _, want := range []string{
		`code 404, message File not found`,
		`"GET /nope HTTP/1.1" 404`,
		`"GET /f.txt HTTP/1.1" 200`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("log missing %q:\n%s", want, got)
		}
	}
}

func TestServeFailsOnMissingDirectory(t *testing.T) {
	err := Serve(Options{
		Dir:  filepath.Join(t.TempDir(), "nope"),
		Port: 0,
		Out:  io.Discard,
		Err:  io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "directory") {
		t.Errorf("Serve = %v, want directory error", err)
	}
}
