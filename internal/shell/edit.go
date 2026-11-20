package shell

import (
	"fmt"
	"strings"
)

// span is a half-open range of line indexes covering one managed block,
// markers included.
type span struct {
	start, end int
}

// Apply inserts block into content. When content already has a managed block,
// the first one is replaced in place (its position is preserved) and any
// further duplicate blocks are removed. The changed flag reports whether the
// returned content differs from the input.
func Apply(content, block string) (string, bool, error) {
	lines := splitLines(content)
	spans, err := findBlocks(lines)
	if err != nil {
		return "", false, err
	}
	if len(spans) == 0 {
		out := appendBlock(content, block)
		return out, out != content, nil
	}

	var b strings.Builder
	for i := 0; i < len(lines); {
		if s, ok := spanAt(spans, i); ok {
			if s.start == spans[0].start {
				b.WriteString(block)
			}
			i = s.end
			continue
		}
		b.WriteString(lines[i])
		i++
	}
	out := b.String()
	return out, out != content, nil
}

// Remove deletes every managed block from content.
func Remove(content string) (string, bool, error) {
	lines := splitLines(content)
	spans, err := findBlocks(lines)
	if err != nil {
		return "", false, err
	}
	if len(spans) == 0 {
		return content, false, nil
	}

	var b strings.Builder
	for i := 0; i < len(lines); {
		if s, ok := spanAt(spans, i); ok {
			i = s.end
			continue
		}
		b.WriteString(lines[i])
		i++
	}
	out := b.String()
	return out, out != content, nil
}

func spanAt(spans []span, i int) (span, bool) {
	for _, s := range spans {
		if s.start == i {
			return s, true
		}
	}
	return span{}, false
}

// findBlocks pairs start/end markers. Unpaired markers mean the file was
// hand-edited into an ambiguous state; report it instead of guessing, so a
// user's config is never silently truncated.
func findBlocks(lines []string) ([]span, error) {
	var spans []span
	start := -1
	for i, line := range lines {
		switch strings.TrimSpace(line) {
		case StartMarker:
			if start >= 0 {
				return nil, fmt.Errorf("start marker at line %d appears before the block opened at line %d ends", i+1, start+1)
			}
			start = i
		case EndMarker:
			if start < 0 {
				return nil, fmt.Errorf("end marker at line %d has no matching start marker", i+1)
			}
			spans = append(spans, span{start: start, end: i + 1})
			start = -1
		}
	}
	if start >= 0 {
		return nil, fmt.Errorf("start marker at line %d has no matching end marker", start+1)
	}
	return spans, nil
}

// splitLines splits content into lines, keeping each line's terminator so the
// untouched parts of the file round-trip byte for byte.
func splitLines(content string) []string {
	var lines []string
	for len(content) > 0 {
		i := strings.IndexByte(content, '\n')
		if i < 0 {
			lines = append(lines, content)
			break
		}
		lines = append(lines, content[:i+1])
		content = content[i+1:]
	}
	return lines
}

func appendBlock(content, block string) string {
	if content == "" {
		return block
	}
	nl := newlineOf(content)
	var b strings.Builder
	b.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		b.WriteString(nl)
	}
	if !strings.HasSuffix(content, "\n\n") && !strings.HasSuffix(content, "\r\n\r\n") {
		b.WriteString(nl)
	}
	b.WriteString(block)
	return b.String()
}

// newlineOf returns the line separator the file predominantly uses.
func newlineOf(content string) string {
	crlf := strings.Count(content, "\r\n")
	if crlf > 0 && crlf == strings.Count(content, "\n") {
		return "\r\n"
	}
	return "\n"
}
