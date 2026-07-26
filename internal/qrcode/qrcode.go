package qrcode

import (
	"image/jpeg"
	"image/png"
	"os"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// Generate creates a QR code image file from the given text.
func Generate(text, outputPath string, size int, format string) error {
	qr, err := qrcode.New(text, qrcode.Highest)
	if err != nil {
		return err
	}

	img := qr.Image(size)

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	switch strings.ToUpper(format) {
	case "JPG", "JPEG":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	case "GIF":
		return png.Encode(f, img) // PNG fallback for GIF (no native GIF encoder needed in simple case)
	default:
		return png.Encode(f, img)
	}
}

// GenerateToFile creates a QR code and writes it to a File object.
func GenerateToFile(text string, file *os.File, size int, format string) error {
	qr, err := qrcode.New(text, qrcode.Highest)
	if err != nil {
		return err
	}

	img := qr.Image(size)

	switch strings.ToUpper(format) {
	case "JPG", "JPEG":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	default:
		return png.Encode(file, img)
	}
}

