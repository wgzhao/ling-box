package convert

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// ToCSV converts JSON or YAML input whose top-level value is an array
// into CSV. A top-level object is rejected: CSV needs rows. Nested
// objects and arrays inside object fields are stringified as compact
// JSON. Fields missing from an object are emitted as empty cells, as
// are explicit nulls. Input must be JSON or YAML; CSV is only an
// output format.
func ToCSV(input []byte) ([]byte, error) {
	data, err := parseAny(input)
	if err != nil {
		return nil, err
	}
	return Render("csv", normalize(data))
}

// renderCSV renders an already-parsed value as CSV. See ToCSV for the
// shape rules.
func renderCSV(data interface{}) ([]byte, error) {
	arr, ok := data.([]interface{})
	if !ok {
		if kindOf(data) == "object" {
			return nil, fmt.Errorf("cannot convert to CSV: top-level value is an object; only arrays are supported (use json format to inspect the structure)")
		}
		return nil, fmt.Errorf("cannot convert to CSV: top-level value must be an array, got %s", kindOf(data))
	}
	if len(arr) == 0 {
		return nil, fmt.Errorf("cannot convert to CSV: array is empty")
	}

	// Elements must be all objects or all scalars; mixed (or nested
	// array) elements are rejected rather than silently coerced.
	objects, scalars := false, false
	for i, el := range arr {
		switch {
		case isObject(el):
			objects = true
		case isScalar(el):
			scalars = true
		default:
			return nil, fmt.Errorf("cannot convert to CSV: element %d is %s; elements must be objects or scalar values", i, kindOf(el))
		}
	}
	if objects && scalars {
		return nil, fmt.Errorf("cannot convert to CSV: array mixes objects and scalar values")
	}

	var rows [][]string
	if objects {
		header := unionKeys(arr)
		if len(header) == 0 {
			return nil, fmt.Errorf("cannot convert to CSV: object elements have no fields")
		}
		rows = append(rows, header)
		for _, el := range arr {
			m := el.(map[string]interface{})
			row := make([]string, len(header))
			for i, k := range header {
				row[i] = cellString(m[k])
			}
			rows = append(rows, row)
		}
	} else {
		for _, el := range arr {
			rows = append(rows, []string{cellString(el)})
		}
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	for _, r := range rows {
		if err := w.Write(r); err != nil {
			return nil, fmt.Errorf("CSV write failed: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("CSV write failed: %w", err)
	}
	// The output keeps its full line structure: a trailing empty row
	// (a null element) is a bare newline that trimming would eat. The
	// caller prints it with fmt.Print, which adds no extra newline.
	return buf.Bytes(), nil
}

// parseCSV parses CSV input as an array of objects, one per data row.
// The first row is the header; column names must be unique and every
// row must have the same number of fields (enforced by csv.Reader).
// Cells stay strings — no type inference.
func parseCSV(input []byte) ([]interface{}, error) {
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 {
		return nil, errors.New("empty input")
	}
	r := csv.NewReader(bytes.NewReader(trimmed))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, errors.New("empty input")
	}
	header := records[0]
	seen := make(map[string]bool, len(header))
	for _, h := range header {
		if seen[h] {
			return nil, fmt.Errorf("invalid CSV: duplicate header column %q", h)
		}
		seen[h] = true
	}
	rows := make([]interface{}, 0, len(records)-1)
	for _, rec := range records[1:] {
		// Values stay strings (no type inference); the map type matches
		// normalized JSON/YAML parsing so renderers can share code.
		m := make(map[string]interface{}, len(header))
		for i, h := range header {
			var v string
			if i < len(rec) {
				v = rec[i]
			}
			m[h] = v
		}
		rows = append(rows, m)
	}
	return rows, nil
}

// parseAny parses input as JSON first, then YAML, mirroring DetectFormat.
// yaml.v3 accepts empty documents, so blank input is rejected up front.
func parseAny(input []byte) (interface{}, error) {
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 {
		return nil, errors.New("empty input")
	}
	var data interface{}
	if err := json.Unmarshal(trimmed, &data); err == nil {
		return data, nil
	}
	if err := yaml.Unmarshal(trimmed, &data); err == nil {
		return data, nil
	}
	return nil, fmt.Errorf("unable to detect format: not valid JSON or YAML")
}

// DetectInputFormat detects the input format of stdin data: JSON,
// YAML, then CSV. A YAML document that decodes to a plain string is
// usually CSV-shaped text, so CSV is probed before committing to YAML.
func DetectInputFormat(input []byte) (string, error) {
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 {
		return "", fmt.Errorf("empty input")
	}
	var v interface{}
	if err := json.Unmarshal(trimmed, &v); err == nil {
		return "json", nil
	}
	if err := yaml.Unmarshal(trimmed, &v); err == nil {
		if _, ok := v.(string); !ok {
			return "yaml", nil
		}
	}
	if looksLikeCSV(trimmed) {
		return "csv", nil
	}
	// Plain YAML scalar string (e.g. "hello world"), or nothing matched.
	if err := yaml.Unmarshal(trimmed, &v); err == nil {
		return "yaml", nil
	}
	return "", fmt.Errorf("unable to detect format: not valid JSON, YAML, or CSV")
}

// looksLikeCSV reports whether input is plausibly a CSV table: at least
// two rows, more than one field per row, and every row with the same
// field count (a single column is indistinguishable from YAML text).
func looksLikeCSV(input []byte) bool {
	r := csv.NewReader(bytes.NewReader(input))
	records, err := r.ReadAll()
	if err != nil {
		return false
	}
	if len(records) < 2 || len(records[0]) < 2 {
		return false
	}
	n := len(records[0])
	for _, rec := range records[1:] {
		if len(rec) != n {
			return false
		}
	}
	return true
}

// kindOf names a decoded value's shape for error messages.
func kindOf(v interface{}) string {
	switch v.(type) {
	case nil:
		return "null"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64, int, int64, uint64:
		return "number"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func isObject(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}

// isScalar reports whether v is a flat CSV cell value. nil is scalar:
// it renders as an empty cell.
func isScalar(v interface{}) bool {
	switch v.(type) {
	case nil, string, bool, float64, int, int64, uint64:
		return true
	}
	return false
}

// cellString renders a decoded value as a CSV cell. Nested containers
// and YAML-only types such as time.Time are encoded as compact JSON;
// nil renders as an empty cell.
func cellString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case float64:
		// 'f' avoids scientific notation (1e6 → "1000000", not "1e+06");
		// JSON decodes all numbers as float64.
		return strconv.FormatFloat(t, 'f', -1, 64)
	case time.Time:
		// yaml.v3 resolves unquoted dates to time.Time; emit bare
		// RFC3339, matching YAMLToJSON's representation.
		return t.Format(time.RFC3339)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// unionKeys collects the keys of all object elements: sorted within each
// element (map iteration is unordered), in first-appearance order across
// elements. Keys missing from an element become empty cells.
func unionKeys(arr []interface{}) []string {
	var keys []string
	seen := make(map[string]bool)
	for _, el := range arr {
		m := el.(map[string]interface{})
		k := make([]string, 0, len(m))
		for key := range m {
			k = append(k, key)
		}
		sort.Strings(k)
		for _, key := range k {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}
