package color

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Color represents an RGB color with components 0-255.
type Color struct {
	R, G, B int
}

// Format describes the color format.
type Format string

const (
	FormatHex  Format = "hex"
	FormatRGB  Format = "rgb"
	FormatHSL  Format = "hsl"
	FormatName Format = "name"
)

// HSL represents a color in HSL space.
type HSL struct {
	H, S, L float64 // H: 0-360, S: 0-100, L: 0-100
}

// Result holds all representations of a single color.
type Result struct {
	Input    string `json:"input"`
	Format   Format `json:"format"`
	Hex      string `json:"hex"`
	RGB      string `json:"rgb"`
	HSL      string `json:"hsl"`
	Name     string `json:"name,omitempty"`
}

// namedColors maps common color names to hex values.
var namedColors = map[string]string{
	"red":          "#FF0000",
	"dark red":     "#8B0000",
	"crimson":      "#DC143C",
	"firebrick":    "#B22222",
	"tomato":       "#FF6347",
	"coral":        "#FF7F50",
	"salmon":       "#FA8072",
	"light coral":  "#F08080",
	"pink":         "#FFC0CB",
	"light pink":   "#FFB6C1",
	"hot pink":     "#FF69B4",
	"deeppink":     "#FF1493",
	"blue":         "#0000FF",
	"dark blue":    "#00008B",
	"medium blue":  "#0000CD",
	"royal blue":   "#4169E1",
	"sky blue":     "#87CEEB",
	"light blue":   "#ADD8E6",
	"steel blue":   "#4682B4",
	"navy":         "#000080",
	"green":        "#008000",
	"dark green":   "#006400",
	"light green":  "#90EE90",
	"lime":         "#00FF00",
	"forest green": "#228B22",
	"sea green":    "#2E8B57",
	"olive":        "#808000",
	"yellow":       "#FFFF00",
	"light yellow": "#FFFFE0",
	"gold":         "#FFD700",
	"orange":       "#FFA500",
	"dark orange":  "#FF8C00",
	"purple":       "#800080",
	"dark magenta": "#8B008B",
	"magenta":      "#FF00FF",
	"violet":       "#EE82EE",
	"plum":         "#DDA0DD",
	"orchid":       "#DA70D6",
	"cyan":         "#00FFFF",
	"teal":         "#008080",
	"turquoise":    "#40E0D0",
	"brown":        "#A52A2A",
	"saddle brown": "#8B4513",
	"sandy brown":  "#F4A460",
	"chocolate":    "#D2691E",
	"white":        "#FFFFFF",
	"snow":         "#FFFAFA",
	"ivory":        "#FFFFF0",
	"gray":         "#808080",
	"dark gray":    "#A9A9A9",
	"light gray":   "#D3D3D3",
	"silver":       "#C0C0C0",
	"dim gray":     "#696969",
	"black":        "#000000",
	"beige":        "#F5F5DC",
	"wheat":        "#F5DEB3",
	"tan":          "#D2B48C",
	"lavender":     "#E6E6FA",
	"thistle":      "#D8BFD8",
	"honeydew":     "#F0FFF0",
	"azure":        "#F0FFFF",
	"alice blue":   "#F0F8FF",
}

var (
	hexRegex = regexp.MustCompile(`^#?([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$`)
	rgbRegex = regexp.MustCompile(`(?i)^(?:rgb\s*)?\(\s*(\d+)\s*[,]\s*(\d+)\s*[,]\s*(\d+)\s*\)$|^(\d+)\s*[,]\s*(\d+)\s*[,]\s*(\d+)\s*$`)
	hslRegex = regexp.MustCompile(`(?i)^(?:hsl\s*)?\(\s*(\d+(?:\.\d+)?)\s*[,]\s*(\d+(?:\.\d+)?)%?\s*[,]\s*(\d+(?:\.\d+)?)%?\s*\)$`)
)

// Convert takes any color input and returns all representations.
func Convert(input string) (*Result, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Try named colors first
	if hex, ok := namedColors[strings.ToLower(input)]; ok {
		c, err := hexToColor(hex)
		if err != nil {
			return nil, err
		}
		return makeResult(input, FormatName, c), nil
	}

	// Try hex
	if m := hexRegex.FindStringSubmatch(input); m != nil {
		c, err := hexToColor(input)
		if err != nil {
			return nil, err
		}
		return makeResult(input, FormatHex, c), nil
	}

	// Try RGB
	if m := rgbRegex.FindStringSubmatch(input); m != nil {
		// The regex has two alternatives: with "rgb(" wrapper (groups 1-3) and without (groups 4-6)
		var r, g, b int
		if m[1] != "" {
			r, _ = strconv.Atoi(m[1])
			g, _ = strconv.Atoi(m[2])
			b, _ = strconv.Atoi(m[3])
		} else {
			r, _ = strconv.Atoi(m[4])
			g, _ = strconv.Atoi(m[5])
			b, _ = strconv.Atoi(m[6])
		}
		if r < 0 || r > 255 || g < 0 || g > 255 || b < 0 || b > 255 {
			return nil, fmt.Errorf("RGB values must be 0-255")
		}
		c := Color{r, g, b}
		return makeResult(input, FormatRGB, c), nil
	}

	// Try HSL
	if m := hslRegex.FindStringSubmatch(input); m != nil {
		h, _ := strconv.ParseFloat(m[1], 64)
		s, _ := strconv.ParseFloat(m[2], 64)
		l, _ := strconv.ParseFloat(m[3], 64)
		if h < 0 || h > 360 || s < 0 || s > 100 || l < 0 || l > 100 {
			return nil, fmt.Errorf("HSL values out of range: H(0-360), S(0-100), L(0-100)")
		}
		c := hslToRGB(HSL{h, s, l})
		return makeResult(input, FormatHSL, c), nil
	}

	return nil, fmt.Errorf("unable to parse color: %q", input)
}

func makeResult(input string, format Format, c Color) *Result {
	r := &Result{
		Input:  input,
		Format: format,
		Hex:    colorToHex(c),
		RGB:    colorToRGB(c),
		HSL:    colorToHSLString(c),
		Name:   findName(c),
	}
	return r
}

func colorToHex(c Color) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

func colorToRGB(c Color) string {
	return fmt.Sprintf("rgb(%d, %d, %d)", c.R, c.G, c.B)
}

func colorToHSLString(c Color) string {
	hsl := rgbToHSL(c)
	return fmt.Sprintf("hsl(%.0f, %.0f%%, %.0f%%)", math.Round(hsl.H), math.Round(hsl.S), math.Round(hsl.L))
}

func hexToColor(hex string) (Color, error) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return Color{}, fmt.Errorf("invalid hex color: %s", hex)
	}

	r, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return Color{}, fmt.Errorf("invalid hex color: %s", hex)
	}
	g, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return Color{}, fmt.Errorf("invalid hex color: %s", hex)
	}
	b, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return Color{}, fmt.Errorf("invalid hex color: %s", hex)
	}

	return Color{int(r), int(g), int(b)}, nil
}

func rgbToHSL(c Color) HSL {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0

	maxV := max(r, g, b)
	minV := min(r, g, b)
	delta := maxV - minV

	var h, s, l float64
	l = (maxV + minV) / 2.0

	if delta == 0 {
		h = 0
		s = 0
	} else {
		if l > 0.5 {
			s = delta / (2.0 - maxV - minV)
		} else {
			s = delta / (maxV + minV)
		}

		switch maxV {
		case r:
			h = math.Mod((g-b)/delta, 6.0)
		case g:
			h = (b-r)/delta + 2.0
		case b:
			h = (r-g)/delta + 4.0
		}
		h *= 60.0
		if h < 0 {
			h += 360.0
		}
	}

	return HSL{h, s * 100, l * 100}
}

func hslToRGB(hsl HSL) Color {
	h := hsl.H / 360.0
	s := hsl.S / 100.0
	l := hsl.L / 100.0

	var r, g, b float64

	if s == 0 {
		r = l
		g = l
		b = l
	} else {
		var v1, v2 float64
		if l < 0.5 {
			v2 = l * (1 + s)
		} else {
			v2 = (l + s) - (s * l)
		}
		v1 = 2*l - v2

		r = hueToRGB(v1, v2, h+1.0/3.0)
		g = hueToRGB(v1, v2, h)
		b = hueToRGB(v1, v2, h-1.0/3.0)
	}

	return Color{
		R: clamp(int(math.Round(r * 255.0))),
		G: clamp(int(math.Round(g * 255.0))),
		B: clamp(int(math.Round(b * 255.0))),
	}
}

func hueToRGB(v1, v2, h float64) float64 {
	if h < 0 {
		h++
	}
	if h > 1 {
		h--
	}
	if h*6 < 1 {
		return v1 + (v2-v1)*6*h
	}
	if h*2 < 1 {
		return v2
	}
	if h*3 < 2 {
		return v1 + (v2-v1)*(2.0/3.0-h)*6
	}
	return v1
}

func clamp(v int) int {
	return min(max(v, 0), 255)
}

func findName(c Color) string {
	hex := colorToHex(c)
	// Exact match
	for name, h := range namedColors {
		if strings.EqualFold(h, hex) {
			return name
		}
	}

	// Fuzzy match - find closest named color by Euclidean distance
	minDist := math.MaxFloat64
	closest := ""
	for name, h := range namedColors {
		nc, err := hexToColor(h)
		if err != nil {
			continue
		}
		dr := float64(c.R - nc.R)
		dg := float64(c.G - nc.G)
		db := float64(c.B - nc.B)
		dist := dr*dr + dg*dg + db*db
		if dist < minDist {
			minDist = dist
			closest = name
		}
	}
	if minDist < 2000 { // threshold for "close enough"
		return closest + " (approx)"
	}
	return ""
}
