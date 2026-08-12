package imgcat

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  NavAction
	}{
		{"q quit", "q", NavQuit},
		{"Q quit", "Q", NavQuit},
		{"ctrl-c quit", "\x03", NavQuit},
		{"space next", " ", NavNext},
		{"enter next", "\r", NavNext},
		{"newline next", "\n", NavNext},
		{"right arrow next", "\x1b[C", NavNext},
		{"down arrow next", "\x1b[B", NavNext},
		{"up arrow prev", "\x1b[A", NavPrev},
		{"left arrow prev", "\x1b[D", NavPrev},
		{"bare esc ignored", "\x1b", NavNone},
		{"unknown escape ignored", "\x1b[Z", NavNone},
		{"letter x ignored", "x", NavNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader([]byte(tt.input))
			got, err := readKey(r)
			if err != nil {
				t.Fatalf("readKey: unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("readKey(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBrowseLoopQuitImmediately(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "a.png")
	writePNGFile(t, pngPath, 10, 6)

	paths := []string{pngPath}
	var buf bytes.Buffer

	err := browseLoop(&buf, bytes.NewReader([]byte("q")), paths, Options{
		Width:    10,
		Renderer: RendererHalfBlock,
	})
	if err != nil {
		t.Fatalf("browseLoop: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Image 1/1") {
		t.Error("output should contain status line with Image 1/1")
	}
	if !strings.Contains(out, "a.png") {
		t.Error("output should contain the filename")
	}
	if strings.Count(out, "Image 1/1") != 1 {
		t.Error("should only render once before quitting")
	}
}

func TestBrowseLoopNavigation(t *testing.T) {
	dir := t.TempDir()
	pngPath1 := filepath.Join(dir, "first.png")
	pngPath2 := filepath.Join(dir, "second.png")
	writePNGFile(t, pngPath1, 10, 6)
	writePNGFile(t, pngPath2, 10, 6)

	paths := []string{pngPath1, pngPath2}
	var buf bytes.Buffer

	// right arrow (next) -> second image, left arrow (prev) -> first image, q -> quit
	keys := "\x1b[C\x1b[Dq"
	err := browseLoop(&buf, bytes.NewReader([]byte(keys)), paths, Options{
		Width:    10,
		Renderer: RendererASCII,
	})
	if err != nil {
		t.Fatalf("browseLoop: %v", err)
	}

	out := buf.String()

	// Should have cleared screen at least twice (for 3 renders: first, second, first).
	if !strings.Contains(out, "\x1b[2J") {
		t.Error("output should contain clear-screen sequences")
	}

	// Should show both status lines.
	if !strings.Contains(out, "Image 1/2") {
		t.Error("output should contain Image 1/2")
	}
	if !strings.Contains(out, "Image 2/2") {
		t.Error("output should contain Image 2/2")
	}

	// First image should appear twice (initial + after prev from second).
	count := strings.Count(out, "first.png")
	if count < 2 {
		t.Errorf("first.png should appear at least twice (initial + after prev), got %d", count)
	}
}

func TestBrowseLoopSkipsInvalid(t *testing.T) {
	dir := t.TempDir()

	// An invalid file (text with .png extension — decodes but won't fail at ReadFile).
	invalidPath := filepath.Join(dir, "bad.png")
	if err := os.WriteFile(invalidPath, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A valid PNG.
	validPath := filepath.Join(dir, "good.png")
	writePNGFile(t, validPath, 10, 6)

	// Invalid file first, then valid — browseLoop should skip the invalid one.
	paths := []string{invalidPath, validPath}
	var buf bytes.Buffer

	err := browseLoop(&buf, bytes.NewReader([]byte("q")), paths, Options{
		Width:    10,
		Renderer: RendererASCII,
	})
	if err != nil {
		t.Fatalf("browseLoop: %v", err)
	}

	out := buf.String()

	// Should NOT show Image 1/2 for the invalid file — it's skipped.
	// Should show Image 2/2 (the valid one becomes the current, but total stays 2).
	if !strings.Contains(out, "good.png") {
		t.Error("output should contain good.png (valid file rendered)")
	}
	if strings.Contains(out, "bad.png") {
		t.Error("output should NOT contain bad.png (invalid file skipped)")
	}
}

func TestBrowseLoopAllInvalid(t *testing.T) {
	dir := t.TempDir()

	invalidPath := filepath.Join(dir, "bad.png")
	if err := os.WriteFile(invalidPath, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}

	paths := []string{invalidPath}
	var buf bytes.Buffer

	err := browseLoop(&buf, bytes.NewReader([]byte("q")), paths, Options{
		Width:    10,
		Renderer: RendererASCII,
	})
	if err == nil {
		t.Error("expected error when all files are invalid")
	}
	if !strings.Contains(err.Error(), "no displayable images") {
		t.Errorf("error should mention no displayable images, got: %v", err)
	}
}

func TestBrowseLoopEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := Browse(&buf, nil, Options{})
	if err == nil {
		t.Error("expected error for empty paths")
	}
}

// Helper: write a PNG file to disk using makeTestPNG.
func writePNGFile(t *testing.T, path string, w, h int) {
	t.Helper()
	data := makeTestPNG(w, h)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
