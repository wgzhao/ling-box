package datecalc

import (
	"testing"
)

func TestAddDays(t *testing.T) {
	tests := []struct {
		date   string
		days   int
		want   string
	}{
		{"2026-01-01", 10, "2026-01-11"},
		{"2026-01-01", 0, "2026-01-01"},
		{"2026-01-01", -10, "2025-12-22"},
		{"2026-12-31", 1, "2027-01-01"},
		{"2026-03-01", -1, "2026-02-28"}, // not leap year
		{"2024-03-01", -1, "2024-02-29"}, // leap year
	}

	for _, tt := range tests {
		t.Run(tt.date+"+"+itoa(tt.days), func(t *testing.T) {
			got, err := AddDays(tt.date, tt.days)
			if err != nil {
				t.Fatalf("AddDays returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("AddDays(%q, %d) = %q, want %q", tt.date, tt.days, got, tt.want)
			}
		})
	}
}

func TestAddDaysInvalidDate(t *testing.T) {
	_, err := AddDays("not-a-date", 10)
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestDiff(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		end       string
		wantDays  int64
		wantMins  int64
		wantSecs  int64
	}{
		{"same day", "2026-01-01", "2026-01-01", 0, 0, 0},
		{"one day", "2026-01-01", "2026-01-02", 1, 1440, 86400},
		{"ten days", "2026-01-01", "2026-01-11", 10, 14400, 864000},
		{"negative", "2026-01-11", "2026-01-01", -10, -14400, -864000},
		{"year cross", "2026-12-31", "2027-01-01", 1, 1440, 86400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Diff(tt.start, tt.end)
			if err != nil {
				t.Fatalf("Diff returned error: %v", err)
			}
			if r.Days != tt.wantDays {
				t.Errorf("Days = %d, want %d", r.Days, tt.wantDays)
			}
			if r.Minutes != tt.wantMins {
				t.Errorf("Minutes = %d, want %d", r.Minutes, tt.wantMins)
			}
			if r.Seconds != tt.wantSecs {
				t.Errorf("Seconds = %d, want %d", r.Seconds, tt.wantSecs)
			}
		})
	}
}

func TestDiffWithTime(t *testing.T) {
	r, err := Diff("2026-01-01 12:00:00", "2026-01-02 12:00:00")
	if err != nil {
		t.Fatalf("Diff returned error: %v", err)
	}
	if r.Days != 1 {
		t.Errorf("Days = %d, want 1", r.Days)
	}
}

func TestDiffInvalidDate(t *testing.T) {
	_, err := Diff("not-a-date", "2026-01-01")
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestNow(t *testing.T) {
	s := Now()
	if len(s) == 0 {
		t.Error("Now() returned empty string")
	}
	// Should contain date in format YYYY-MM-DD HH:MM:SS
	if len(s) != 19 {
		t.Errorf("Now() length = %d, want 19", len(s))
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	n := i
	if n < 0 {
		s = "-"
		n = -n
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return s + string(digits)
}
