package baseconv

import (
	"testing"
)

func TestConvertDecimal(t *testing.T) {
	r, err := Convert("255", Dec)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	if r.Binary != "11111111" {
		t.Errorf("Binary = %q, want %q", r.Binary, "11111111")
	}
	if r.Octal != "377" {
		t.Errorf("Octal = %q, want %q", r.Octal, "377")
	}
	if r.Decimal != "255" {
		t.Errorf("Decimal = %q, want %q", r.Decimal, "255")
	}
	if r.Hex != "FF" {
		t.Errorf("Hex = %q, want %q", r.Hex, "FF")
	}
}

func TestConvertHex(t *testing.T) {
	r, err := Convert("FF", Hex)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if r.Decimal != "255" {
		t.Errorf("Decimal = %q, want %q", r.Decimal, "255")
	}
}

func TestConvertHexWithPrefix(t *testing.T) {
	r, err := Convert("0xFF", Hex)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if r.Decimal != "255" {
		t.Errorf("Decimal = %q, want %q", r.Decimal, "255")
	}
}

func TestConvertBinary(t *testing.T) {
	r, err := Convert("1010", Bin)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if r.Decimal != "10" {
		t.Errorf("Decimal = %q, want %q", r.Decimal, "10")
	}
	if r.Hex != "A" {
		t.Errorf("Hex = %q, want %q", r.Hex, "A")
	}
}

func TestConvertOctal(t *testing.T) {
	r, err := Convert("377", Oct)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if r.Decimal != "255" {
		t.Errorf("Decimal = %q, want %q", r.Decimal, "255")
	}
	if r.Binary != "11111111" {
		t.Errorf("Binary = %q, want %q", r.Binary, "11111111")
	}
}

func TestAutoDetect(t *testing.T) {
	tests := []struct {
		input string
		want  Base
	}{
		{"0xFF", Hex},
		{"0x1A", Hex},
		{"0b1010", Bin},
		{"0777", Oct},
		{"255", Dec},
		{"ABC", Hex},
		{"123", Dec},
		{"0", Dec},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := AutoDetect(tt.input)
			if err != nil {
				t.Fatalf("AutoDetect returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("AutoDetect(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertLarge(t *testing.T) {
	r, err := Convert("18446744073709551615", Dec)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if r.Hex != "FFFFFFFFFFFFFFFF" {
		t.Errorf("Hex = %q, want %q", r.Hex, "FFFFFFFFFFFFFFFF")
	}
	if r.Binary != "1111111111111111111111111111111111111111111111111111111111111111" {
		t.Errorf("Binary mismatch for max uint64")
	}
}

func TestConvertErrors(t *testing.T) {
	_, err := Convert("", Dec)
	if err == nil {
		t.Error("expected error for empty input")
	}

	_, err = Convert("XYZ", Dec)
	if err == nil {
		t.Error("expected error for invalid decimal")
	}

	_, err = Convert("102", Bin)
	if err == nil {
		t.Error("expected error for invalid binary")
	}
}

func TestParseBase(t *testing.T) {
	tests := []struct {
		input string
		want  Base
	}{
		{"bin", Bin},
		{"oct", Oct},
		{"dec", Dec},
		{"hex", Hex},
		{"HEX", Hex},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseBase(tt.input)
			if err != nil {
				t.Fatalf("ParseBase returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseBase(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}

	_, err := ParseBase("invalid")
	if err == nil {
		t.Error("expected error for invalid base")
	}
}
