package qrcode

import (
	"image"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCreatesQRCodeFile(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "test-qr.png")

	err := Generate("https://example.com", output, 300, "PNG")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}

func TestGenerateCreatesQRCodeWithCorrectSize(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "test-qr-sized.png")
	expectedSize := 400

	err := Generate("https://example.com", output, expectedSize, "PNG")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	f, err := os.Open(output)
	if err != nil {
		t.Fatalf("cannot open output: %v", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("cannot decode image: %v", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width != expectedSize {
		t.Errorf("expected width %d, got %d", expectedSize, width)
	}
	if height != expectedSize {
		t.Errorf("expected height %d, got %d", expectedSize, height)
	}
}

func TestGenerateSupportsDifferentFormats(t *testing.T) {
	dir := t.TempDir()

	pngFile := filepath.Join(dir, "test-qr.png")
	jpgFile := filepath.Join(dir, "test-qr.jpg")

	err := Generate("test", pngFile, 300, "PNG")
	if err != nil {
		t.Fatalf("Generate PNG returned error: %v", err)
	}

	var pngExists bool
	if _, err := os.Stat(pngFile); err == nil {
		pngExists = true
	}
	if !pngExists {
		t.Error("PNG file not created")
	}

	err = Generate("test", jpgFile, 300, "JPG")
	if err != nil {
		t.Fatalf("Generate JPG returned error: %v", err)
	}

	var jpgExists bool
	if _, err := os.Stat(jpgFile); err == nil {
		jpgExists = true
	}
	if !jpgExists {
		t.Error("JPG file not created")
	}
}

func TestGeneratePNG(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "test.png")

	err := Generate("hello", output, 100, "PNG")
	if err != nil {
		t.Fatalf("got error %v", err)
	}

	// Verify it's actually a PNG by decoding the header
	f, _ := os.Open(output)
	defer f.Close()
	_, format, err := image.Decode(f)
	if err != nil {
		t.Fatalf("cannot decode image: %v", err)
	}
	if !strings.EqualFold(format, "png") {
		t.Errorf("expected PNG format, got %s", format)
	}
}

func TestGenerateHandlesChineseCharacters(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "test-qr-chinese.png")

	err := Generate("你好世界", output, 300, "PNG")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}

func TestGenerateToFile(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "test-qr-file.png")

	f, err := os.Create(output)
	if err != nil {
		t.Fatalf("cannot create file: %v", err)
	}
	defer f.Close()

	err = GenerateToFile("https://example.com", f, 300, "PNG")
	if err != nil {
		t.Fatalf("GenerateToFile returned error: %v", err)
	}
}

func verifyPNGSize(t *testing.T, path string, expectedSize int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("cannot open file: %v", err)
	}
	defer f.Close()

	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatalf("cannot decode PNG: %v", err)
	}

	if cfg.Width != expectedSize {
		t.Errorf("expected width %d, got %d", expectedSize, cfg.Width)
	}
	if cfg.Height != expectedSize {
		t.Errorf("expected height %d, got %d", expectedSize, cfg.Height)
	}
}
