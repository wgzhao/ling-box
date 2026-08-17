package convert

import (
	"strings"
	"testing"
)

func TestToMarkdownTable(t *testing.T) {
	out, err := ToMarkdown([]byte(`[{"name":"alice","age":30},{"name":"bob","age":25}]`))
	if err != nil {
		t.Fatalf("ToMarkdown(table): %v", err)
	}
	want := `| age | name |
| --- | --- |
| 30 | alice |
| 25 | bob |`
	if string(out) != want {
		t.Errorf("ToMarkdown(table):\n%s\nwant:\n%s", out, want)
	}
}

func TestToMarkdownList(t *testing.T) {
	out, err := ToMarkdown([]byte(`[1, "two", true]`))
	if err != nil {
		t.Fatalf("ToMarkdown(list): %v", err)
	}
	if string(out) != "- 1\n- two\n- true" {
		t.Errorf("ToMarkdown(list) = %q", out)
	}
}

func TestToMarkdownKVTable(t *testing.T) {
	out, err := ToMarkdown([]byte(`{"b":2,"a":1}`))
	if err != nil {
		t.Fatalf("ToMarkdown(kv): %v", err)
	}
	want := `| key | value |
| --- | --- |
| a | 1 |
| b | 2 |`
	if string(out) != want {
		t.Errorf("ToMarkdown(kv):\n%s\nwant:\n%s", out, want)
	}
}

func TestToMarkdownNestedStringify(t *testing.T) {
	out, err := ToMarkdown([]byte(`[{"n":{"x":1},"tags":["a","b"]}]`))
	if err != nil {
		t.Fatalf("ToMarkdown(nested): %v", err)
	}
	if !strings.Contains(string(out), `| {"x":1} |`) {
		t.Errorf("nested object not stringified: %s", out)
	}
	if !strings.Contains(string(out), `["a","b"]`) {
		t.Errorf("nested array not stringified: %s", out)
	}
}

func TestToMarkdownPipeEscaping(t *testing.T) {
	out, err := ToMarkdown([]byte(`[{"a":"x|y"}]`))
	if err != nil {
		t.Fatalf("ToMarkdown(pipe): %v", err)
	}
	if !strings.Contains(string(out), `x\|y`) {
		t.Errorf("pipe not escaped: %s", out)
	}
}

func TestToMarkdownFromYAML(t *testing.T) {
	out, err := ToMarkdown([]byte("- name: alice\n  age: 30\n- name: bob\n  age: 25\n"))
	if err != nil {
		t.Fatalf("ToMarkdown(YAML): %v", err)
	}
	if !strings.Contains(string(out), "| age | name |") {
		t.Errorf("YAML header wrong: %s", out)
	}
}

func TestToMarkdownErrors(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`[]`, "empty"},
		{`[{"a":1}, 2]`, "mixes objects and scalar"},
		{`[[1,2]]`, "element 0 is array"},
		{`[{},{}]`, "no fields"},
		{`"str"`, "unsupported top-level value"},
	}
	for _, c := range cases {
		_, err := ToMarkdown([]byte(c.in))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("ToMarkdown(%s): error %v should contain %q", c.in, err, c.want)
		}
	}
}
