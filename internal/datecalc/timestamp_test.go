package datecalc

import (
	"testing"
	"time"
)

func mustLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	return loc
}

func TestParseTimestampUnixSeconds(t *testing.T) {
	got, err := ParseTimestamp("1735689600", TimestampOptions{Location: time.UTC})
	if err != nil {
		t.Fatal(err)
	}
	if got.Unix != 1735689600 || got.UnixMilli != 1735689600000 {
		t.Errorf("unix = %d/%d", got.Unix, got.UnixMilli)
	}
	if got.UTC != "2025-01-01T00:00:00Z" {
		t.Errorf("utc = %q", got.UTC)
	}
	if got.Local != "2025-01-01T00:00:00Z" {
		t.Errorf("local = %q", got.Local)
	}
	if got.Weekday != "Wednesday" {
		t.Errorf("weekday = %q", got.Weekday)
	}
}

func TestParseTimestampAutoMilliseconds(t *testing.T) {
	got, err := ParseTimestamp("1735689600000", TimestampOptions{Location: time.UTC})
	if err != nil {
		t.Fatal(err)
	}
	if got.Unix != 1735689600 {
		t.Errorf("unix = %d, want 1735689600", got.Unix)
	}
}

func TestParseTimestampForceMilliseconds(t *testing.T) {
	got, err := ParseTimestamp("1735689600", TimestampOptions{Location: time.UTC, ForceMillis: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Unix != 1735689 {
		t.Errorf("unix = %d, want 1735689", got.Unix)
	}
}

func TestParseTimestampFormats(t *testing.T) {
	shanghai := mustLocation(t, "Asia/Shanghai")
	tests := []struct {
		input string
		loc   *time.Location
	}{
		{"2025-01-01T08:00:00+08:00", time.UTC},
		{"2025-01-01 08:00:00", shanghai},
		{"2025-01-01T08:00:00", shanghai},
		{"2025-01-01", time.UTC},
	}
	for _, tt := range tests {
		got, err := ParseTimestamp(tt.input, TimestampOptions{Location: tt.loc})
		if err != nil {
			t.Errorf("%q: %v", tt.input, err)
			continue
		}
		if tt.input == "2025-01-01" {
			if got.Unix != 1735689600 {
				t.Errorf("%q: unix = %d, want 1735689600", tt.input, got.Unix)
			}
			continue
		}
		if got.Unix != 1735689600 {
			t.Errorf("%q: unix = %d, want 1735689600", tt.input, got.Unix)
		}
	}
}

func TestParseTimestampLocalTimezone(t *testing.T) {
	shanghai := mustLocation(t, "Asia/Shanghai")
	got, err := ParseTimestamp("1735689600", TimestampOptions{Location: shanghai})
	if err != nil {
		t.Fatal(err)
	}
	if got.Local != "2025-01-01T08:00:00+08:00" {
		t.Errorf("local = %q", got.Local)
	}
	if got.Timezone != "CST +08:00" {
		t.Errorf("timezone = %q", got.Timezone)
	}
}

func TestParseTimestampNegative(t *testing.T) {
	got, err := ParseTimestamp("-1", TimestampOptions{Location: time.UTC})
	if err != nil {
		t.Fatal(err)
	}
	if got.UTC != "1969-12-31T23:59:59Z" {
		t.Errorf("utc = %q", got.UTC)
	}
}

func TestParseTimestampErrors(t *testing.T) {
	for _, input := range []string{"", "not-a-date", "2025-13-01", "2025-01-01 25:00:00"} {
		if _, err := ParseTimestamp(input, TimestampOptions{Location: time.UTC}); err == nil {
			t.Errorf("ParseTimestamp(%q): expected error", input)
		}
	}
}
