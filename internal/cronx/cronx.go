// Package cronx parses cron expressions, explains them, and computes the next
// run times.
package cronx

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

var macros = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

var monthNames = map[string]int{
	"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
	"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
}

var dowNames = map[string]int{
	"SUN": 0, "MON": 1, "TUE": 2, "WED": 3, "THU": 4, "FRI": 5, "SAT": 6,
}

var monthLabels = []string{"", "January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December"}

var dowLabels = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

type field struct {
	label  string
	star   bool
	min    int
	max    int
	values map[int]bool
}

func (f *field) has(v int) bool { return f.values[v] }

func (f *field) list() []int {
	out := make([]int, 0, len(f.values))
	for v := range f.values {
		out = append(out, v)
	}
	sort.Ints(out)
	return out
}

// Schedule is a parsed cron expression.
type Schedule struct {
	Expr    string
	Seconds bool

	loc    *time.Location
	second *field
	minute *field
	hour   *field
	dom    *field
	month  *field
	dow    *field
}

// Parse accepts a standard 5-field expression, a 6-field expression with a
// leading seconds field, or an @macro.
func Parse(expr string, loc *time.Location) (*Schedule, error) {
	raw := strings.TrimSpace(expr)
	if raw == "" {
		return nil, fmt.Errorf("empty cron expression")
	}
	if strings.HasPrefix(raw, "@") {
		expanded, ok := macros[strings.ToLower(raw)]
		if !ok {
			return nil, fmt.Errorf("unknown macro %q (supported: @yearly, @annually, @monthly, @weekly, @daily, @midnight, @hourly)", raw)
		}
		raw = expanded
	}

	parts := strings.Fields(raw)
	if len(parts) != 5 && len(parts) != 6 {
		return nil, fmt.Errorf("expected 5 or 6 fields, got %d", len(parts))
	}
	if loc == nil {
		loc = time.Local
	}

	s := &Schedule{Expr: expr, Seconds: len(parts) == 6, loc: loc}
	var err error
	if s.Seconds {
		s.second, err = parseField(parts[0], "second", 0, 59, nil, false)
		if err != nil {
			return nil, err
		}
		parts = parts[1:]
	}

	parsers := []struct {
		dst           **field
		label         string
		min, max      int
		names         map[string]int
		allowQuestion bool
	}{
		{&s.minute, "minute", 0, 59, nil, false},
		{&s.hour, "hour", 0, 23, nil, false},
		{&s.dom, "day of month", 1, 31, nil, true},
		{&s.month, "month", 1, 12, monthNames, false},
		{&s.dow, "day of week", 0, 7, dowNames, true},
	}
	for i, p := range parsers {
		*p.dst, err = parseField(parts[i], p.label, p.min, p.max, p.names, p.allowQuestion)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func parseField(raw, label string, min, max int, names map[string]int, allowQuestion bool) (*field, error) {
	if raw == "?" {
		if !allowQuestion {
			return nil, fmt.Errorf("%s: ? is only allowed for day fields", label)
		}
		raw = "*"
	}

	f := &field{
		label:  label,
		star:   raw == "*",
		min:    min,
		max:    max,
		values: map[int]bool{},
	}

	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("%s: empty list element", label)
		}

		rangePart := part
		step := 1
		if i := strings.IndexByte(part, '/'); i >= 0 {
			if i == 0 || i == len(part)-1 {
				return nil, fmt.Errorf("%s: invalid step in %q", label, part)
			}
			step, _ = strconv.Atoi(part[i+1:])
			if step <= 0 {
				return nil, fmt.Errorf("%s: step must be positive in %q", label, part)
			}
			rangePart = part[:i]
		}

		lo, hi, err := parseRange(rangePart, min, max, names)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", label, err)
		}
		if rangePart != "*" && !strings.Contains(rangePart, "-") && step > 1 {
			hi = max
		}

		for v := lo; v <= hi; v += step {
			nv := v
			if nv == 7 && max == 7 {
				nv = 0
			}
			f.values[nv] = true
		}
	}

	if len(f.values) == 0 {
		return nil, fmt.Errorf("%s: no matching values", label)
	}
	return f, nil
}

func parseRange(s string, min, max int, names map[string]int) (int, int, error) {
	parseValue := func(v string) (int, error) {
		v = strings.TrimSpace(v)
		if names != nil {
			if n, ok := names[strings.ToUpper(v)]; ok {
				return n, nil
			}
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("invalid value %q", v)
		}
		if n < min || n > max {
			return 0, fmt.Errorf("value %d out of range %d-%d", n, min, max)
		}
		return n, nil
	}

	if s == "*" {
		return min, max, nil
	}
	if i := strings.IndexByte(s, '-'); i >= 0 {
		lo, err := parseValue(s[:i])
		if err != nil {
			return 0, 0, err
		}
		hi, err := parseValue(s[i+1:])
		if err != nil {
			return 0, 0, err
		}
		if lo > hi {
			return 0, 0, fmt.Errorf("range start %d is after end %d", lo, hi)
		}
		return lo, hi, nil
	}
	v, err := parseValue(s)
	if err != nil {
		return 0, 0, err
	}
	return v, v, nil
}

// Next returns the next n matching times strictly after from.
func (s *Schedule) Next(from time.Time, n int) ([]time.Time, error) {
	if n <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}

	t := from.In(s.loc)
	if s.Seconds {
		t = t.Truncate(time.Second).Add(time.Second)
	} else {
		t = t.Truncate(time.Minute).Add(time.Minute)
	}
	horizon := t.AddDate(5, 0, 0)

	out := make([]time.Time, 0, n)
	for len(out) < n {
		if t.After(horizon) {
			return out, fmt.Errorf("no matching time within 5 years; check the day/month combination")
		}

		if !s.month.has(int(t.Month())) {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, s.loc)
			continue
		}
		if !s.dayMatches(t) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, s.loc)
			continue
		}
		if !s.hour.has(t.Hour()) {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, s.loc)
			continue
		}
		if !s.minute.has(t.Minute()) {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+1, 0, 0, s.loc)
			continue
		}
		if s.Seconds && !s.second.has(t.Second()) {
			t = t.Add(time.Second)
			continue
		}

		out = append(out, t)
		if s.Seconds {
			t = t.Add(time.Second)
		} else {
			t = t.Add(time.Minute)
		}
	}
	return out, nil
}

func (s *Schedule) dayMatches(t time.Time) bool {
	domOK := s.dom.has(t.Day())
	dowOK := s.dow.has(int(t.Weekday()))
	if !s.dom.star && !s.dow.star {
		return domOK || dowOK
	}
	return domOK && dowOK
}

// Explain returns one human-readable line per field.
func (s *Schedule) Explain() []string {
	var lines []string
	if s.Seconds {
		lines = append(lines, "Second      : "+describe(s.second, "second", nil))
	}
	lines = append(lines,
		"Minute      : "+describe(s.minute, "minute", nil),
		"Hour        : "+describe(s.hour, "hour", nil),
		"Day of month: "+describe(s.dom, "day", nil),
		"Month       : "+describe(s.month, "month", monthLabels),
		"Day of week : "+describe(s.dow, "weekday", dowLabels),
	)
	return lines
}

func describe(f *field, kind string, labels []string) string {
	if f.star {
		switch kind {
		case "day", "weekday":
			return "every day"
		case "month":
			return "every month"
		default:
			return "every " + kind
		}
	}

	label := func(v int) string {
		if labels != nil && v >= 0 && v < len(labels) && labels[v] != "" {
			return labels[v]
		}
		return strconv.Itoa(v)
	}

	vals := f.list()
	if step, ok := arithmeticStep(vals); ok && vals[0] == f.min && vals[len(vals)-1]+step > f.max {
		if step == 1 {
			return "every " + kind
		}
		unit := kind + "s"
		switch kind {
		case "day":
			unit = "days"
		case "weekday":
			unit = "days"
		case "month":
			unit = "months"
		}
		return fmt.Sprintf("every %d %s", step, unit)
	}

	if len(vals) == 1 {
		switch kind {
		case "month":
			return "in " + label(vals[0])
		case "weekday":
			return "on " + label(vals[0])
		default:
			return "at " + kind + " " + label(vals[0])
		}
	}

	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = label(v)
	}
	switch kind {
	case "month", "weekday":
		return strings.Join(parts, ", ")
	default:
		return kind + "s " + strings.Join(parts, ", ")
	}
}

func arithmeticStep(vals []int) (int, bool) {
	if len(vals) < 3 {
		return 0, false
	}
	step := vals[1] - vals[0]
	if step <= 0 {
		return 0, false
	}
	for i := 2; i < len(vals); i++ {
		if vals[i]-vals[i-1] != step {
			return 0, false
		}
	}
	return step, true
}
