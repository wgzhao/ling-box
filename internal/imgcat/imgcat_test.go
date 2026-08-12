package imgcat

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"golang.org/x/image/bmp"
)

// makeTestPNG creates a small PNG image (10×6 pixels, solid red gradient-like)
// and returns the raw PNG bytes.
func makeTestPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := uint8(255 * x / w)
			g := uint8(255 * y / h)
			b := uint8(128)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestDisplayHalfBlock(t *testing.T) {
	data := makeTestPNG(10, 6)
	var buf bytes.Buffer

	err := Display(&buf, data, Options{
		Width:    10,
		Renderer: RendererHalfBlock,
	})
	if err != nil {
		t.Fatalf("Display halfblock: %v", err)
	}

	out := buf.String()

	// Should contain half-block characters (▀)
	if !strings.Contains(out, "▀") {
		t.Error("half-block output should contain ▀ characters")
	}

	// Should contain ANSI escape sequences for 24-bit color
	if !strings.Contains(out, "\x1b[48;2;") {
		t.Error("half-block output should contain background ANSI true color sequences")
	}
	if !strings.Contains(out, "\x1b[38;2;") {
		t.Error("half-block output should contain foreground ANSI true color sequences")
	}

	// Should reset at end of each line
	if !strings.Contains(out, "\x1b[0m\n") {
		t.Error("half-block output should reset colors at end of each line")
	}

	// Height should be ceil(6/2) = 3 rows of ▀ chars
	t.Logf("output:\n%s", out)
}

func TestDisplayASCII(t *testing.T) {
	data := makeTestPNG(10, 6)
	var buf bytes.Buffer

	err := Display(&buf, data, Options{
		Width:    10,
		Renderer: RendererASCII,
	})
	if err != nil {
		t.Fatalf("Display ascii: %v", err)
	}

	out := buf.String()

	// Should not contain ANSI escape sequences
	if strings.Contains(out, "\x1b[") {
		t.Error("ASCII output should not contain ANSI escape sequences")
	}

	// Strip trailing newline only — leading chars may be spaces from the ramp.
	trimmed := strings.TrimRight(out, "\n")
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 {
		t.Fatal("ASCII output should have at least one line")
	}
	// Each line should have exactly width characters (no ANSI escapes in ASCII mode).
	if len(lines[0]) != 10 {
		t.Errorf("ASCII output lines should be 10 chars wide, got %d (%q)", len(lines[0]), lines[0])
	}
}

func TestDisplayITerm2(t *testing.T) {
	data := makeTestPNG(10, 6)
	var buf bytes.Buffer

	err := Display(&buf, data, Options{
		Renderer: RendererITerm2,
	})
	if err != nil {
		t.Fatalf("Display iterm2: %v", err)
	}

	out := buf.String()

	// Should contain iTerm2 escape sequence
	if !strings.Contains(out, "\x1b]1337;File=inline=1:") {
		t.Error("iTerm2 output should contain OSC 1337 sequence")
	}
	if !strings.Contains(out, "\x07") {
		t.Error("iTerm2 output should end with BEL character")
	}
}

func TestDisplayKitty(t *testing.T) {
	data := makeTestPNG(10, 6)
	var buf bytes.Buffer

	err := Display(&buf, data, Options{
		Renderer: RendererKitty,
	})
	if err != nil {
		t.Fatalf("Display kitty: %v", err)
	}

	out := buf.String()

	// Should contain Kitty graphics protocol sequence
	if !strings.Contains(out, "\x1b_Ga=T,f=100;") {
		t.Error("Kitty output should contain APC G sequence")
	}
	if !strings.Contains(out, "\x1b\\") {
		t.Error("Kitty output should end with ST (string terminator)")
	}
}

func TestDisplayInvalidRenderer(t *testing.T) {
	data := makeTestPNG(10, 6)
	var buf bytes.Buffer

	// Unknown renderer should fall back to half-block
	err := Display(&buf, data, Options{
		Width:    10,
		Renderer: "unknown",
	})
	if err != nil {
		t.Fatalf("Display unknown renderer: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "▀") {
		t.Error("unknown renderer should fall back to half-block")
	}
}

func TestDisplaySmallImage(t *testing.T) {
	// Image smaller than target width — should not be resized.
	data := makeTestPNG(5, 5)
	var buf bytes.Buffer

	err := Display(&buf, data, Options{
		Width:    10,
		Renderer: RendererHalfBlock,
	})
	if err != nil {
		t.Fatalf("Display small image: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "▀") {
		t.Error("small image should still render half-block output")
	}
}

func TestResize(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	// Fill with something to avoid all-zero
	for x := 0; x < 200; x++ {
		img.Set(x, 50, color.RGBA{255, 0, 0, 255})
	}

	t.Run("downscale", func(t *testing.T) {
		resized := resize(img, 50, RendererHalfBlock)
		bounds := resized.Bounds()
		if bounds.Dx() != 50 {
			t.Errorf("expected width 50, got %d", bounds.Dx())
		}
		if bounds.Dy() != 25 {
			t.Errorf("expected height 25, got %d", bounds.Dy())
		}
	})

	t.Run("ascii-height-halved", func(t *testing.T) {
		resized := resize(img, 50, RendererASCII)
		bounds := resized.Bounds()
		if bounds.Dx() != 50 {
			t.Errorf("expected width 50, got %d", bounds.Dx())
		}
		// 50 * 100/200 = 25, then /2 for ASCII = 12
		if bounds.Dy() != 12 {
			t.Errorf("expected height 12 (halved for ASCII), got %d", bounds.Dy())
		}
	})

	t.Run("already-small", func(t *testing.T) {
		resized := resize(img, 300, RendererHalfBlock)
		bounds := resized.Bounds()
		if bounds.Dx() != 200 {
			t.Errorf("image smaller than target should not resize up, got width %d", bounds.Dx())
		}
	})
}

func TestUnpackRGB(t *testing.T) {
	c := color.RGBA{255, 128, 64, 255}
	r, g, b := unpackRGB(c)
	if r != 255 || g != 128 || b != 64 {
		t.Errorf("unpackRGB(%v) = (%d, %d, %d), want (255, 128, 64)", c, r, g, b)
	}

	// Black
	c2 := color.RGBA{0, 0, 0, 255}
	r, g, b = unpackRGB(c2)
	if r != 0 || g != 0 || b != 0 {
		t.Errorf("unpackRGB black = (%d, %d, %d), want (0, 0, 0)", r, g, b)
	}
}

func TestDetectRenderer(t *testing.T) {
	// detectRenderer returns a valid renderer even without special env vars.
	r := detectRenderer()
	switch r {
	case RendererHalfBlock, RendererITerm2, RendererKitty:
		// all valid
	default:
		t.Errorf("detectRenderer returned unknown renderer: %s", r)
	}
}

func TestRenderITerm2PassthroughPNG(t *testing.T) {
	data := makeTestPNG(10, 6)
	var buf bytes.Buffer

	err := renderITerm2(&buf, data)
	if err != nil {
		t.Fatalf("renderITerm2: %v", err)
	}

	payload := extractITerm2Payload(t, buf.String())
	if len(payload) < 4 {
		t.Fatal("payload too short")
	}

	// PNG magic bytes: \x89 P N G
	if payload[0] != 0x89 || payload[1] != 'P' || payload[2] != 'N' || payload[3] != 'G' {
		t.Error("PNG input should pass through unchanged (PNG magic bytes expected)")
	}
}

func TestRenderITerm2NormalizesBMP(t *testing.T) {
	// Create a BMP image — BMP is not natively decoded by terminal emulators,
	// so renderITerm2 should re-encode it to PNG.
	img := image.NewRGBA(image.Rect(0, 0, 10, 6))
	var bmpBuf bytes.Buffer
	if err := bmp.Encode(&bmpBuf, img); err != nil {
		t.Fatalf("bmp encode: %v", err)
	}

	var out bytes.Buffer
	err := renderITerm2(&out, bmpBuf.Bytes())
	if err != nil {
		t.Fatalf("renderITerm2: %v", err)
	}

	payload := extractITerm2Payload(t, out.String())
	if len(payload) < 4 {
		t.Fatal("payload too short")
	}

	// Should start with PNG magic bytes (re-encoded).
	if payload[0] != 0x89 || payload[1] != 'P' || payload[2] != 'N' || payload[3] != 'G' {
		t.Error("BMP input should be re-encoded to PNG (PNG magic bytes expected)")
	}
}

// extractITerm2Payload extracts and decodes the base64 payload from an
// iTerm2 OSC 1337 escape sequence.
func extractITerm2Payload(t *testing.T, output string) []byte {
	t.Helper()

	const prefix = "\x1b]1337;File=inline=1:"
	if !strings.HasPrefix(output, prefix) {
		t.Fatalf("output does not start with iTerm2 OSC 1337 prefix")
	}

	// Find the BEL terminator.
	rest := output[len(prefix):]
	belIdx := strings.IndexByte(rest, '\x07')
	if belIdx < 0 {
		t.Fatal("output missing BEL terminator")
	}

	encoded := rest[:belIdx]
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	return payload
}
