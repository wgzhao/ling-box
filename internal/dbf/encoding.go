package dbf

import (
	"fmt"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

// decodeFunc decodes a text field's raw bytes into a string.
type decodeFunc func([]byte) string

// ldidCodePages maps the language driver ID (header byte 29) to the
// code page of the table. Only the commonly used IDs are listed;
// anything else falls back to GBK.
var ldidCodePages = map[byte]string{
	0x01: "cp437",
	0x02: "cp850",
	0x03: "cp1252",
	0x08: "cp865",
	0x0F: "cp437",
	0x13: "shiftjis", // cp932
	0x4D: "gbk",      // cp936
	0x4E: "euckr",    // cp949
	0x4F: "big5",     // cp950
	0x57: "cp1252",
	0x58: "cp1252",
	0x59: "cp1252",
	0x64: "cp852",
	0x65: "cp866",
	0x66: "cp865",
	0x6C: "cp863",
	0x78: "big5",     // cp950
	0x79: "euckr",    // cp949
	0x7A: "gbk",      // cp936
	0x7B: "shiftjis", // cp932
	0xC8: "cp1250",
	0xC9: "cp1251",
	0xCA: "cp1254",
	0xF0: "utf-8",
}

// encodingNames is the set accepted by the -e flag, in display order.
// (cp857/869/737/874 are not covered by x/text's charmap and are
// omitted; unknown language driver IDs fall back to GBK.)
var encodingNames = []string{
	"auto", "gbk", "gb18030", "big5", "shiftjis", "euckr",
	"utf-8", "ascii",
	"cp437", "cp850", "cp852", "cp863", "cp865", "cp866",
	"cp1250", "cp1251", "cp1252", "cp1254",
}

// EncodingNames returns the encodings accepted by -e, for help text.
func EncodingNames() string {
	return strings.Join(encodingNames, "|")
}

// detectEncoding resolves a language driver ID to a code page name.
// An unknown or zero ID returns "".
func detectEncoding(ldid byte) string {
	if cp, ok := ldidCodePages[ldid]; ok {
		return cp
	}
	return ""
}

// resolveEncoding picks the decoder for the table: the explicit
// encoding from -e when given, otherwise the code page from the
// language driver ID, defaulting to GBK (most Chinese DBF files leave
// byte 29 unset).
func resolveEncoding(ldid byte) (string, decodeFunc, error) {
	return resolveEncodingWith(ldid, "")
}

// resolveEncodingWith resolves the decoder, honoring an explicit
// override name ("" or "auto" = auto-detect).
func resolveEncodingWith(ldid byte, override string) (string, decodeFunc, error) {
	name := override
	if name == "auto" {
		name = ""
	}
	if name == "" {
		name = detectEncoding(ldid)
	}
	if name == "" {
		name = "gbk"
	}
	dec, err := decoderFor(name)
	if err != nil {
		return "", nil, err
	}
	return name, dec, nil
}

// decoderFor returns a decodeFunc for a code page name.
func decoderFor(name string) (decodeFunc, error) {
	var enc encoding.Encoding
	switch name {
	case "utf-8", "ascii":
		enc = nil // identity
	case "gbk", "gb18030":
		if name == "gb18030" {
			enc = simplifiedchinese.GB18030
		} else {
			enc = simplifiedchinese.GBK
		}
	case "big5":
		enc = traditionalchinese.Big5
	case "shiftjis":
		enc = japanese.ShiftJIS
	case "euckr":
		enc = korean.EUCKR
	case "cp437":
		enc = charmap.CodePage437
	case "cp850":
		enc = charmap.CodePage850
	case "cp852":
		enc = charmap.CodePage852
	case "cp863":
		enc = charmap.CodePage863
	case "cp865":
		enc = charmap.CodePage865
	case "cp866":
		enc = charmap.CodePage866
	case "cp1250":
		enc = charmap.Windows1250
	case "cp1251":
		enc = charmap.Windows1251
	case "cp1252":
		enc = charmap.Windows1252
	case "cp1254":
		enc = charmap.Windows1254
	default:
		return nil, fmt.Errorf("unknown encoding %q (valid: %s)", name, EncodingNames())
	}
	if enc == nil {
		return func(b []byte) string { return string(b) }, nil
	}
	return func(b []byte) string {
		// A fresh decoder per call keeps transformer state from
		// leaking across fields with trailing partial sequences.
		out, _, err := transform.Bytes(enc.NewDecoder(), b)
		if err != nil {
			// Invalid bytes: fall back to the raw bytes with invalid
			// UTF-8 sequences replaced, so one bad field never breaks
			// the whole table.
			return strings.ToValidUTF8(string(b), "�")
		}
		return string(out)
	}, nil
}
