package url

import (
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple string", "hello world", "hello+world"},
		{"special characters", "hello&world=test", "hello%26world%3Dtest"},
		{"chinese characters", "你好", "%E4%BD%A0%E5%A5%BD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Encode(tt.input)
			if result != tt.expected {
				t.Errorf("Encode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple string", "hello+world", "hello world"},
		{"special characters", "hello%26world%3Dtest", "hello&world=test"},
		{"chinese characters", "%E4%BD%A0%E5%A5%BD", "你好"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Decode(tt.input)
			if result != tt.expected {
				t.Errorf("Decode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	original := "Hello World! 你好世界 @#$%"
	encoded := Encode(original)
	decoded := Decode(encoded)
	if decoded != original {
		t.Errorf("Round trip failed: %q -> %q -> %q", original, encoded, decoded)
	}
}
