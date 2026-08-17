package convert

import (
	"encoding/csv"
	"strings"
	"testing"
)

// parseCSV reads CSV output back into rows for field-level assertions.
// csv.Reader handles all quoting, so escaping is verified by round-trip
// rather than raw string comparison.
func parseCSVOutput(t *testing.T, out []byte) [][]string {
	t.Helper()
	r := csv.NewReader(strings.NewReader(string(out)))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse CSV output: %v", err)
	}
	return rows
}

func wantCSV(t *testing.T, got [][]string, want [][]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("rows = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if len(got[i]) != len(want[i]) {
			t.Fatalf("row %d = %v, want %v", i, got[i], want[i])
		}
		for j := range want[i] {
			if got[i][j] != want[i][j] {
				t.Errorf("cell [%d][%d] = %q, want %q", i, j, got[i][j], want[i][j])
			}
		}
	}
}

func TestToCSVObjectError(t *testing.T) {
	_, err := ToCSV([]byte(`{"a":1}`))
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "object") {
		t.Errorf("error %q should name the object kind", err)
	}
}

func TestToCSVTopLevelKindsError(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`"str"`, "string"},
		{`42`, "number"},
		{`true`, "boolean"},
		{`null`, "null"},
	}
	for _, c := range cases {
		_, err := ToCSV([]byte(c.in))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("ToCSV(%s): error %v should name %s", c.in, err, c.want)
		}
	}
}

func TestToCSVEmptyArrayError(t *testing.T) {
	_, err := ToCSV([]byte(`[]`))
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("ToCSV([]): error %v should say the array is empty", err)
	}
}

func TestToCSVEmptyInputError(t *testing.T) {
	for _, in := range []string{``, `  `, "\n\t"} {
		_, err := ToCSV([]byte(in))
		if err == nil || !strings.Contains(err.Error(), "empty input") {
			t.Errorf("ToCSV(%q): error %v should say empty input", in, err)
		}
	}
}

func TestToCSVInvalidInputError(t *testing.T) {
	_, err := ToCSV([]byte(`{{bad`))
	if err == nil || !strings.Contains(err.Error(), "unable to detect format") {
		t.Errorf("ToCSV({{bad): error %v should say format undetectable", err)
	}
}

func TestToCSVMixedElementsError(t *testing.T) {
	_, err := ToCSV([]byte(`[{"a":1}, 2]`))
	if err == nil || !strings.Contains(err.Error(), "mixes objects and scalar") {
		t.Errorf("ToCSV(mixed): error %v should reject mixed elements", err)
	}
}

func TestToCSVArrayElementError(t *testing.T) {
	_, err := ToCSV([]byte(`[[1,2],[3]]`))
	if err == nil || !strings.Contains(err.Error(), "element 0 is array") {
		t.Errorf("ToCSV(nested array): error %v should name element 0", err)
	}
}

func TestToCSVEmptyObjectsError(t *testing.T) {
	_, err := ToCSV([]byte(`[{},{}]`))
	if err == nil || !strings.Contains(err.Error(), "no fields") {
		t.Errorf("ToCSV([{},{}]): error %v should say no fields", err)
	}
}

func TestToCSVScalarArray(t *testing.T) {
	// null as a trailing element is covered by TestToCSVNullLastRow:
	// csv.Reader skips its blank line, so it cannot be round-tripped.
	out, err := ToCSV([]byte(`[1, "two", true]`))
	if err != nil {
		t.Fatalf("ToCSV(scalars): %v", err)
	}
	wantCSV(t, parseCSVOutput(t, out), [][]string{
		{"1"}, {"two"}, {"true"},
	})
}

func TestToCSVHeaderUnionOrder(t *testing.T) {
	out, err := ToCSV([]byte(`[{"b":2,"a":1},{"c":3,"a":4}]`))
	if err != nil {
		t.Fatalf("ToCSV(union): %v", err)
	}
	wantCSV(t, parseCSVOutput(t, out), [][]string{
		{"a", "b", "c"},
		{"1", "2", ""},
		{"4", "", "3"},
	})
}

func TestToCSVMissingKeyAndNull(t *testing.T) {
	out, err := ToCSV([]byte(`[{"a":1,"b":null},{"a":2}]`))
	if err != nil {
		t.Fatalf("ToCSV(null): %v", err)
	}
	wantCSV(t, parseCSVOutput(t, out), [][]string{
		{"a", "b"},
		{"1", ""},
		{"2", ""},
	})
}

func TestToCSVNestedStringify(t *testing.T) {
	out, err := ToCSV([]byte(`[{"n":{"x":1},"l":[1,"a"]}]`))
	if err != nil {
		t.Fatalf("ToCSV(nested): %v", err)
	}
	wantCSV(t, parseCSVOutput(t, out), [][]string{
		{"l", "n"},
		{`[1,"a"]`, `{"x":1}`},
	})
}

func TestToCSVQuoting(t *testing.T) {
	// The \n inside the JSON string is a real newline once parsed; the
	// CSV writer must quote the field so it survives the round-trip.
	in := `[{"a":"x,y","b":"say \"hi\"","c":"line1\nline2","d":" leading"}]`
	out, err := ToCSV([]byte(in))
	if err != nil {
		t.Fatalf("ToCSV(quoting): %v", err)
	}
	wantCSV(t, parseCSVOutput(t, out), [][]string{
		{"a", "b", "c", "d"},
		{"x,y", `say "hi"`, "line1\nline2", " leading"},
	})
}

func TestToCSVNullLastRow(t *testing.T) {
	// A null element is written as an empty row (a bare newline). The
	// full line structure must survive — trimming the trailing newline
	// would delete the row entirely. Note csv.Reader skips blank lines,
	// so this asserts raw bytes, not a round-trip.
	out, err := ToCSV([]byte(`[1, null]`))
	if err != nil {
		t.Fatalf("ToCSV(null last): %v", err)
	}
	if string(out) != "1\n\n" {
		t.Errorf("ToCSV([1, null]) = %q, want %q", out, "1\n\n")
	}
}

func TestToCSVFloatFormat(t *testing.T) {
	out, err := ToCSV([]byte(`[1e6, 0.5, 1e-7]`))
	if err != nil {
		t.Fatalf("ToCSV(floats): %v", err)
	}
	wantCSV(t, parseCSVOutput(t, out), [][]string{
		{"1000000"}, {"0.5"}, {"0.0000001"},
	})
}

func TestToCSVFromYAML(t *testing.T) {
	in := "- 1: one\n  a: x\n- a: y\n"
	out, err := ToCSV([]byte(in))
	if err != nil {
		t.Fatalf("ToCSV(YAML): %v", err)
	}
	// The non-string key 1 goes through normalize() and becomes "1".
	wantCSV(t, parseCSVOutput(t, out), [][]string{
		{"1", "a"},
		{"one", "x"},
		{"", "y"},
	})
}

func TestToCSVYAMLTimestamp(t *testing.T) {
	// yaml.v3 resolves an unquoted date to time.Time; cellString emits
	// it as RFC3339, matching YAMLToJSON behavior.
	out, err := ToCSV([]byte("- d: 2026-08-17\n"))
	if err != nil {
		t.Fatalf("ToCSV(timestamp): %v", err)
	}
	wantCSV(t, parseCSVOutput(t, out), [][]string{
		{"d"},
		{"2026-08-17T00:00:00Z"},
	})
}

func TestToCSVFullLineStructure(t *testing.T) {
	// Every record ends with a newline; the caller prints with
	// fmt.Print, so the last line is complete and not doubled.
	out, err := ToCSV([]byte(`[{"a":1},{"a":2}]`))
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	if !strings.HasSuffix(string(out), "\n") {
		t.Errorf("output should keep its final newline: %q", out)
	}
	wantCSV(t, parseCSVOutput(t, out), [][]string{
		{"a"}, {"1"}, {"2"},
	})
}
