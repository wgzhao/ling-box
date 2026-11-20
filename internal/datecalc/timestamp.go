package datecalc

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TimestampOptions controls timestamp parsing.
type TimestampOptions struct {
	// Location is used for local-time output and for inputs without a zone.
	Location *time.Location
	// ForceMillis interprets a numeric input as milliseconds, even when it
	// looks like seconds.
	ForceMillis bool
}

// Timestamp is a parsed instant in every useful representation.
type Timestamp struct {
	Input     string `json:"input"`
	Unix      int64  `json:"unix"`
	UnixMilli int64  `json:"unix_milli"`
	UTC       string `json:"utc"`
	Local     string `json:"local"`
	Timezone  string `json:"timezone"`
	Weekday   string `json:"weekday"`
}

// ParseTimestamp accepts Unix seconds, Unix milliseconds (auto-detected or
// forced), RFC3339, "YYYY-MM-DD HH:MM:SS", "YYYY-MM-DDTHH:MM:SS", and
// "YYYY-MM-DD".
func ParseTimestamp(input string, opts TimestampOptions) (*Timestamp, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return nil, fmt.Errorf("empty timestamp")
	}

	loc := opts.Location
	if loc == nil {
		loc = time.Local
	}

	var t time.Time
	if isNumeric(s) {
		if strings.Contains(s, ".") {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid timestamp %q", input)
			}
			sec := int64(f)
			nsec := int64((f - float64(sec)) * 1e9)
			t = time.Unix(sec, nsec)
		} else {
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid timestamp %q", input)
			}
			switch {
			case opts.ForceMillis:
				t = time.UnixMilli(n)
			case n >= 100_000_000_000 || n <= -100_000_000_000:
				t = time.UnixMilli(n)
			default:
				t = time.Unix(n, 0)
			}
		}
	} else {
		var err error
		t, err = parseTimestampText(s, loc)
		if err != nil {
			return nil, err
		}
	}

	in := t.In(loc)
	return &Timestamp{
		Input:     input,
		Unix:      t.Unix(),
		UnixMilli: t.UnixMilli(),
		UTC:       t.UTC().Format(time.RFC3339),
		Local:     in.Format(time.RFC3339),
		Timezone:  in.Format("MST -07:00"),
		Weekday:   in.Weekday().String(),
	}, nil
}

func parseTimestampText(s string, loc *time.Location) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}

	layouts := []string{
		"2006-01-02T15:04:05",
		DateTimeFormat,
		DateFormat,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid timestamp %q (use Unix seconds/ms, RFC3339, YYYY-MM-DD[ HH:MM:SS], or YYYY-MM-DD)", s)
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '-' || s[0] == '+' {
		s = s[1:]
	}
	if s == "" {
		return false
	}
	dots := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r == '.':
			dots++
			if dots > 1 {
				return false
			}
		default:
			return false
		}
	}
	return true
}
