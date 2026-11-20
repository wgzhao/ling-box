package textx

import "testing"

func TestCaseConversions(t *testing.T) {
	tests := []struct {
		in       string
		camel    string
		pascal   string
		snake    string
		kebab    string
		constant string
		title    string
		slug     string
	}{
		{"Hello World", "helloWorld", "HelloWorld", "hello_world", "hello-world", "HELLO_WORLD", "Hello World", "hello-world"},
		{"fooBar", "fooBar", "FooBar", "foo_bar", "foo-bar", "FOO_BAR", "Foo Bar", "foo-bar"},
		{"HTTPServer", "httpServer", "HttpServer", "http_server", "http-server", "HTTP_SERVER", "Http Server", "http-server"},
		{"XMLHttpRequest", "xmlHttpRequest", "XmlHttpRequest", "xml_http_request", "xml-http-request", "XML_HTTP_REQUEST", "Xml Http Request", "xml-http-request"},
		{"version 2 api", "version2Api", "Version2Api", "version_2_api", "version-2-api", "VERSION_2_API", "Version 2 Api", "version-2-api"},
		{"你好 世界", "你好世界", "你好世界", "你好_世界", "你好-世界", "你好_世界", "你好 世界", "你好-世界"},
		{"  Hello,   World!  ", "helloWorld", "HelloWorld", "hello_world", "hello-world", "HELLO_WORLD", "Hello World", "hello-world"},
		{"", "", "", "", "", "", "", ""},
	}
	for _, tt := range tests {
		if got := Camel(tt.in); got != tt.camel {
			t.Errorf("Camel(%q) = %q, want %q", tt.in, got, tt.camel)
		}
		if got := Pascal(tt.in); got != tt.pascal {
			t.Errorf("Pascal(%q) = %q, want %q", tt.in, got, tt.pascal)
		}
		if got := Snake(tt.in); got != tt.snake {
			t.Errorf("Snake(%q) = %q, want %q", tt.in, got, tt.snake)
		}
		if got := Kebab(tt.in); got != tt.kebab {
			t.Errorf("Kebab(%q) = %q, want %q", tt.in, got, tt.kebab)
		}
		if got := Constant(tt.in); got != tt.constant {
			t.Errorf("Constant(%q) = %q, want %q", tt.in, got, tt.constant)
		}
		if got := Title(tt.in); got != tt.title {
			t.Errorf("Title(%q) = %q, want %q", tt.in, got, tt.title)
		}
		if got := Slug(tt.in); got != tt.slug {
			t.Errorf("Slug(%q) = %q, want %q", tt.in, got, tt.slug)
		}
	}
}

func TestEncodingConversions(t *testing.T) {
	raw := `<a href="x">'&'</a>`
	encoded := HTMLEncode(raw)
	if encoded == raw {
		t.Fatal("HTMLEncode did not change the input")
	}
	if got := HTMLDecode(encoded); got != raw {
		t.Errorf("HTML round trip = %q, want %q", got, raw)
	}

	text := "hello\n世界"
	escaped, err := JSONEscape(text)
	if err != nil {
		t.Fatal(err)
	}
	if escaped != `hello\n世界` {
		t.Errorf("JSONEscape = %q", escaped)
	}
	unescaped, err := JSONUnescape(`hello\n世界`)
	if err != nil {
		t.Fatal(err)
	}
	if unescaped != text {
		t.Errorf("JSONUnescape = %q, want %q", unescaped, text)
	}
	if _, err := JSONUnescape(`\uZZZZ`); err == nil {
		t.Error("JSONUnescape invalid escape: expected error")
	}

	if got := Fullwidth("ABC123"); got != "ＡＢＣ１２３" {
		t.Errorf("Fullwidth = %q", got)
	}
	if got := Halfwidth("ＡＢＣ１２３"); got != "ABC123" {
		t.Errorf("Halfwidth = %q", got)
	}
}

func TestStats(t *testing.T) {
	got := Analyze("Hello 世界\nsecond line")
	if got.Characters != 20 {
		t.Errorf("characters = %d, want 20", got.Characters)
	}
	if got.Bytes != 24 {
		t.Errorf("bytes = %d, want 24", got.Bytes)
	}
	if got.Words != 4 {
		t.Errorf("words = %d, want 4", got.Words)
	}
	if got.Lines != 2 {
		t.Errorf("lines = %d, want 2", got.Lines)
	}
	if got.CJK != 2 {
		t.Errorf("cjk = %d, want 2", got.CJK)
	}
	if got.ReadingMinutes < 1 {
		t.Errorf("reading minutes = %d, want at least 1", got.ReadingMinutes)
	}
}

func TestApply(t *testing.T) {
	if got, err := Apply("snake", "Hello World"); err != nil || got != "hello_world" {
		t.Errorf("Apply snake = %q, %v", got, err)
	}
	if got, err := Apply("width-full", "ABC"); err != nil || got != "ＡＢＣ" {
		t.Errorf("Apply width-full = %q, %v", got, err)
	}
	if _, err := Apply("nope", "x"); err == nil {
		t.Error("unknown operation: expected error")
	}
}
