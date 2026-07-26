package uuid

import (
	"testing"
)

func TestGenerateDefault(t *testing.T) {
	ids, err := Generate(1, DefaultType)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 UUID, got %d", len(ids))
	}
	if !IsValid(ids[0]) {
		t.Errorf("invalid UUID: %s", ids[0])
	}
}

func TestGenerateV4(t *testing.T) {
	ids, err := Generate(5, TypeV4)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(ids) != 5 {
		t.Fatalf("expected 5 UUIDs, got %d", len(ids))
	}

	seen := make(map[string]bool)
	for _, id := range ids {
		if !IsValid(id) {
			t.Errorf("invalid UUID: %s", id)
		}
		// All V4 UUIDs have version nibble 4 at position 14
		if id[14] != '4' {
			t.Errorf("expected version 4 at position 14, got %c in %s", id[14], id)
		}
		if seen[id] {
			t.Errorf("duplicate UUID: %s", id)
		}
		seen[id] = true
	}
}

func TestGenerateV7(t *testing.T) {
	ids, err := Generate(1, TypeV7)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 UUID, got %d", len(ids))
	}
	if !IsValid(ids[0]) {
		t.Errorf("invalid UUID: %s", ids[0])
	}
	// V7 UUIDs have version nibble 7 at position 14
	if ids[0][14] != '7' {
		t.Errorf("expected version 7 at position 14, got %c in %s", ids[0][14], ids[0])
	}
}

func TestGenerateV1(t *testing.T) {
	ids, err := Generate(1, TypeV1)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 UUID, got %d", len(ids))
	}
	if !IsValid(ids[0]) {
		t.Errorf("invalid UUID: %s", ids[0])
	}
	// V1 UUIDs have version nibble 1 at position 14
	if ids[0][14] != '1' {
		t.Errorf("expected version 1 at position 14, got %c in %s", ids[0][14], ids[0])
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"not-a-uuid", false},
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"00000000-0000-0000-0000-000000000000", true},
		// missing hyphens — still valid, Parse accepts it
		{"550e8400e29b41d4a716446655440000", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsValid(tt.input)
			if got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	// Round-trip: generate and parse
	ids, _ := Generate(1, TypeV4)
	parsed, err := Parse(ids[0])
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if parsed != ids[0] {
		t.Errorf("Parse(%q) = %q, want %q", ids[0], parsed, ids[0])
	}

	// Invalid
	_, err = Parse("not-a-uuid")
	if err == nil {
		t.Error("expected error for invalid UUID")
	}
}
