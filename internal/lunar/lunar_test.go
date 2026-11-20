package lunar

import "testing"

func TestFromSolar(t *testing.T) {
	info, err := FromSolar(2025, 1, 29)
	if err != nil {
		t.Fatal(err)
	}
	if info.Solar != "2025-01-29" || info.SolarWeekday != "星期三" {
		t.Errorf("solar = %q / %q", info.Solar, info.SolarWeekday)
	}
	if info.Lunar != "2025年正月初一" || info.LunarYear != 2025 || info.LunarMonth != 1 || info.LunarDay != 1 {
		t.Errorf("lunar = %+v", info)
	}
	if info.LunarLeap || info.MonthName != "正" || info.DayName != "初一" {
		t.Errorf("month/day = %+v", info)
	}
	if info.Zodiac != "蛇" || info.GanZhiYear != "乙巳" || info.GanZhiMonth != "丁丑" || info.GanZhiDay != "戊戌" {
		t.Errorf("ganzhi = %+v", info)
	}
	if info.SolarTerm != "" {
		t.Errorf("solar term = %q", info.SolarTerm)
	}

	dragon, err := FromSolar(2024, 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if dragon.Zodiac != "龙" || dragon.GanZhiYear != "甲辰" || dragon.Lunar != "2024年正月初一" {
		t.Errorf("2024 = %+v", dragon)
	}
}

func TestFromSolarLeapMonthAndTerm(t *testing.T) {
	leap, err := FromSolar(2025, 7, 25)
	if err != nil {
		t.Fatal(err)
	}
	if !leap.LunarLeap || leap.LunarMonth != 6 || leap.Lunar != "2025年闰六月初一" {
		t.Errorf("leap = %+v", leap)
	}

	term, err := FromSolar(2025, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if term.SolarTerm != "立春" {
		t.Errorf("solar term = %q", term.SolarTerm)
	}

	next, err := FromSolar(2025, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if next.SolarTerm != "" {
		t.Errorf("Feb 4 term = %q", next.SolarTerm)
	}
}

func TestFromLunar(t *testing.T) {
	tests := []struct {
		year, month, day int
		leap             bool
		wantSolar        string
		wantLunar        string
	}{
		{2025, 1, 1, false, "2025-01-29", "2025年正月初一"},
		{2025, 6, 1, false, "2025-06-25", "2025年六月初一"},
		{2025, 6, 1, true, "2025-07-25", "2025年闰六月初一"},
		{2025, 6, 30, false, "2025-07-24", "2025年六月三十"},
	}
	for _, tt := range tests {
		info, err := FromLunar(tt.year, tt.month, tt.day, tt.leap)
		if err != nil {
			t.Errorf("FromLunar(%d,%d,%d,%v): %v", tt.year, tt.month, tt.day, tt.leap, err)
			continue
		}
		if info.Solar != tt.wantSolar || info.Lunar != tt.wantLunar {
			t.Errorf("FromLunar(%d,%d,%d,%v) = %q / %q, want %q / %q",
				tt.year, tt.month, tt.day, tt.leap, info.Solar, info.Lunar, tt.wantSolar, tt.wantLunar)
		}
	}
}

func TestErrors(t *testing.T) {
	solar := [][3]int{
		{2025, 2, 30}, {2025, 13, 1}, {2025, 0, 1}, {1899, 1, 1}, {2101, 1, 1},
	}
	for _, d := range solar {
		if _, err := FromSolar(d[0], d[1], d[2]); err == nil {
			t.Errorf("FromSolar(%v): expected error", d)
		}
	}

	lunar := []struct {
		y, m, d int
		leap    bool
	}{
		{2025, 1, 1, true}, // 2025 has a leap 6th month, not 1st
		{2025, 13, 1, false},
		{2025, 1, 31, false},
		{2025, 5, 30, false}, // 5th month has 29 days
		{2025, 6, 30, true},  // leap 6th month has 29 days
		{1899, 1, 1, false},
		{2101, 1, 1, false},
	}
	for _, d := range lunar {
		if _, err := FromLunar(d.y, d.m, d.d, d.leap); err == nil {
			t.Errorf("FromLunar(%d,%d,%d,%v): expected error", d.y, d.m, d.d, d.leap)
		}
	}

	if _, err := Terms(1899); err == nil {
		t.Error("Terms(1899): expected error")
	}
}

func TestTerms(t *testing.T) {
	terms, err := Terms(2025)
	if err != nil {
		t.Fatal(err)
	}
	if len(terms) != 24 {
		t.Fatalf("got %d terms, want 24", len(terms))
	}
	if terms[0].Date != "2025-01-05" || terms[0].Name != "小寒" {
		t.Errorf("first term = %+v", terms[0])
	}
	if terms[len(terms)-1].Date != "2025-12-21" || terms[len(terms)-1].Name != "冬至" {
		t.Errorf("last term = %+v", terms[len(terms)-1])
	}
	found := map[string]string{}
	for _, term := range terms {
		found[term.Name] = term.Date
	}
	if found["立春"] != "2025-02-03" {
		t.Errorf("立春 = %q", found["立春"])
	}
	if found["谷雨"] != "2025-04-20" {
		t.Errorf("谷雨 = %q", found["谷雨"])
	}
}
