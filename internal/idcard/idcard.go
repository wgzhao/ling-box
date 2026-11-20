// Package idcard validates Chinese resident identity card numbers and
// generates synthetic test numbers.
package idcard

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
	"unicode"
)

// Region is a county-level administrative division.
type Region struct {
	Province string
	City     string
	Area     string
}

// Name returns the readable full name, skipping placeholder city names used
// by municipalities and directly administered counties.
func (r Region) Name() string {
	if r.City == "" || r.City == "市辖区" || r.City == "县" ||
		strings.Contains(r.City, "直辖县级") {
		return r.Province + r.Area
	}
	return r.Province + r.City + r.Area
}

// LookupRegion returns the division for a 6-digit county-level code.
func LookupRegion(code string) (Region, bool) {
	r, ok := regions[code]
	return r, ok
}

// DefaultRegion is Beijing Chaoyang, used when no region is requested.
const DefaultRegion = "110105"

var defaultBirth = time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)

var weights = [17]int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}

const checkCodes = "10X98765432"

// Options controls validation behavior.
type Options struct {
	// StrictRegion treats an unknown region code as invalid instead of a
	// warning. Useful for new IDs, while old IDs may use historical codes.
	StrictRegion bool
	// Now is the reference time for future-birth checks; zero means time.Now.
	Now time.Time
}

// Info is the result of validating an identity card number.
type Info struct {
	Input  string `json:"input"`
	Valid  bool   `json:"valid"`
	Format string `json:"format,omitempty"`
	Number string `json:"number,omitempty"`

	RegionCode  string `json:"region_code,omitempty"`
	Province    string `json:"province,omitempty"`
	City        string `json:"city,omitempty"`
	Area        string `json:"area,omitempty"`
	RegionName  string `json:"region_name,omitempty"`
	RegionKnown bool   `json:"region_known"`

	BirthDate string `json:"birth_date,omitempty"`
	Gender    string `json:"gender,omitempty"`
	Sequence  string `json:"sequence,omitempty"`

	CheckDigit         string `json:"check_digit,omitempty"`
	ExpectedCheckDigit string `json:"expected_check_digit,omitempty"`

	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// Validate checks a 15- or 18-digit identity card number. Invalid numbers are
// returned as a result with Valid=false and one or more Errors; a Go error is
// reserved for programming mistakes, so the returned error is always nil.
func Validate(input string, opts Options) Info {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	info := Info{Input: input}
	s := normalize(input)
	if s == "" {
		info.Errors = append(info.Errors, "ID number is empty")
		return info
	}

	switch len(s) {
	case 18:
		validate18(&info, s, opts, now)
	case 15:
		validate15(&info, s, opts, now)
	default:
		info.Errors = append(info.Errors, "ID number must have 15 or 18 digits")
	}
	info.Valid = len(info.Errors) == 0
	return info
}

func validate18(info *Info, s string, opts Options, now time.Time) {
	info.Format = "18-digit"
	info.Number = s
	info.RegionCode = s[:6]
	info.Sequence = s[14:17]
	info.CheckDigit = strings.ToUpper(s[17:18])

	first17 := s[:17]
	if !allDigits(first17) {
		info.Errors = append(info.Errors, "ID number must contain 17 digits followed by a check digit")
		return
	}
	if !allDigits(s[17:18]) && s[17] != 'X' && s[17] != 'x' {
		info.Errors = append(info.Errors, "check digit must be a digit or X")
	}

	applyRegion(info, s[:6], opts)
	applyBirth(info, s[6:14], now)
	info.Gender = genderFromDigit(s[16])

	expected, err := checkDigit(first17)
	if err != nil {
		info.Errors = append(info.Errors, err.Error())
		return
	}
	info.ExpectedCheckDigit = string(expected)
	if info.CheckDigit != "" && info.CheckDigit != string(expected) {
		info.Errors = append(info.Errors, fmt.Sprintf("checksum mismatch: expected %c, got %s", expected, info.CheckDigit))
	}
}

func validate15(info *Info, s string, opts Options, now time.Time) {
	info.Format = "15-digit"
	info.RegionCode = s[:6]
	info.Sequence = s[12:15]
	info.Gender = genderFromDigit(s[14])

	if !allDigits(s) {
		info.Errors = append(info.Errors, "15-digit ID number must contain only digits")
		return
	}

	birthYYMMDD := s[6:12]
	applyBirth(info, "19"+birthYYMMDD, now)

	first17 := s[:6] + "19" + birthYYMMDD + s[12:15]
	expected, err := checkDigit(first17)
	if err != nil {
		info.Errors = append(info.Errors, err.Error())
		return
	}
	info.Number = first17 + string(expected)
	info.CheckDigit = string(expected)
	info.ExpectedCheckDigit = string(expected)
	applyRegion(info, s[:6], opts)
}

func applyRegion(info *Info, code string, opts Options) {
	region, ok := LookupRegion(code)
	info.RegionKnown = ok
	if !ok {
		msg := fmt.Sprintf("unknown region code %s (may be historical or invalid)", code)
		if opts.StrictRegion {
			info.Errors = append(info.Errors, msg)
		} else {
			info.Warnings = append(info.Warnings, msg)
		}
		return
	}
	info.Province = region.Province
	info.City = region.City
	info.Area = region.Area
	info.RegionName = region.Name()
}

func applyBirth(info *Info, yyyymmdd string, now time.Time) {
	t, err := time.Parse("20060102", yyyymmdd)
	if err != nil {
		info.Errors = append(info.Errors, fmt.Sprintf("invalid birth date %s", yyyymmdd))
		return
	}
	info.BirthDate = t.Format("2006-01-02")
	if t.Year() < 1900 {
		info.Errors = append(info.Errors, "birth year must be 1900 or later")
	}
	if t.After(now) {
		info.Errors = append(info.Errors, fmt.Sprintf("birth date %s is in the future", info.BirthDate))
	}
}

func checkDigit(first17 string) (byte, error) {
	if len(first17) != 17 {
		return 0, fmt.Errorf("internal error: expected 17 digits")
	}
	sum := 0
	for i := 0; i < 17; i++ {
		if first17[i] < '0' || first17[i] > '9' {
			return 0, fmt.Errorf("ID number contains a non-digit")
		}
		sum += int(first17[i]-'0') * weights[i]
	}
	return checkCodes[sum%11], nil
}

func genderFromDigit(b byte) string {
	if (b-'0')%2 == 1 {
		return "male"
	}
	return "female"
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

func normalize(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
	return strings.ToUpper(s)
}

// GenerateOptions controls synthetic ID generation.
type GenerateOptions struct {
	Region string
	Birth  time.Time
	Gender string // male, female, or any
	Count  int
	Now    time.Time
}

// Generated is one synthetic identity card number.
type Generated struct {
	Number     string `json:"number"`
	RegionCode string `json:"region_code"`
	RegionName string `json:"region_name"`
	BirthDate  string `json:"birth_date"`
	Gender     string `json:"gender"`
	Sequence   string `json:"sequence"`
}

// Generate creates identity card numbers with valid checksums for testing.
// The numbers are synthetic but may coincidentally match real people; they
// must not be used as real identities.
func Generate(opts GenerateOptions) ([]Generated, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	count := opts.Count
	if count == 0 {
		count = 1
	}
	if count < 0 || count > 1000 {
		return nil, fmt.Errorf("count must be between 1 and 1000")
	}

	regionCode := opts.Region
	if regionCode == "" {
		regionCode = DefaultRegion
	}
	region, ok := LookupRegion(regionCode)
	if !ok {
		return nil, fmt.Errorf("unknown region code %q", regionCode)
	}

	birth := opts.Birth
	if birth.IsZero() {
		birth = defaultBirth
	}
	if birth.Year() < 1900 {
		return nil, fmt.Errorf("birth year must be 1900 or later")
	}
	if birth.After(now) {
		return nil, fmt.Errorf("birth date %s is in the future", birth.Format("2006-01-02"))
	}

	gender, err := normalizeGender(opts.Gender)
	if err != nil {
		return nil, err
	}

	out := make([]Generated, 0, count)
	for i := 0; i < count; i++ {
		g := gender
		if g == "any" {
			if rand.IntN(2) == 0 {
				g = "male"
			} else {
				g = "female"
			}
		}

		seq := 100 + rand.IntN(900)
		if g == "male" {
			if seq%2 == 0 {
				seq++
			}
		} else if seq%2 == 1 {
			seq--
		}

		seqStr := fmt.Sprintf("%03d", seq)
		first17 := regionCode + birth.Format("20060102") + seqStr
		check, err := checkDigit(first17)
		if err != nil {
			return nil, err
		}
		out = append(out, Generated{
			Number:     first17 + string(check),
			RegionCode: regionCode,
			RegionName: region.Name(),
			BirthDate:  birth.Format("2006-01-02"),
			Gender:     g,
			Sequence:   seqStr,
		})
	}
	return out, nil
}

func normalizeGender(g string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(g)) {
	case "", "any":
		return "any", nil
	case "male", "m", "男":
		return "male", nil
	case "female", "f", "女":
		return "female", nil
	default:
		return "", fmt.Errorf("unknown gender %q (supported: male, female, any)", g)
	}
}
