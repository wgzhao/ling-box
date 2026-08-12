// Package imgcat renders images in the terminal using ANSI true color,
// iTerm2 inline protocol, Kitty graphics protocol, or ASCII art.
package imgcat

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"   // register GIF decoder
	_ "image/jpeg"  // register JPEG decoder
	"image/png"
	"io"
	"math"
	"os"
	"strings"

	_ "golang.org/x/image/bmp"  // register BMP decoder
	_ "golang.org/x/image/tiff" // register TIFF decoder
	_ "golang.org/x/image/webp" // register WebP decoder

	"golang.org/x/image/draw"
	"golang.org/x/term"
)

// Renderer identifies the terminal image rendering method.
type Renderer string

const (
	// RendererAuto detects the best renderer for the current terminal.
	RendererAuto Renderer = "auto"
	// RendererHalfBlock uses ▀ characters with 24-bit ANSI background/foreground colors.
	// Two vertical pixels per character — best compatibility across modern terminals.
	RendererHalfBlock Renderer = "halfblock"
	// RendererITerm2 uses the OSC 1337 inline images protocol.
	// Despite the name, it works in iTerm2, WezTerm, Warp, kaku,
	// kitty (compat mode), VS Code terminal, and others.
	// Terminals that don't understand the sequence silently ignore it.
	RendererITerm2 Renderer = "iterm2"
	// RendererKitty uses the Kitty terminal graphics protocol.
	// Full-quality image, works in Kitty.
	RendererKitty Renderer = "kitty"
	// RendererASCII renders the image as grayscale ASCII art.
	// Maximum compatibility — works in any terminal.
	RendererASCII Renderer = "ascii"
)

// Options controls how an image is rendered.
type Options struct {
	// Width is the desired output width in character columns.
	// 0 (default) means auto-detect from terminal width.
	Width int
	// Renderer selects the rendering method.
	// "" or "auto" means auto-detect from terminal environment.
	Renderer Renderer
}

// Display renders raw image bytes to w. data should be the raw bytes of a
// PNG, JPEG, or GIF file.
func Display(w io.Writer, data []byte, opts Options) error {
	if opts.Renderer == "" || opts.Renderer == RendererAuto {
		opts.Renderer = detectRenderer()
	}

	// iTerm2 and Kitty work with raw image bytes — pass them directly.
	if opts.Renderer == RendererITerm2 {
		return renderITerm2(w, data)
	}
	if opts.Renderer == RendererKitty {
		return renderKitty(w, data)
	}

	// Half-block and ASCII need to decode the image first.
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	if opts.Width <= 0 {
		opts.Width = terminalWidth()
	}

	img = resize(img, opts.Width, opts.Renderer)

	switch opts.Renderer {
	case RendererASCII:
		return renderASCII(w, img)
	default:
		return renderHalfBlock(w, img)
	}
}

// detectRenderer picks the best renderer for the current terminal.
//
// Defaults to iTerm2 (OSC 1337) because it is widely supported —
// iTerm2, WezTerm, Warp, kaku, kitty (compat mode), VS Code terminal,
// and many others — and produces lossless output. Terminals that don't
// understand OSC 1337 silently ignore it, so if you get no output, use
// -r halfblock to fall back to ANSI true-color.
func detectRenderer() Renderer {
	// Kitty has its own protocol; prefer it on native Kitty.
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return RendererKitty
	}

	// OSC 1337 is the default — safe to emit everywhere.
	return RendererITerm2
}

// terminalWidth returns the terminal width in columns, or a safe default.
func terminalWidth() int {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return 80
	}
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		return 80
	}
	// Use 80% of terminal width for a comfortable margin.
	w := width * 80 / 100
	if w < 20 {
		w = 20
	}
	return w
}

// resize scales the image to the target width, preserving aspect ratio.
// For ASCII renderer, the height is halved to compensate for the ~2:1
// character-cell aspect ratio of most terminals.
func resize(img image.Image, targetWidth int, r Renderer) image.Image {
	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	targetHeight := int(math.Round(float64(targetWidth) * float64(origH) / float64(origW)))

	// Terminal character cells are roughly twice as tall as they are wide.
	// Half-block renders 2 source pixels per character row, naturally
	// compensating for this. ASCII renders 1 pixel per row, so we halve
	// the height to keep the correct aspect ratio.
	if r == RendererASCII {
		targetHeight /= 2
		if targetHeight < 1 {
			targetHeight = 1
		}
	}

	// Return early only if both dimensions are already at or below target.
	if origW <= targetWidth && origH <= targetHeight {
		return img
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

// renderHalfBlock renders the image using ▀ (U+2580, upper half block).
// Each character cell carries two vertical pixels: the top half uses the
// background color, the bottom half uses the foreground color.
func renderHalfBlock(w io.Writer, img image.Image) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var sb strings.Builder
	// Pre-allocate: each pixel pair ≈ 40 bytes of escape sequences.
	sb.Grow(width * ((height + 1) / 2) * 42)

	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x++ {
			top := img.At(x, y)
			bot := top
			if y+1 < height {
				bot = img.At(x, y+1)
			}

			tr, tg, tb := unpackRGB(top)
			br, bg, bb := unpackRGB(bot)

			// Background = top pixel, foreground = bottom pixel.
			// The ▀ glyph fills the upper half of the cell with the background
			// color and the lower half with the foreground color.
			fmt.Fprintf(&sb, "\x1b[48;2;%d;%d;%dm\x1b[38;2;%d;%d;%dm▀", tr, tg, tb, br, bg, bb)
		}
		sb.WriteString("\x1b[0m\n")
	}

	// Final reset.
	_, err := fmt.Fprint(w, sb.String())
	return err
}

// unpackRGB extracts 8-bit R, G, B values from a color.Color.
func unpackRGB(c color.Color) (r, g, b int) {
	cr, cg, cb, _ := c.RGBA()
	// RGBA() returns premultiplied 16-bit values (0–65535).
	// Shift right by 8 to get standard 0–255.
	return int(cr >> 8), int(cg >> 8), int(cb >> 8)
}

// nativeTermFormat reports whether data is in a format that terminal
// emulators commonly decode natively (PNG, JPEG, GIF), based on magic bytes.
func nativeTermFormat(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// PNG: \x89 P N G
	if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return true
	}
	// JPEG: \xff \xd8 \xff
	if data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return true
	}
	// GIF: G I F 8
	if data[0] == 'G' && data[1] == 'I' && data[2] == 'F' && data[3] == '8' {
		return true
	}
	return false
}

// renderITerm2 outputs the image using iTerm2's inline image protocol.
// See: https://iterm2.com/documentation-images.html
func renderITerm2(w io.Writer, data []byte) error {
	payload := data
	if !nativeTermFormat(data) {
		pngData, err := encodePNG(data)
		if err != nil {
			return fmt.Errorf("re-encode to PNG for iTerm2: %w", err)
		}
		payload = pngData
	}
	encoded := base64.StdEncoding.EncodeToString(payload)
	_, err := fmt.Fprintf(w, "\x1b]1337;File=inline=1:%s\x07\n", encoded)
	return err
}

// renderKitty outputs the image using the Kitty terminal graphics protocol.
// See: https://sw.kovidgoyal.net/kitty/graphics-protocol/
func renderKitty(w io.Writer, data []byte) error {
	// Re-encode to PNG in case the input is JPEG/GIF, since Kitty expects
	// a format it can decode natively.
	pngData, err := encodePNG(data)
	if err != nil {
		return fmt.Errorf("re-encode to PNG for Kitty: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(pngData)
	// APC G a=T (transmit), f=100 (PNG format); chunk size is omitted so
	// the whole image goes in one transmission.
	_, err = fmt.Fprintf(w, "\x1b_Ga=T,f=100;%s\x1b\\", encoded)
	return err
}

// encodePNG re-encodes an image (any format) as PNG bytes.
func encodePNG(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderASCII renders the image as grayscale ASCII art using a character ramp.
func renderASCII(w io.Writer, img image.Image) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	const ramp = " .:-=+*#%@"
	var sb strings.Builder
	sb.Grow(width*height + height) // 1 char per pixel + newlines

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gray := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			// Map 0–255 to an index in the ramp.
			idx := int(float64(gray.Y) / 255.0 * float64(len(ramp)-1))
			sb.WriteByte(ramp[idx])
		}
		sb.WriteByte('\n')
	}

	_, err := fmt.Fprint(w, sb.String())
	return err
}
