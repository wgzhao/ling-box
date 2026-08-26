// Package jsonx provides JSON formatting and validation utilities.
package jsonx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// FormatOptions controls how JSON is reformatted.
type FormatOptions struct {
	Indent   string // indentation string; defaults to two spaces
	SortKeys bool   // sort object keys alphabetically
	Compact  bool   // output as a single line
}

// Format re-indents valid JSON input. Key order is preserved (unlike
// Unmarshal + Marshal, whose map iteration reorders keys), unless
// SortKeys is set. Compact takes precedence over Indent.
func Format(input []byte, opts FormatOptions) ([]byte, error) {
	if !json.Valid(input) {
		return nil, fmt.Errorf("invalid JSON: %s", syntaxError(input))
	}
	indent := opts.Indent
	if indent == "" {
		indent = "  "
	}
	if opts.SortKeys {
		var v any
		if err := json.Unmarshal(input, &v); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		if opts.Compact {
			out, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("JSON marshal failed: %w", err)
			}
			return out, nil
		}
		out, err := json.MarshalIndent(v, "", indent)
		if err != nil {
			return nil, fmt.Errorf("JSON marshal failed: %w", err)
		}
		return out, nil
	}
	if opts.Compact {
		// Compact without SortKeys preserves the original key order.
		var buf bytes.Buffer
		if err := json.Compact(&buf, input); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		return buf.Bytes(), nil
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, input, "", indent); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return buf.Bytes(), nil
}

// Verify reports whether input is syntactically valid JSON, returning an
// error with the first offending line and column otherwise.
func Verify(input []byte) error {
	if json.Valid(input) {
		return nil
	}
	return fmt.Errorf("invalid JSON: %s", syntaxError(input))
}

// syntaxError renders the parse error with line:column coordinates, the
// position json.Valid lacks but Unmarshal exposes via SyntaxError.Offset.
func syntaxError(input []byte) string {
	var v any
	err := json.Unmarshal(input, &v)
	if serr, ok := errors.AsType[*json.SyntaxError](err); ok {
		return fmt.Sprintf("%s at %s", serr.Error(), offsetPos(input, serr.Offset))
	}
	if err != nil {
		return err.Error()
	}
	return "unexpected content"
}

// offsetPos converts a byte offset into line:column coordinates.
func offsetPos(input []byte, offset int64) string {
	if offset < 1 || offset > int64(len(input)) {
		return fmt.Sprintf("offset %d", offset)
	}
	line, col := 1, 1
	for i := int64(0); i < offset; i++ {
		if input[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return fmt.Sprintf("line %d, column %d", line, col)
}
