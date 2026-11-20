package cronx

import (
	"strings"
	"testing"
	"time"
)

func parseAt(t *testing.T, expr, from string) *Schedule {
	t.Helper()
	s, err := Parse(expr, time.UTC)
	if err != nil {
		t.Fatalf("Parse(%q): %v", expr, err)
	}
	if _, err := time.Parse(time.RFC3339, from); err != nil {
		t.Fatalf("bad test time %q: %v", from, err)
	}
	return s
}

func TestNextMinuteLevel(t *testing.T) {
	s := parseAt(t, "*/15 * * * *", "2026-01-01T00:00:00Z")
	from, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	got, err := s.Next(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-01-01T00:15:00Z", "2026-01-01T00:30:00Z", "2026-01-01T00:45:00Z"}
	for i, w := range want {
		if got[i].Format(time.RFC3339) != w {
			t.Errorf("run %d = %s, want %s", i, got[i].Format(time.RFC3339), w)
		}
	}
}

func TestNextWeekdays(t *testing.T) {
	s, err := Parse("0 9 * * MON-FRI", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	from, _ := time.Parse(time.RFC3339, "2026-01-02T00:00:00Z") // Friday
	got, err := s.Next(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-01-02T09:00:00Z", "2026-01-05T09:00:00Z", "2026-01-06T09:00:00Z"}
	for i, w := range want {
		if got[i].Format(time.RFC3339) != w {
			t.Errorf("run %d = %s, want %s", i, got[i].Format(time.RFC3339), w)
		}
	}
}

func TestDayOfMonthOrDayOfWeek(t *testing.T) {
	s, err := Parse("0 0 13 * FRI", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	from, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	got, err := s.Next(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-01-02T00:00:00Z", "2026-01-09T00:00:00Z", "2026-01-13T00:00:00Z"}
	for i, w := range want {
		if got[i].Format(time.RFC3339) != w {
			t.Errorf("run %d = %s, want %s", i, got[i].Format(time.RFC3339), w)
		}
	}
}

func TestSecondLevelAndMacros(t *testing.T) {
	s, err := Parse("*/10 * * * * *", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	from, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	got, err := s.Next(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-01-01T00:00:10Z", "2026-01-01T00:00:20Z", "2026-01-01T00:00:30Z"}
	for i, w := range want {
		if got[i].Format(time.RFC3339) != w {
			t.Errorf("run %d = %s, want %s", i, got[i].Format(time.RFC3339), w)
		}
	}

	daily, err := Parse("@daily", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	got, err = daily.Next(from, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Format(time.RFC3339) != "2026-01-02T00:00:00Z" {
		t.Errorf("@daily next = %s", got[0].Format(time.RFC3339))
	}
}

func TestMonthAndSundayNames(t *testing.T) {
	s, err := Parse("0 0 1 JAN *", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	from, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	got, err := s.Next(from, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Format(time.RFC3339) != "2027-01-01T00:00:00Z" {
		t.Errorf("January run = %s", got[0].Format(time.RFC3339))
	}

	sun, err := Parse("0 0 * * 7", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	from, _ = time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	got, err = sun.Next(from, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Format(time.RFC3339) != "2026-01-04T00:00:00Z" {
		t.Errorf("Sunday run = %s", got[0].Format(time.RFC3339))
	}
}

func TestExplain(t *testing.T) {
	s, err := Parse("*/5 * * * *", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Join(s.Explain(), "\n")
	if !strings.Contains(lines, "every 5 minutes") {
		t.Errorf("explain = %s", lines)
	}

	s, err = Parse("0 9 * * MON-FRI", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	lines = strings.Join(s.Explain(), "\n")
	for _, want := range []string{"minute 0", "hour 9", "Monday", "Friday"} {
		if !strings.Contains(lines, want) {
			t.Errorf("explain missing %q:\n%s", want, lines)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, expr := range []string{
		"", "* * *", "* * * * * * *",
		"60 * * * *", "*/0 * * * *", "0 0 * * FUNDAY", "@bad",
		"0 0 32 * *", "0 0 * 13 *", "*/x * * * *",
	} {
		if _, err := Parse(expr, time.UTC); err == nil {
			t.Errorf("Parse(%q): expected error", expr)
		}
	}
}

func TestNextImpossibleDate(t *testing.T) {
	s, err := Parse("0 0 31 2 *", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	from, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	if _, err := s.Next(from, 1); err == nil {
		t.Error("Feb 31: expected no-match error")
	}
}
