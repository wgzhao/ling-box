package imgcat

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"png", "photo.png", true},
		{"PNG uppercase", "photo.PNG", true},
		{"jpeg", "photo.jpeg", true},
		{"jpg", "photo.jpg", true},
		{"gif", "photo.gif", true},
		{"bmp", "photo.bmp", true},
		{"webp", "photo.webp", true},
		{"tiff", "photo.tiff", true},
		{"txt", "notes.txt", false},
		{"no extension", "photo", false},
		{"double extension", "photo.png.bak", false},
		{"markdown", "README.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsImageFile(tt.filename); got != tt.want {
				t.Errorf("IsImageFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestListImages(t *testing.T) {
	dir := t.TempDir()

	// Create image files with different extensions.
	createFile(t, filepath.Join(dir, "b.png"))
	createFile(t, filepath.Join(dir, "a.jpg"))
	createFile(t, filepath.Join(dir, "photo.JPEG"))
	// Create a non-image file.
	createFile(t, filepath.Join(dir, "readme.txt"))
	// Create a subdirectory (should not be listed).
	subDir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	createFile(t, filepath.Join(subDir, "nested.png"))

	paths, err := ListImages(dir)
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}

	// Should be sorted by filename: a.jpg, b.png, photo.JPEG.
	if len(paths) != 3 {
		t.Fatalf("expected 3 images, got %d: %v", len(paths), paths)
	}
	want := []string{"a.jpg", "b.png", "photo.JPEG"}
	for i, w := range want {
		got := filepath.Base(paths[i])
		if got != w {
			t.Errorf("index %d: got %s, want %s", i, got, w)
		}
	}
}

func TestListImagesEmpty(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "readme.txt"))

	_, err := ListImages(dir)
	if err == nil {
		t.Error("expected error for directory with no images")
	}
	if !strings.Contains(err.Error(), dir) {
		t.Errorf("error should mention directory, got: %v", err)
	}
}

func TestListImagesMissingDir(t *testing.T) {
	_, err := ListImages("/nonexistent/path/xyz")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

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

func TestBrowseLoopEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := Browse(&buf, nil, Options{})
	if err == nil {
		t.Error("expected error for empty paths")
	}
}

// Helper: create an empty file for ListImages tests.
func createFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
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
