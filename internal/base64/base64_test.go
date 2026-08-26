package base64

import (
	"strings"
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple string", "hello", "aGVsbG8="},
		{"chinese characters", "你好", "5L2g5aW9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Encode(tt.input, false)
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
		{"simple string", "aGVsbG8=", "hello"},
		{"chinese characters", "5L2g5aW9", "你好"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Decode(tt.input, false)
			if err != nil {
				t.Fatalf("Decode(%q) returned error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("Decode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	original := "Hello World! 你好世界"
	encoded := Encode(original, false)
	decoded, err := Decode(encoded, false)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if decoded != original {
		t.Errorf("Round trip failed: %q -> %q -> %q", original, encoded, decoded)
	}
}

func TestURLSafeEncode(t *testing.T) {
	input := "test+/string"
	result := Encode(input, true)
	if strings.Contains(result, "+") || strings.Contains(result, "/") {
		t.Errorf("URL-safe encode should not contain + or /, got %q", result)
	}
}

func TestURLSafeRoundTrip(t *testing.T) {
	original := "test+/string?param=value"
	encoded := Encode(original, true)
	decoded, err := Decode(encoded, true)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if decoded != original {
		t.Errorf("Round trip failed: %q -> %q -> %q", original, encoded, decoded)
	}
}
