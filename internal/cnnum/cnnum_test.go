package cnnum

import "testing"

func TestArabicToLower(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"0", "零"},
		{"1", "一"},
		{"10", "十"},
		{"11", "十一"},
		{"20", "二十"},
		{"99", "九十九"},
		{"100", "一百"},
		{"101", "一百零一"},
		{"110", "一百一十"},
		{"111", "一百一十一"},
		{"1000", "一千"},
		{"1001", "一千零一"},
		{"1010", "一千零一十"},
		{"1100", "一千一百"},
		{"10000", "一万"},
		{"10001", "一万零一"},
		{"10010", "一万零一十"},
		{"11000", "一万一千"},
		{"100000", "十万"},
		{"1000000", "一百万"},
		{"10000000", "一千万"},
		{"100000000", "一亿"},
		{"100000001", "一亿零一"},
		{"100010000", "一亿零一万"},
		{"110000000", "一亿一千万"},
		{"123456789", "一亿二千三百四十五万六千七百八十九"},
		{"100230045", "一亿零二十三万零四十五"},
		{"1000000000", "十亿"},
		{"1000000001", "十亿零一"},
		{"1001000000", "十亿零一百万"},
		{"1000000000000", "一万亿"},
		{"1000000000001", "一万亿零一"},
		{"10000000000000000", "一亿亿"},
		{"123", "一百二十三"},
		{"-123", "负一百二十三"},
		{"12.30", "十二点三零"},
		{"0.5", "零点五"},
		{"-0.05", "负零点零五"},
		{"1,234", "一千二百三十四"},
	}
	for _, tt := range tests {
		got, err := ArabicToChinese(tt.in, Lower)
		if err != nil {
			t.Errorf("ArabicToChinese(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ArabicToChinese(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestArabicToUpper(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"0", "零"},
		{"10", "壹拾"},
		{"123", "壹佰贰拾叁"},
		{"10001", "壹万零壹"},
		{"12.30", "壹拾贰点叁零"},
		{"-123", "负壹佰贰拾叁"},
	}
	for _, tt := range tests {
		got, err := ArabicToChinese(tt.in, Upper)
		if err != nil {
			t.Errorf("ArabicToChinese(%q, upper): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ArabicToChinese(%q, upper) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestChineseToArabic(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"零", "0"},
		{"〇", "0"},
		{"一", "1"},
		{"两", "2"},
		{"十", "10"},
		{"十一", "11"},
		{"一十二", "12"},
		{"一百", "100"},
		{"一百零一", "101"},
		{"一百一十", "110"},
		{"一百十", "110"},
		{"一千零一十", "1010"},
		{"一万", "10000"},
		{"一万零一", "10001"},
		{"一万零一十", "10010"},
		{"十万", "100000"},
		{"一亿", "100000000"},
		{"一亿零一", "100000001"},
		{"一亿零一万", "100010000"},
		{"一万亿", "1000000000000"},
		{"一亿亿", "10000000000000000"},
		{"一百二十三", "123"},
		{"壹佰贰拾叁", "123"},
		{"壹万贰仟叁佰肆拾伍", "12345"},
		{"一百二十三万四千五百六十七", "1234567"},
		{"二零二五", "2025"},
		{"一二三", "123"},
		{"负十二点三零", "-12.30"},
		{"十二点三", "12.3"},
		{"零点零五", "0.05"},
		{"壹佰贰拾叁点肆伍", "123.45"},
		{"负零", "0"},
	}
	for _, tt := range tests {
		got, err := ChineseToArabic(tt.in)
		if err != nil {
			t.Errorf("ChineseToArabic(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ChineseToArabic(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestConvertDirectionAndStyle(t *testing.T) {
	got, err := Convert("123", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Direction != ToChinese || got.Style != Lower || got.Output != "一百二十三" {
		t.Errorf("Convert(123) = %+v", got)
	}

	got, err = Convert("壹佰贰拾叁", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Direction != ToArabic || got.Output != "123" {
		t.Errorf("Convert(壹佰贰拾叁) = %+v", got)
	}

	if _, err := Convert("123", "weird"); err == nil {
		t.Error("invalid style: expected error")
	}
}

func TestErrors(t *testing.T) {
	arabic := []string{"", "abc", "1.2.3", "1e3", "--1", "9223372036854775808"}
	for _, in := range arabic {
		if _, err := ArabicToChinese(in, Lower); err == nil {
			t.Errorf("ArabicToChinese(%q): expected error", in)
		}
	}

	chinese := []string{
		"", "abc", "一百百", "十百", "百", "万", "亿", "零万",
		"一十二三", "三点", "点五", "一.五",
	}
	for _, in := range chinese {
		if _, err := ChineseToArabic(in); err == nil {
			t.Errorf("ChineseToArabic(%q): expected error", in)
		}
	}
}
