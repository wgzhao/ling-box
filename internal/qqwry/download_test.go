package qqwry

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testServer serves the dat fixture on /good and 404s on /bad, counting
// requests.
func testServer(t *testing.T, fixture []byte) (*httptest.Server, *int) {
	t.Helper()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path == "/bad" {
			http.NotFound(w, r)
			return
		}
		w.Write(fixture)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// setSources replaces Sources for the duration of the test.
func setSources(t *testing.T, urls []string) {
	t.Helper()
	orig := Sources
	Sources = urls
	t.Cleanup(func() { Sources = orig })
}

func TestEnsureDownloads(t *testing.T) {
	fixture := buildDat(t, [][]byte{
		directRecord(t, ipv4("255.255.255.255"), "IANA", ""),
	}, []testEntry{{start: ipv4("0.0.0.1"), rec: 0}})
	srv, hits := testServer(t, fixture)
	setSources(t, []string{srv.URL + "/bad", srv.URL + "/good"})

	path := filepath.Join(t.TempDir(), "qqwry.dat")
	var progress bytes.Buffer
	downloaded, err := Ensure(path, false, nil, &progress, true)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if !downloaded {
		t.Error("Ensure: expected a download")
	}
	if *hits != 2 { // first source failed, second succeeded
		t.Errorf("server hits = %d, want 2", *hits)
	}
	for _, want := range []string{"downloading qqwry.dat", "downloaded qqwry.dat", path} {
		if !strings.Contains(progress.String(), want) {
			t.Errorf("progress output %q missing %q", progress.String(), want)
		}
	}
	// interactive mode reports percentage steps
	if !strings.Contains(progress.String(), "%") {
		t.Errorf("interactive progress output %q missing percentage steps", progress.String())
	}
	// the downloaded file parses as a database
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open(downloaded): %v", err)
	}
	if rec, err := db.QueryStr("1.2.3.4"); err != nil || rec.Location() != "IANA" {
		t.Errorf("QueryStr on downloaded db = %q, %v", rec.Location(), err)
	}
	// a second call finds the file and skips the download
	downloaded, err = Ensure(path, false, nil, nil, false)
	if err != nil || downloaded {
		t.Errorf("Ensure(existing) = %v, %v; want false, nil", downloaded, err)
	}
	if *hits != 2 {
		t.Errorf("server hits after Ensure(existing) = %d, want 2", *hits)
	}
	// force re-downloads
	downloaded, err = Ensure(path, true, nil, nil, false)
	if err != nil || !downloaded {
		t.Errorf("Ensure(force) = %v, %v; want true, nil", downloaded, err)
	}
	if *hits != 4 {
		t.Errorf("server hits after Ensure(force) = %d, want 4", *hits)
	}
}

func TestEnsureProgressNonInteractive(t *testing.T) {
	fixture := buildDat(t, [][]byte{
		directRecord(t, ipv4("255.255.255.255"), "IANA", ""),
	}, []testEntry{{start: ipv4("0.0.0.1"), rec: 0}})
	srv, _ := testServer(t, fixture)
	setSources(t, []string{srv.URL + "/good"})

	var progress bytes.Buffer
	if _, err := Ensure(filepath.Join(t.TempDir(), "qqwry.dat"), false, nil, &progress, false); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if !strings.Contains(progress.String(), "downloaded qqwry.dat") {
		t.Errorf("progress output %q missing completion notice", progress.String())
	}
	if strings.Contains(progress.String(), "%") {
		t.Errorf("non-interactive progress output %q contains percentage steps", progress.String())
	}
}

func TestEnsureAllSourcesFail(t *testing.T) {
	srv, _ := testServer(t, nil)
	setSources(t, []string{srv.URL + "/bad"})

	path := filepath.Join(t.TempDir(), "qqwry.dat")
	downloaded, err := Ensure(path, false, nil, nil, false)
	if err == nil {
		t.Fatal("Ensure: expected error, got nil")
	}
	if downloaded {
		t.Error("Ensure: expected no download")
	}
	if !strings.Contains(err.Error(), srv.URL) {
		t.Errorf("error %q does not mention the source URL", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Error("failed download left a file behind")
	}
}

func TestEnsureRejectsGarbage(t *testing.T) {
	fixture := buildDat(t, [][]byte{
		directRecord(t, ipv4("255.255.255.255"), "IANA", ""),
	}, []testEntry{{start: ipv4("0.0.0.1"), rec: 0}})
	srv, _ := testServer(t, []byte("not a qqwry.dat"))
	setSources(t, []string{srv.URL + "/good"})

	path := filepath.Join(t.TempDir(), "qqwry.dat")
	if err := os.WriteFile(path, fixture, 0o644); err != nil {
		t.Fatal(err)
	}
	// forcing an update downloads garbage; the existing file must survive
	if _, err := Ensure(path, true, nil, nil, false); err == nil {
		t.Fatal("Ensure: expected error for garbage download, got nil")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, fixture) {
		t.Error("existing database was clobbered by a failed update")
	}
}

func TestDefaultPath(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("ling-box", "qqwry.dat")) {
		t.Errorf("DefaultPath() = %q, want suffix %q", path, filepath.Join("ling-box", "qqwry.dat"))
	}
}

func TestDefaultDirFallback(t *testing.T) {
	fail := func() (string, error) { return "", errors.New("unavailable") }
	ok := func(d string) func() (string, error) {
		return func() (string, error) { return d, nil }
	}
	tests := []struct {
		name    string
		cache   func() (string, error)
		config  func() (string, error)
		home    func() (string, error)
		want    string
		wantErr bool
	}{
		{name: "cache dir first", cache: ok("/cache"), config: ok("/config"), home: ok("/home"), want: "/cache"},
		{name: "config dir fallback", cache: fail, config: ok("/config"), home: ok("/home"), want: "/config"},
		{name: "home fallback", cache: fail, config: fail, home: ok("/home"), want: "/home"},
		{name: "all unavailable", cache: fail, config: fail, home: fail, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := defaultDir(tc.cache, tc.config, tc.home)
			if tc.wantErr {
				if err == nil {
					t.Fatal("defaultDir: expected error, got nil")
				}
				return
			}
			if err != nil || got != tc.want {
				t.Errorf("defaultDir() = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}
