package convert

import (
	"bytes"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// ToMarkdown converts JSON or YAML input into a Markdown document:
// an array of objects becomes a table, an array of scalars a bullet
// list, and a top-level object a two-column key/value table. Nested
// objects and arrays inside cells are encoded as compact JSON, matching
// CSV output.
func ToMarkdown(input []byte) ([]byte, error) {
	data, err := parseAny(input)
	if err != nil {
		return nil, err
	}
	return Render("markdown", normalize(data))
}

// renderMarkdown renders an already-parsed value as Markdown. See
// ToMarkdown for the shape rules.
func renderMarkdown(data interface{}) ([]byte, error) {
	switch v := data.(type) {
	case []interface{}:
		if len(v) == 0 {
			return nil, fmt.Errorf("cannot convert to Markdown: array is empty")
		}
		objects, scalars := false, false
		for i, el := range v {
			switch {
			case isObject(el):
				objects = true
			case isScalar(el):
				scalars = true
			default:
				return nil, fmt.Errorf("cannot convert to Markdown: element %d is %s; elements must be objects or scalar values", i, kindOf(el))
			}
		}
		if objects && scalars {
			return nil, fmt.Errorf("cannot convert to Markdown: array mixes objects and scalar values")
		}
		if objects {
			return markdownTable(v)
		}
		return markdownList(v), nil
	case map[string]interface{}:
		return markdownKVTable(v), nil
	default:
		return nil, fmt.Errorf("cannot convert to Markdown: unsupported top-level value of kind %s", kindOf(data))
	}
}

// markdownTable renders an array of objects as a Markdown table with a
// header row, a separator row, and one data row per element.
func markdownTable(arr []interface{}) ([]byte, error) {
	header := unionKeys(arr)
	if len(header) == 0 {
		return nil, fmt.Errorf("cannot convert to Markdown: object elements have no fields")
	}
	var buf bytes.Buffer
	buf.WriteString(mdRow(header))
	buf.WriteString(mdSeparator(len(header)))
	for _, el := range arr {
		m := el.(map[string]interface{})
		row := make([]string, len(header))
		for i, k := range header {
			row[i] = mdCell(cellString(m[k]))
		}
		buf.WriteString(mdRow(row))
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// markdownList renders an array of scalars as a bullet list.
func markdownList(arr []interface{}) []byte {
	var buf bytes.Buffer
	for _, el := range arr {
		fmt.Fprintf(&buf, "- %s\n", mdCell(cellString(el)))
	}
	return bytes.TrimRight(buf.Bytes(), "\n")
}

// markdownKVTable renders a top-level object as a two-column key/value
// table with keys sorted.
func markdownKVTable(m map[string]interface{}) []byte {
	keys := slices.Sorted(maps.Keys(m))
	var buf bytes.Buffer
	buf.WriteString(mdRow([]string{"key", "value"}))
	buf.WriteString(mdSeparator(2))
	for _, k := range keys {
		buf.WriteString(mdRow([]string{mdCell(k), mdCell(cellString(m[k]))}))
	}
	return bytes.TrimRight(buf.Bytes(), "\n")
}

// mdRow renders one table row: | a | b |.
func mdRow(cells []string) string {
	return "| " + strings.Join(cells, " | ") + " |\n"
}

// mdSeparator renders the header separator row: | --- | --- |.
func mdSeparator(n int) string {
	return "| " + strings.TrimRight(strings.Repeat("--- | ", n), " ") + "\n"
}

// mdCell escapes Markdown table syntax in a cell: pipes become \| and
// literal newlines become <br>.
func mdCell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	return strings.ReplaceAll(s, "\n", "<br>")
}
