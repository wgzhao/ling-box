package qrdecode

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	qrcode "github.com/skip2/go-qrcode"
)

func TestDecodeGeneratedPNG(t *testing.T) {
	for _, want := range []string{"https://example.com/你好", "lingbox"} {
		data, err := qrcode.Encode(want, qrcode.Medium, 256)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeBytes(data)
		if err != nil {
			t.Fatalf("%q: %v", want, err)
		}
		if got.Text != want {
			t.Errorf("text = %q, want %q", got.Text, want)
		}
		if got.Format != "QR_CODE" {
			t.Errorf("format = %q", got.Format)
		}
	}
}

func TestDecodeFile(t *testing.T) {
	const want = "file-based QR"
	data, err := qrcode.Encode(want, qrcode.Low, 200)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "code.png")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := DecodeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != want {
		t.Errorf("text = %q, want %q", got.Text, want)
	}
}

func TestDecodeRejectsNonImage(t *testing.T) {
	if _, err := DecodeBytes([]byte("not an image")); err == nil {
		t.Error("expected an error for a non-image")
	}
}

func TestDecodeBlankImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 128, 128))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeBytes(buf.Bytes()); err == nil {
		t.Error("expected an error for an image without a QR code")
	}
}
