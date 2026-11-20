// Package cnnum converts between Arabic numerals and Chinese numerals in the
// common lowercase (一百二十三) and financial uppercase (壹佰贰拾叁) forms.
package cnnum

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// Style selects the Chinese numeral character set.
type Style string

// Supported styles.
const (
	Lower Style = "lower"
	Upper Style = "upper"
)

// Direction reports the conversion performed by Convert.
type Direction string

// Supported directions.
const (
	ToChinese Direction = "to_chinese"
	ToArabic  Direction = "to_arabic"
)

// Result is the outcome of an auto-detected conversion.
type Result struct {
	Input     string    `json:"input"`
	Output    string    `json:"output"`
	Direction Direction `json:"direction"`
	Style     Style     `json:"style,omitempty"`
}

var (
	lowerDigits = []rune("零一二三四五六七八九")
	upperDigits = []rune("零壹贰叁肆伍陆柒捌玖")
)

// Convert detects the input direction, formats Arabic input with style, and
// parses Chinese input (either character set) back to Arabic digits.
func Convert(input string, style Style) (Result, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return Result{}, fmt.Errorf("empty input")
	}

	if isArabicInput(s) {
		st, err := normalizeStyle(style)
		if err != nil {
			return Result{}, err
		}
		out, err := ArabicToChinese(s, st)
		if err != nil {
			return Result{}, err
		}
		return Result{Input: input, Output: out, Direction: ToChinese, Style: st}, nil
	}

	if containsChineseNumeral(s) {
		out, err := ChineseToArabic(s)
		if err != nil {
			return Result{}, err
		}
		return Result{Input: input, Output: out, Direction: ToArabic}, nil
	}

	return Result{}, fmt.Errorf("cannot detect input: expected an Arabic number or a Chinese numeral")
}

func normalizeStyle(style Style) (Style, error) {
	switch Style(strings.ToLower(string(style))) {
	case "":
		return Lower, nil
	case Lower:
		return Lower, nil
	case Upper:
		return Upper, nil
	default:
		return "", fmt.Errorf("unknown style %q (supported: lower, upper)", style)
	}
}

// ArabicToChinese converts an Arabic numeral string (integer or decimal) to
// Chinese characters.
func ArabicToChinese(input string, style Style) (string, error) {
	st, err := normalizeStyle(style)
	if err != nil {
		return "", err
	}

	s := clean(input)
	if s == "" {
		return "", fmt.Errorf("empty number")
	}

	neg := false
	for _, sign := range []string{"-", "−", "－", "+"} {
		if strings.HasPrefix(s, sign) {
			neg = sign != "+"
			s = strings.TrimPrefix(s, sign)
			break
		}
	}
	s = strings.ReplaceAll(s, "．", ".")
	if s == "" {
		return "", fmt.Errorf("missing digits")
	}

	intPart, fracPart, hasFrac := strings.Cut(s, ".")
	if hasFrac && strings.Contains(fracPart, ".") {
		return "", fmt.Errorf("invalid number %q", input)
	}
	if intPart == "" {
		return "", fmt.Errorf("missing integer part in %q", input)
	}
	if hasFrac && fracPart == "" {
		return "", fmt.Errorf("missing decimal digits in %q", input)
	}
	if !allASCIIDigits(intPart) || (hasFrac && !allASCIIDigits(fracPart)) {
		return "", fmt.Errorf("invalid number %q", input)
	}

	intPart = strings.TrimLeft(intPart, "0")
	if intPart == "" {
		intPart = "0"
	}
	n, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return "", fmt.Errorf("number out of range for int64: %s", intPart)
	}

	var b strings.Builder
	out := formatInteger(n, st)
	b.WriteString(out)
	if hasFrac {
		b.WriteRune('点')
		digits := digitRunes(st)
		for _, r := range fracPart {
			b.WriteRune(digits[r-'0'])
		}
	}

	result := b.String()
	if neg && !isZero(result, st) {
		result = "负" + result
	}
	return result, nil
}

// ChineseToArabic converts a Chinese numeral (lowercase, uppercase, or mixed)
// to an Arabic decimal string.
func ChineseToArabic(input string) (string, error) {
	s := clean(input)
	if s == "" {
		return "", fmt.Errorf("empty number")
	}

	neg := false
	if strings.HasPrefix(s, "负") {
		neg = true
		s = strings.TrimPrefix(s, "负")
	} else if strings.HasPrefix(s, "負") {
		neg = true
		s = strings.TrimPrefix(s, "負")
	}
	if s == "" {
		return "", fmt.Errorf("missing digits")
	}

	intPart := s
	fracPart := ""
	if i := strings.IndexAny(s, "点點"); i >= 0 {
		intPart = s[:i]
		fracPart = s[i+len("点"):]
		if strings.ContainsAny(fracPart, "点點") {
			return "", fmt.Errorf("multiple decimal points")
		}
		if fracPart == "" {
			return "", fmt.Errorf("missing decimal digits")
		}
	}
	if intPart == "" {
		return "", fmt.Errorf("missing integer part")
	}

	var intOut string
	if allChineseDigits(intPart) {
		digits, err := chineseDigitsToASCII(intPart)
		if err != nil {
			return "", err
		}
		digits = strings.TrimLeft(digits, "0")
		if digits == "" {
			digits = "0"
		}
		intOut = digits
	} else {
		n, err := parseChineseInteger(intPart)
		if err != nil {
			return "", err
		}
		intOut = strconv.FormatInt(n, 10)
	}

	out := intOut
	if fracPart != "" {
		digits, err := chineseDigitsToASCII(fracPart)
		if err != nil {
			return "", err
		}
		out += "." + digits
	}

	if neg && !isZeroDecimal(out) {
		out = "-" + out
	}
	return out, nil
}

func formatInteger(n int64, style Style) string {
	if n == 0 {
		return "零"
	}

	units := []string{"", "万", "亿", "万亿", "亿亿"}
	var groups [5]int64
	g := n
	for i := 0; i < len(groups) && g > 0; i++ {
		groups[i] = g % 10000
		g /= 10000
	}

	var parts []string
	needZero := false
	for i := len(groups) - 1; i >= 0; i-- {
		v := groups[i]
		if v == 0 {
			if len(parts) > 0 {
				needZero = true
			}
			continue
		}
		switch {
		case needZero:
			parts = append(parts, "零")
			needZero = false
		case len(parts) > 0 && v < 1000:
			parts = append(parts, "零")
		}
		parts = append(parts, readGroup(v, style)+units[i])
	}

	out := strings.Join(parts, "")
	if style == Lower && strings.HasPrefix(out, "一十") {
		out = "十" + out[len("一十"):]
	}
	return out
}

func readGroup(n int64, style Style) string {
	digits := [4]int{
		int(n / 1000),
		int(n/100) % 10,
		int(n/10) % 10,
		int(n % 10),
	}
	unitChars := []string{"千", "百", "十", ""}
	if style == Upper {
		unitChars = []string{"仟", "佰", "拾", ""}
	}
	digitChars := digitRunes(style)

	var b strings.Builder
	started := false
	zeroPending := false
	for i, d := range digits {
		if d == 0 {
			if started {
				for _, rest := range digits[i+1:] {
					if rest != 0 {
						zeroPending = true
						break
					}
				}
			}
			continue
		}
		if zeroPending {
			b.WriteRune('零')
			zeroPending = false
		}
		b.WriteRune(digitChars[d])
		b.WriteString(unitChars[i])
		started = true
	}
	return b.String()
}

func digitRunes(style Style) []rune {
	if style == Upper {
		return upperDigits
	}
	return lowerDigits
}

func safeAdd(a, b int64) (int64, error) {
	if b > 0 && a > math.MaxInt64-b {
		return 0, fmt.Errorf("number out of range for int64")
	}
	return a + b, nil
}

func safeMul(a, b int64) (int64, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	if a > math.MaxInt64/b {
		return 0, fmt.Errorf("number out of range for int64")
	}
	return a * b, nil
}

func parseChineseInteger(s string) (int64, error) {
	var total, section, number int64
	saw := false

	applyUnit := func(unit int64) error {
		if number == 0 {
			if unit != 10 {
				return fmt.Errorf("unit must follow a digit")
			}
			number = 1
		}
		v, err := safeMul(number, unit)
		if err != nil {
			return err
		}
		section, err = safeAdd(section, v)
		number = 0
		return err
	}

	for _, r := range s {
		if d, ok := digitValue(r); ok {
			if number != 0 {
				return 0, fmt.Errorf("missing unit between digits")
			}
			number = int64(d)
			saw = true
			continue
		}

		switch r {
		case '零', '〇':
			if number != 0 {
				return 0, fmt.Errorf("unexpected zero after a digit")
			}
		case '十', '拾':
			if err := applyUnit(10); err != nil {
				return 0, err
			}
		case '百', '佰':
			if err := applyUnit(100); err != nil {
				return 0, err
			}
		case '千', '仟':
			if err := applyUnit(1000); err != nil {
				return 0, err
			}
		case '万', '萬':
			if section == 0 && number == 0 {
				return 0, fmt.Errorf("万 must follow a value")
			}
			v, err := safeAdd(section, number)
			if err != nil {
				return 0, err
			}
			section, err = safeMul(v, 10000)
			if err != nil {
				return 0, err
			}
			number = 0
		case '亿', '億':
			if total == 0 && section == 0 && number == 0 {
				return 0, fmt.Errorf("亿 must follow a value")
			}
			v, err := safeAdd(total, section)
			if err != nil {
				return 0, err
			}
			v, err = safeAdd(v, number)
			if err != nil {
				return 0, err
			}
			total, err = safeMul(v, 100000000)
			if err != nil {
				return 0, err
			}
			section = 0
			number = 0
		default:
			return 0, fmt.Errorf("unexpected character %q", r)
		}
	}

	if !saw && total == 0 && section == 0 {
		return 0, fmt.Errorf("no numeric value")
	}
	var err error
	total, err = safeAdd(total, section)
	if err != nil {
		return 0, err
	}
	return safeAdd(total, number)
}

func digitValue(r rune) (int, bool) {
	switch r {
	case '零', '〇':
		return 0, true
	case '一', '壹':
		return 1, true
	case '二', '贰', '貳', '两', '兩':
		return 2, true
	case '三', '叁', '參':
		return 3, true
	case '四', '肆':
		return 4, true
	case '五', '伍':
		return 5, true
	case '六', '陆', '陸':
		return 6, true
	case '七', '柒':
		return 7, true
	case '八', '捌':
		return 8, true
	case '九', '玖':
		return 9, true
	}
	return 0, false
}

func allChineseDigits(s string) bool {
	for _, r := range s {
		if _, ok := digitValue(r); !ok {
			return false
		}
	}
	return s != ""
}

func chineseDigitsToASCII(s string) (string, error) {
	var b strings.Builder
	for _, r := range s {
		d, ok := digitValue(r)
		if !ok {
			return "", fmt.Errorf("unexpected character %q in decimal digits", r)
		}
		b.WriteByte(byte('0' + d))
	}
	return b.String(), nil
}

func allASCIIDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

func isArabicInput(s string) bool {
	hasDigit := false
	dot := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '-' || r == '+' || r == '−' || r == '－':
		case r == '.' || r == '．':
			if dot {
				return false
			}
			dot = true
		case r == ',' || r == '，' || unicode.IsSpace(r):
		default:
			return false
		}
	}
	return hasDigit
}

func containsChineseNumeral(s string) bool {
	for _, r := range s {
		if _, ok := digitValue(r); ok {
			return true
		}
		switch r {
		case '十', '拾', '百', '佰', '千', '仟', '万', '萬', '亿', '億',
			'负', '負', '点', '點':
			return true
		}
	}
	return false
}

func clean(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r), r == ',', r == '，':
			return -1
		default:
			return r
		}
	}, strings.TrimSpace(s))
}

func isZero(s string, style Style) bool {
	digits := digitRunes(style)
	for _, r := range s {
		switch r {
		case digits[0], '点':
		default:
			return false
		}
	}
	return true
}

func isZeroDecimal(s string) bool {
	for _, r := range s {
		if r != '0' && r != '.' {
			return false
		}
	}
	return true
}
