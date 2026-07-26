package unicode

import (
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ascii only", "hello", "hello"},
		{"chinese characters", "你好", "\\u4F60\\u597D"},
		{"mixed ascii and unicode", "Hello 世界", "Hello \\u4E16\\u754C"},
		{"with spaces", "a b c", "a b c"},
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
		{"ascii only", "hello", "hello"},
		{"chinese characters", "\\u4F60\\u597D", "你好"},
		{"mixed", "Hello \\u4E16\\u754C", "Hello 世界"},
		{"with spaces", "a b c", "a b c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Decode(tt.input)
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
	originals := []string{
		"Hello World!",
		"你好世界",
		"Hello 世界! @#$%",
		"a",
		"日本語",
		"😀🚀", // Emoji (surrogate pairs)
	}

	for _, original := range originals {
		t.Run(original, func(t *testing.T) {
			encoded := Encode(original)
			decoded, err := Decode(encoded)
			if err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
			if decoded != original {
				t.Errorf("Round trip failed: %q -> %q -> %q", original, encoded, decoded)
			}
		})
	}
}

func TestEncodeAll(t *testing.T) {
	result := EncodeAll("a")
	if result != "\\u0061" {
		t.Errorf("EncodeAll(\"a\") = %q, want %q", result, "\\u0061")
	}
}

func TestIsEncoded(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"hello", false},
		{"\\u4F60", true},
		{"\\u0041", true},
		{"Hello \\u4E16\\u754C", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsEncoded(tt.input)
			if got != tt.want {
				t.Errorf("IsEncoded(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDecodeInvalidEscape(t *testing.T) {
	_, err := Decode("\\uXYZ")
	if err == nil {
		t.Error("expected error for invalid escape sequence")
	}
}

func TestEncodeDecodeSpecialChars(t *testing.T) {
	input := "\n\t\r"
	encoded := Encode(input)
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if decoded != input {
		t.Errorf("Round trip failed: %q -> %q -> %q", input, encoded, decoded)
	}
}
