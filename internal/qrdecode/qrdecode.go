// Package qrdecode extracts text from QR code images.
package qrdecode

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// Result holds the decoded payload.
type Result struct {
	Text   string `json:"text"`
	Format string `json:"format"`
}

// DecodeFile decodes the first QR code found in path.
func DecodeFile(path string) (Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return Result{}, fmt.Errorf("open image: %w", err)
	}
	defer f.Close()
	return Decode(f)
}

// DecodeBytes decodes a QR code from encoded image bytes.
func DecodeBytes(data []byte) (Result, error) {
	return Decode(bytes.NewReader(data))
}

// Decode reads an image from r and decodes its QR code.
func Decode(r io.Reader) (Result, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return Result{}, fmt.Errorf("decode image: %w", err)
	}
	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return Result{}, fmt.Errorf("prepare image: %w", err)
	}
	decoded, err := qrcode.NewQRCodeReader().Decode(bitmap, nil)
	if err != nil {
		return Result{}, fmt.Errorf("no QR code found: %w", err)
	}
	return Result{Text: decoded.GetText(), Format: "QR_CODE"}, nil
}
