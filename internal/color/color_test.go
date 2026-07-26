package color

import (
	"testing"
)

func TestConvertHex(t *testing.T) {
	tests := []struct {
		input    string
		wantHex  string
		wantRGB  string
		wantName string
	}{
		{"#FF0000", "#FF0000", "rgb(255, 0, 0)", "red"},
		{"#00FF00", "#00FF00", "rgb(0, 255, 0)", "lime"},
		{"#0000FF", "#0000FF", "rgb(0, 0, 255)", "blue"},
		{"#000000", "#000000", "rgb(0, 0, 0)", "black"},
		{"#FFFFFF", "#FFFFFF", "rgb(255, 255, 255)", "white"},
		{"FF0000", "#FF0000", "rgb(255, 0, 0)", ""},       // no #
		{"#A9A9A9", "#A9A9A9", "rgb(169, 169, 169)", "dark gray"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, err := Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert(%q) returned error: %v", tt.input, err)
			}
			if r.Hex != tt.wantHex {
				t.Errorf("Hex = %q, want %q", r.Hex, tt.wantHex)
			}
			if r.RGB != tt.wantRGB {
				t.Errorf("RGB = %q, want %q", r.RGB, tt.wantRGB)
			}
		})
	}
}

func TestConvertShortHex(t *testing.T) {
	r, err := Convert("#F00")
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if r.Hex != "#FF0000" {
		t.Errorf("Hex = %q, want %q", r.Hex, "#FF0000")
	}
}

func TestConvertRGB(t *testing.T) {
	tests := []struct {
		input   string
		wantHex string
		wantHSL string
	}{
		{"rgb(255, 0, 0)", "#FF0000", "hsl(0, 100%, 50%)"},
		{"rgb(0, 255, 0)", "#00FF00", "hsl(120, 100%, 50%)"},
		{"rgb(0, 0, 255)", "#0000FF", "hsl(240, 100%, 50%)"},
		{"100, 100, 100", "#646464", "hsl(0, 0%, 39%)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, err := Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert(%q) returned error: %v", tt.input, err)
			}
			if r.Hex != tt.wantHex {
				t.Errorf("Hex = %q, want %q", r.Hex, tt.wantHex)
			}
			if r.HSL != tt.wantHSL {
				t.Errorf("HSL = %q, want %q", r.HSL, tt.wantHSL)
			}
		})
	}
}

func TestConvertHSL(t *testing.T) {
	tests := []struct {
		input   string
		wantHex string
		wantRGB string
	}{
		{"hsl(0, 100%, 50%)", "#FF0000", "rgb(255, 0, 0)"},
		{"hsl(120, 100%, 50%)", "#00FF00", "rgb(0, 255, 0)"},
		{"hsl(240, 100%, 50%)", "#0000FF", "rgb(0, 0, 255)"},
		{"hsl(0, 0%, 0%)", "#000000", "rgb(0, 0, 0)"},
		{"hsl(0, 0%, 100%)", "#FFFFFF", "rgb(255, 255, 255)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, err := Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert(%q) returned error: %v", tt.input, err)
			}
			if r.Hex != tt.wantHex {
				t.Errorf("Hex = %q, want %q", r.Hex, tt.wantHex)
			}
			if r.RGB != tt.wantRGB {
				t.Errorf("RGB = %q, want %q", r.RGB, tt.wantRGB)
			}
		})
	}
}

func TestConvertNamedColors(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"red", "#FF0000"},
		{"dark gray", "#A9A9A9"},
		{"light yellow", "#FFFFE0"},
		{"blue", "#0000FF"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, err := Convert(tt.input)
			if err != nil {
				t.Fatalf("Convert(%q) returned error: %v", tt.input, err)
			}
			if r.Hex != tt.want {
				t.Errorf("Hex = %q, want %q", r.Hex, tt.want)
			}
		})
	}
}

func TestConvertErrors(t *testing.T) {
	_, err := Convert("")
	if err == nil {
		t.Error("expected error for empty input")
	}

	_, err = Convert("#GGG")
	if err == nil {
		t.Error("expected error for invalid hex")
	}

	_, err = Convert("rgb(300, 0, 0)")
	if err == nil {
		t.Error("expected error for RGB values out of range")
	}
}

func TestHSL(t *testing.T) {
	// Red
	r := rgbToHSL(Color{255, 0, 0})
	if mathRound(r.H) != 0 || mathRound(r.S) != 100 || mathRound(r.L) != 50 {
		t.Errorf("Red HSL = %.0f, %.0f, %.0f; want 0, 100, 50", r.H, r.S, r.L)
	}

	// Green
	g := rgbToHSL(Color{0, 255, 0})
	if mathRound(g.H) != 120 || mathRound(g.S) != 100 || mathRound(g.L) != 50 {
		t.Errorf("Green HSL = %.0f, %.0f, %.0f; want 120, 100, 50", g.H, g.S, g.L)
	}

	// Blue
	b := rgbToHSL(Color{0, 0, 255})
	if mathRound(b.H) != 240 || mathRound(b.S) != 100 || mathRound(b.L) != 50 {
		t.Errorf("Blue HSL = %.0f, %.0f, %.0f; want 240, 100, 50", b.H, b.S, b.L)
	}

	// White
	w := rgbToHSL(Color{255, 255, 255})
	if mathRound(w.L) != 100 {
		t.Errorf("White lightness = %.0f; want 100", w.L)
	}

	// Black
	bl := rgbToHSL(Color{0, 0, 0})
	if mathRound(bl.L) != 0 {
		t.Errorf("Black lightness = %.0f; want 0", bl.L)
	}
}

func TestRoundTripHSL(t *testing.T) {
	colors := []Color{
		{255, 0, 0},
		{0, 255, 0},
		{0, 0, 255},
		{128, 128, 128},
		{255, 255, 0},
		{0, 255, 255},
		{255, 0, 255},
		{42, 128, 200},
		{10, 20, 30},
	}

	for _, c := range colors {
		hsl := rgbToHSL(c)
		restored := hslToRGB(hsl)
		if restored.R != c.R || restored.G != c.G || restored.B != c.B {
			t.Errorf("Round trip failed for RGB(%d,%d,%d): got RGB(%d,%d,%d)",
				c.R, c.G, c.B, restored.R, restored.G, restored.B)
		}
	}
}

func mathRound(f float64) float64 {
	if f < 0 {
		return float64(int(f - 0.5))
	}
	return float64(int(f + 0.5))
}
