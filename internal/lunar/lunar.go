// Package lunar converts between the Gregorian calendar and the Chinese lunar
// calendar, and exposes sexagenary cycle, zodiac, and solar-term information.
package lunar

import (
	"fmt"
	"time"

	"github.com/6tail/lunar-go/calendar"
)

// Supported Gregorian/lunar year range. Outside this range the underlying
// calendar data is ambiguous in practice, so the tool refuses to guess.
const (
	MinYear = 1900
	MaxYear = 2100
)

// DateInfo describes the same day in both calendars.
type DateInfo struct {
	Solar        string `json:"solar"`
	SolarWeekday string `json:"solar_weekday"`

	Lunar      string `json:"lunar"`
	LunarYear  int    `json:"lunar_year"`
	LunarMonth int    `json:"lunar_month"`
	LunarDay   int    `json:"lunar_day"`
	LunarLeap  bool   `json:"lunar_leap"`
	MonthName  string `json:"month_name"`
	DayName    string `json:"day_name"`

	Zodiac      string `json:"zodiac"`
	GanZhiYear  string `json:"ganzhi_year"`
	GanZhiMonth string `json:"ganzhi_month"`
	GanZhiDay   string `json:"ganzhi_day"`

	SolarTerm string `json:"solar_term,omitempty"`
}

// Term is one of the 24 solar terms.
type Term struct {
	Date string `json:"date"`
	Name string `json:"name"`
}

// FromSolar converts a Gregorian date to lunar date information.
func FromSolar(year, month, day int) (info DateInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			info = DateInfo{}
			err = fmt.Errorf("lunar conversion failed: %v", r)
		}
	}()

	if err := validateSolar(year, month, day); err != nil {
		return DateInfo{}, err
	}
	solar := calendar.NewSolarFromYmd(year, month, day)
	return build(solar, solar.GetLunar()), nil
}

// FromLunar converts a lunar date to Gregorian date information. leap selects
// the leap month when the year has one.
func FromLunar(year, month, day int, leap bool) (info DateInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			info = DateInfo{}
			err = fmt.Errorf("lunar conversion failed: %v", r)
		}
	}()

	if err := validateYear(year); err != nil {
		return DateInfo{}, err
	}
	if month < 1 || month > 12 {
		return DateInfo{}, fmt.Errorf("lunar month must be between 1 and 12")
	}
	if day < 1 || day > 30 {
		return DateInfo{}, fmt.Errorf("lunar day must be between 1 and 30")
	}

	lunarYear := calendar.NewLunarYear(year)
	if leap {
		leapMonth := lunarYear.GetLeapMonth()
		switch {
		case leapMonth == 0:
			return DateInfo{}, fmt.Errorf("lunar year %d has no leap month", year)
		case leapMonth != month:
			return DateInfo{}, fmt.Errorf("lunar year %d leap month is %d, not %d", year, leapMonth, month)
		}
	}

	signedMonth := month
	if leap {
		signedMonth = -month
	}
	lunarMonth := lunarYear.GetMonth(signedMonth)
	if lunarMonth == nil {
		return DateInfo{}, fmt.Errorf("lunar year %d has no month %d", year, signedMonth)
	}
	if day > lunarMonth.GetDayCount() {
		return DateInfo{}, fmt.Errorf("lunar %d month %d has only %d days", year, signedMonth, lunarMonth.GetDayCount())
	}

	lunar := calendar.NewLunarFromYmd(year, signedMonth, day)
	if lunar.GetYear() != year || lunar.GetMonth() != signedMonth || lunar.GetDay() != day {
		return DateInfo{}, fmt.Errorf("invalid lunar date %d-%02d-%02d", year, signedMonth, day)
	}
	solar := lunar.GetSolar()
	return build(solar, solar.GetLunar()), nil
}

// Terms returns the 24 solar terms of a Gregorian year in chronological
// order.
func Terms(year int) (terms []Term, err error) {
	defer func() {
		if r := recover(); r != nil {
			terms = nil
			err = fmt.Errorf("solar term calculation failed: %v", r)
		}
	}()

	if err := validateYear(year); err != nil {
		return nil, err
	}
	terms = make([]Term, 0, 24)
	for t := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC); t.Year() == year; t = t.AddDate(0, 0, 1) {
		solar := calendar.NewSolarFromYmd(t.Year(), int(t.Month()), t.Day())
		if name := solar.GetLunar().GetJieQi(); name != "" {
			terms = append(terms, Term{Date: solar.ToYmd(), Name: name})
		}
	}
	return terms, nil
}

func build(solar *calendar.Solar, lunar *calendar.Lunar) DateInfo {
	signedMonth := lunar.GetMonth()
	leap := signedMonth < 0
	month := signedMonth
	if month < 0 {
		month = -month
	}

	return DateInfo{
		Solar:        solar.ToYmd(),
		SolarWeekday: "星期" + solar.GetWeekInChinese(),

		Lunar:      fmt.Sprintf("%d年%s月%s", lunar.GetYear(), lunar.GetMonthInChinese(), lunar.GetDayInChinese()),
		LunarYear:  lunar.GetYear(),
		LunarMonth: month,
		LunarDay:   lunar.GetDay(),
		LunarLeap:  leap,
		MonthName:  lunar.GetMonthInChinese(),
		DayName:    lunar.GetDayInChinese(),

		Zodiac:      lunar.GetYearShengXiao(),
		GanZhiYear:  lunar.GetYearInGanZhi(),
		GanZhiMonth: lunar.GetMonthInGanZhi(),
		GanZhiDay:   lunar.GetDayInGanZhi(),

		SolarTerm: lunar.GetJieQi(),
	}
}

func validateYear(year int) error {
	if year < MinYear || year > MaxYear {
		return fmt.Errorf("year must be between %d and %d", MinYear, MaxYear)
	}
	return nil
}

func validateSolar(year, month, day int) error {
	if err := validateYear(year); err != nil {
		return err
	}
	if month < 1 || month > 12 {
		return fmt.Errorf("month must be between 1 and 12")
	}
	if day < 1 || day > 31 {
		return fmt.Errorf("day must be between 1 and 31")
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return fmt.Errorf("invalid Gregorian date %04d-%02d-%02d", year, month, day)
	}
	return nil
}
