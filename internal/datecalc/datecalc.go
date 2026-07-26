package datecalc

import (
	"fmt"
	"math"
	"time"
)

// DateFormat is the expected date input format.
const DateFormat = "2006-01-02"

// DateTimeFormat for full datetime parsing.
const DateTimeFormat = "2006-01-02 15:04:05"

// AddDays adds/subtracts days from a date and returns the new date.
func AddDays(dateStr string, days int) (string, error) {
	t, err := time.Parse(DateFormat, dateStr)
	if err != nil {
		return "", fmt.Errorf("invalid date %q (expected format: YYYY-MM-DD)", dateStr)
	}

	result := t.AddDate(0, 0, days)
	return result.Format(DateFormat), nil
}

// DiffResult holds the difference between two dates.
type DiffResult struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Days      int64  `json:"days"`
	Minutes   int64  `json:"minutes"`
	Seconds   int64  `json:"seconds"`
}

// Diff calculates the difference between two dates.
func Diff(startDate, endDate string) (*DiffResult, error) {
	start, err := parseDateOrDateTime(startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date %q (expected format: YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)", startDate)
	}

	end, err := parseDateOrDateTime(endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date %q (expected format: YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)", endDate)
	}

	diff := end.Sub(start)
	totalSeconds := int64(math.Round(diff.Seconds()))
	totalMinutes := totalSeconds / 60
	totalDays := totalMinutes / (24 * 60)

	return &DiffResult{
		StartDate: start.Format(DateTimeFormat),
		EndDate:   end.Format(DateTimeFormat),
		Days:      totalDays,
		Minutes:   totalMinutes,
		Seconds:   totalSeconds,
	}, nil
}

// Now returns the current date as a formatted string.
func Now() string {
	return time.Now().Format(DateTimeFormat)
}

func parseDateOrDateTime(s string) (time.Time, error) {
	t, err := time.Parse(DateTimeFormat, s)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse(DateFormat, s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("cannot parse %q", s)
}
