// Package textx provides text case conversion, escaping, width conversion
// and statistics.
package textx

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/width"
)

// Operations returns the supported transform names.
func Operations() []string {
	return []string{
		"camel", "pascal", "snake", "kebab", "constant", "title",
		"lower", "upper", "slug",
		"html-encode", "html-decode",
		"json-escape", "json-unescape",
		"fullwidth", "halfwidth",
	}
}

// Apply performs one named transform on input.
func Apply(op, input string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "camel":
		return Camel(input), nil
	case "pascal":
		return Pascal(input), nil
	case "snake":
		return Snake(input), nil
	case "kebab":
		return Kebab(input), nil
	case "constant", "const":
		return Constant(input), nil
	case "title":
		return Title(input), nil
	case "lower":
		return strings.ToLower(input), nil
	case "upper":
		return strings.ToUpper(input), nil
	case "slug":
		return Slug(input), nil
	case "html-encode":
		return HTMLEncode(input), nil
	case "html-decode":
		return HTMLDecode(input), nil
	case "json-escape":
		return JSONEscape(input)
	case "json-unescape":
		return JSONUnescape(input)
	case "fullwidth", "width-full":
		return Fullwidth(input), nil
	case "halfwidth", "width-half":
		return Halfwidth(input), nil
	default:
		return "", fmt.Errorf("unknown operation %q (supported: %s, stats)", op, strings.Join(Operations(), ", "))
	}
}

// Camel converts "Hello World" to "helloWorld".
func Camel(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(strings.ToLower(words[0]))
	for _, w := range words[1:] {
		b.WriteString(capitalize(w))
	}
	return b.String()
}

// Pascal converts "hello world" to "HelloWorld".
func Pascal(s string) string {
	var b strings.Builder
	for _, w := range splitWords(s) {
		b.WriteString(capitalize(w))
	}
	return b.String()
}

// Snake converts "Hello World" to "hello_world".
func Snake(s string) string {
	return joinLower(s, "_")
}

// Kebab converts "Hello World" to "hello-world".
func Kebab(s string) string {
	return joinLower(s, "-")
}

// Constant converts "Hello World" to "HELLO_WORLD".
func Constant(s string) string {
	return strings.ToUpper(joinLower(s, "_"))
}

// Title converts "hello world" to "Hello World".
func Title(s string) string {
	words := splitWords(s)
	for i, w := range words {
		words[i] = capitalize(w)
	}
	return strings.Join(words, " ")
}

// Slug converts "Hello, World!" to "hello-world".
func Slug(s string) string {
	return joinLower(s, "-")
}

// HTMLEncode escapes HTML special characters.
func HTMLEncode(s string) string { return html.EscapeString(s) }

// HTMLDecode unescapes HTML entities, including named entities.
func HTMLDecode(s string) string { return html.UnescapeString(s) }

// JSONEscape escapes s for use inside a JSON string (without the quotes).
func JSONEscape(s string) (string, error) {
	q := strconv.Quote(s)
	if len(q) < 2 {
		return "", fmt.Errorf("cannot escape input")
	}
	return q[1 : len(q)-1], nil
}

// JSONUnescape decodes JSON string escapes, with or without surrounding
// quotes.
func JSONUnescape(s string) (string, error) {
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		out, err := strconv.Unquote(s)
		if err != nil {
			return "", fmt.Errorf("invalid JSON string: %w", err)
		}
		return out, nil
	}
	out, err := strconv.Unquote(`"` + s + `"`)
	if err != nil {
		return "", fmt.Errorf("invalid JSON escape: %w", err)
	}
	return out, nil
}

// Fullwidth converts ASCII to full-width forms.
func Fullwidth(s string) string { return width.Widen.String(s) }

// Halfwidth converts full-width forms to ASCII.
func Halfwidth(s string) string { return width.Narrow.String(s) }

// Stats summarizes a piece of text.
type Stats struct {
	Characters     int `json:"characters"`
	Bytes          int `json:"bytes"`
	Words          int `json:"words"`
	Lines          int `json:"lines"`
	CJK            int `json:"cjk"`
	ReadingMinutes int `json:"reading_minutes"`
}

// Analyze counts characters, bytes, words, lines, CJK characters, and
// estimates reading time (200 words or 300 CJK characters per minute).
func Analyze(s string) Stats {
	st := Stats{
		Characters: len([]rune(s)),
		Bytes:      len(s),
		Words:      len(strings.Fields(s)),
	}
	if s != "" {
		st.Lines = 1 + strings.Count(s, "\n")
	}
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			st.CJK++
		}
	}

	minutes := st.Words / 200
	if cjkMinutes := st.CJK / 300; cjkMinutes > minutes {
		minutes = cjkMinutes
	}
	if s != "" && minutes == 0 {
		minutes = 1
	}
	st.ReadingMinutes = minutes
	return st
}

func joinLower(s, sep string) string {
	words := splitWords(s)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, sep)
}

func capitalize(w string) string {
	r := []rune(strings.ToLower(w))
	if len(r) == 0 {
		return ""
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// splitWords breaks camelCase, PascalCase, snake_case, kebab-case and plain
// prose into words. Acronyms followed by a capitalized word split correctly
// ("HTTPServer" -> "HTTP", "Server").
func splitWords(s string) []string {
	runes := []rune(s)
	var (
		words []string
		cur   []rune
	)
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = nil
		}
	}

	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if len(cur) > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				flush()
			} else if unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
				flush()
			}
		}
		cur = append(cur, r)
	}
	flush()
	return words
}
