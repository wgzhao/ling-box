package convert

import (
	"strings"
	"testing"
)

func TestParseCSV(t *testing.T) {
	rows, err := Parse("csv", []byte("name,age\n alice ,30\nbob,25\n"))
	if err != nil {
		t.Fatalf("Parse(csv): %v", err)
	}
	arr, ok := rows.([]interface{})
	if !ok || len(arr) != 2 {
		t.Fatalf("Parse(csv) = %#v, want 2 elements", rows)
	}
	m := arr[0].(map[string]interface{})
	// Cells stay strings; leading spaces preserved (CSV semantics).
	if m["name"] != " alice " || m["age"] != "30" {
		t.Errorf("row 0 = %v, want name= alice , age=30", m)
	}
}

func TestParseCSVQuotedFields(t *testing.T) {
	rows, err := Parse("csv", []byte("a,b\n\"x,y\",\"say \"\"hi\"\"\"\n"))
	if err != nil {
		t.Fatalf("Parse(csv quoted): %v", err)
	}
	arr := rows.([]interface{})
	m := arr[0].(map[string]interface{})
	if m["a"] != "x,y" || m["b"] != `say "hi"` {
		t.Errorf("quoted fields = %v", m)
	}
}

func TestParseCSVErrors(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{``, "empty input"},
		{"  \n", "empty input"},
		{"a,b\n1\n", "invalid CSV"},
		{"a,a\n1,2\n", "duplicate header"},
	}
	for _, c := range cases {
		_, err := Parse("csv", []byte(c.in))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("Parse(csv %q): error %v should contain %q", c.in, err, c.want)
		}
	}
}

func TestCSVToJSON(t *testing.T) {
	data, err := Parse("csv", []byte("name,age\nalice,30\nbob,25\n"))
	if err != nil {
		t.Fatalf("Parse(csv): %v", err)
	}
	out, err := Render("json", data)
	if err != nil {
		t.Fatalf("Render(json): %v", err)
	}
	got := string(out)
	for _, want := range []string{`"name": "alice"`, `"age": "30"`, `"bob"`} {
		if !strings.Contains(got, want) {
			t.Errorf("CSV→JSON missing %s: %s", want, got)
		}
	}
}

func TestCSVToYAML(t *testing.T) {
	data, err := Parse("csv", []byte("name,age\nalice,30\n"))
	if err != nil {
		t.Fatalf("Parse(csv): %v", err)
	}
	out, err := Render("yaml", data)
	if err != nil {
		t.Fatalf("Render(yaml): %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "name: alice") || !strings.Contains(got, "age: \"30\"") {
		t.Errorf("CSV→YAML: %s", got)
	}
}

func TestCSVToMarkdown(t *testing.T) {
	data, err := Parse("csv", []byte("name,age\nalice,30\nbob,25\n"))
	if err != nil {
		t.Fatalf("Parse(csv): %v", err)
	}
	out, err := Render("markdown", data)
	if err != nil {
		t.Fatalf("Render(markdown): %v", err)
	}
	// Header columns are sorted alphabetically by the shared renderer,
	// same as JSON/YAML input (age before name).
	want := "| age | name |\n| --- | --- |\n| 30 | alice |\n| 25 | bob |"
	if string(out) != want {
		t.Errorf("CSV→Markdown:\n%s\nwant:\n%s", out, want)
	}
}

func TestDetectInputFormat(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`{"a":1}`, "json"},
		{`[1, 2]`, "json"},
		{"a: 1\nb: 2\n", "yaml"},
		{"- a\n- b\n", "yaml"},
		{"name,age\nalice,30\nbob,25\n", "csv"},
		{"a,\"x,y\"\nb,\"c,d\"\n", "csv"}, // quoted comma inside a cell
		{"hello world", "yaml"},           // plain scalar string falls back to YAML
	}
	for _, c := range cases {
		got, err := DetectInputFormat([]byte(c.in))
		if err != nil || got != c.want {
			t.Errorf("DetectInputFormat(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestDetectInputFormatErrors(t *testing.T) {
	for _, in := range []string{"", "   "} {
		if _, err := DetectInputFormat([]byte(in)); err == nil {
			t.Errorf("DetectInputFormat(%q): want error, got nil", in)
		}
	}
}
