package jsonx

import (
	"strings"
	"testing"
)

const messy = `{"b":1,"a":[true,null,"x"],"c":{"d":2}}`

func TestFormatDefault(t *testing.T) {
	out, err := Format([]byte(messy), FormatOptions{})
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	want := `{
  "b": 1,
  "a": [
    true,
    null,
    "x"
  ],
  "c": {
    "d": 2
  }
}`
	if string(out) != want {
		t.Errorf("Format output:\n%s\nwant:\n%s", out, want)
	}
	// Key order must be preserved (b before a).
	if !strings.HasPrefix(string(out), "{\n  \"b\":") {
		t.Errorf("key order not preserved: %s", out)
	}
}

func TestFormatIndent(t *testing.T) {
	out, err := Format([]byte(`{"a":1}`), FormatOptions{Indent: "    "})
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if string(out) != "{\n    \"a\": 1\n}" {
		t.Errorf("Format(4 spaces) = %q", out)
	}
}

func TestFormatSortKeys(t *testing.T) {
	out, err := Format([]byte(messy), FormatOptions{SortKeys: true})
	if err != nil {
		t.Fatalf("Format(sort): %v", err)
	}
	if !strings.HasPrefix(string(out), "{\n  \"a\":") {
		t.Errorf("SortKeys did not reorder: %s", out)
	}
	if !strings.Contains(string(out), `"c": {`) || !strings.Contains(string(out), `"d": 2`) {
		t.Errorf("SortKeys lost nested data: %s", out)
	}
}

func TestFormatCompact(t *testing.T) {
	out, err := Format([]byte(messy), FormatOptions{Compact: true})
	if err != nil {
		t.Fatalf("Format(compact): %v", err)
	}
	if string(out) != `{"b":1,"a":[true,null,"x"],"c":{"d":2}}` {
		t.Errorf("Format(compact) = %q", out)
	}
}

func TestFormatSortAndCompact(t *testing.T) {
	out, err := Format([]byte(`{"b":1,"a":2}`), FormatOptions{SortKeys: true, Compact: true})
	if err != nil {
		t.Fatalf("Format(sort+compact): %v", err)
	}
	if string(out) != `{"a":2,"b":1}` {
		t.Errorf("Format(sort+compact) = %q, want sorted single line", out)
	}
}

func TestFormatInvalid(t *testing.T) {
	for _, bad := range []string{`{"a":}`, `[1,2`, `{'a':1}`, ``} {
		if _, err := Format([]byte(bad), FormatOptions{}); err == nil {
			t.Errorf("Format(%q): want error, got nil", bad)
		}
	}
}

func TestVerifyValid(t *testing.T) {
	for _, good := range []string{
		`{}`,
		`{"a":[1,2,{"b":null}]}`,
		`"just a string"`,
		`42`,
		`true`,
		`null`,
		`  {"padded": true}  `, // surrounding whitespace is fine
	} {
		if err := Verify([]byte(good)); err != nil {
			t.Errorf("Verify(%q): %v", good, err)
		}
	}
}

func TestVerifyInvalid(t *testing.T) {
	cases := []struct {
		in   string
		want string // substring of the error
	}{
		{`{"a":}`, "line 1"},
		{"{\n  \"a\": 1,\n  \"b\": [1, 2,\n}", "line 4"},
		{`{'a':1}`, "line 1"},
		{`[1,2`, "line 1"},
		{``, "invalid JSON"},
	}
	for _, c := range cases {
		err := Verify([]byte(c.in))
		if err == nil {
			t.Errorf("Verify(%q): want error, got nil", c.in)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("Verify(%q) error %q does not contain %q", c.in, err, c.want)
		}
	}
	// The location must point at the real breakage, not the start.
	err := Verify([]byte("{\n  \"a\": 1,\n  \"b\": [1, 2,\n}"))
	if err != nil && !strings.Contains(err.Error(), "line 4") {
		t.Errorf("multi-line error location wrong: %v", err)
	}
}
