// Package dbf reads dBase / FoxPro DBF table files: the file header,
// field descriptors, and fixed-length records. Text fields are decoded
// from the table's code page (language driver ID), defaulting to GBK,
// and memo fields are resolved from the companion .fpt or .dbt file
// when present.
package dbf

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Field describes one column of a DBF table.
type Field struct {
	Name    string
	Type    byte // 'C','N','F','D','L','M','I','T','Y','B',...
	Length  int  // descriptor byte 16
	Decimal int  // descriptor byte 17
	Offset  int  // byte offset within the record, delete flag excluded
}

// Header is the parsed 32-byte DBF file header plus field descriptors.
type Header struct {
	Version     byte
	VersionName string // "dBASE III+", "Visual FoxPro", ...
	LastUpdate  string // "2006-01-02", or "" when unset
	RecordCount int
	HeaderSize  int // byte offset of the first data record
	RecordSize  int // bytes per record, including the delete flag
	Fields      []Field
	LanguageID  byte // byte 29
	CodePage    string
	HasMemo     bool   // any field of type M
	MemoFile    string // companion memo file found ("" = none)
}

// Record is one decoded row.
type Record struct {
	Deleted bool
	Values  []string // one per field
	MemoErr string   // set when a memo field was referenced but unreadable
}

// Table is an opened DBF file ready for record iteration.
type Table struct {
	Header *Header
	dec    decodeFunc
	raw    []byte // the whole file
	data   []byte // the record area
	memo   *memoFile
}

// FieldTypeName returns a human-readable name for a field type letter.
func FieldTypeName(t byte) string {
	switch t {
	case 'C':
		return "Character"
	case 'N':
		return "Numeric"
	case 'F':
		return "Float"
	case 'D':
		return "Date"
	case 'L':
		return "Logical"
	case 'M':
		return "Memo"
	case 'I':
		return "Integer"
	case 'T':
		return "DateTime"
	case 'Y':
		return "Currency"
	case 'B':
		return "Double"
	case 'G':
		return "General"
	case 'P':
		return "Picture"
	default:
		return "Unknown"
	}
}

// versionNames maps version bytes to format names.
var versionNames = map[byte]string{
	0x02: "FoxBASE",
	0x03: "dBASE III+ (no memo)",
	0x04: "dBASE IV (no memo)",
	0x05: "dBASE V (no memo)",
	0x30: "Visual FoxPro",
	0x31: "Visual FoxPro (autoincrement)",
	0x32: "Visual FoxPro (VARCHAR/VARBINARY)",
	0x43: "dBASE IV SQL (no memo)",
	0x63: "dBASE IV SQL",
	0x7B: "dBASE IV (no memo)",
	0x83: "dBASE III+ (memo)",
	0x8B: "dBASE IV (memo)",
	0xCB: "dBASE IV SQL (memo)",
	0xF5: "FoxPro 2.x (memo)",
	0xFB: "FoxBASE (memo)",
}

// Open reads and validates a DBF file, auto-detecting the text
// encoding from the language driver ID. The file is kept entirely in
// memory, which is fine for the small tables this tool targets.
func Open(path string) (*Table, error) {
	return OpenWithEncoding(path, "")
}

// OpenWithEncoding opens a DBF with an explicit encoding override
// ("" = auto-detect from the language driver ID, defaulting to GBK).
func OpenWithEncoding(path, encoding string) (*Table, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) < 33 {
		return nil, fmt.Errorf("file too short to be a DBF file (%d bytes)", len(raw))
	}
	h, err := parseHeader(raw)
	if err != nil {
		return nil, err
	}
	codePage, dec, err := resolveEncodingWith(h.LanguageID, encoding)
	if err != nil {
		return nil, err
	}
	h.CodePage = codePage
	// Field names may be GBK-encoded Chinese; decode them too.
	for i := range h.Fields {
		h.Fields[i].Name = dec([]byte(h.Fields[i].Name))
	}

	end := h.HeaderSize + h.RecordCount*h.RecordSize
	if end > len(raw) {
		end = len(raw)
	}
	if end < h.HeaderSize {
		end = h.HeaderSize
	}
	t := &Table{
		Header: h,
		dec:    dec,
		raw:    raw,
		data:   raw[h.HeaderSize:end],
	}
	if h.HasMemo {
		if mfPath, mf := openMemo(path, h.Version); mf != nil {
			t.memo = mf
			h.MemoFile = mfPath
		}
	}
	return t, nil
}

// parseHeader parses the file header and the field descriptors.
func parseHeader(raw []byte) (*Header, error) {
	h := &Header{
		Version:     raw[0],
		VersionName: versionNames[raw[0]],
		RecordCount: int(binary.LittleEndian.Uint32(raw[4:8])),
		HeaderSize:  int(binary.LittleEndian.Uint16(raw[8:10])),
		RecordSize:  int(binary.LittleEndian.Uint16(raw[10:12])),
		LanguageID:  raw[29],
	}
	if h.VersionName == "" {
		h.VersionName = "unknown"
	}
	if y, m, d := raw[1], raw[2], raw[3]; !(y == 0 && m == 0 && d == 0) {
		// The header stores a 2-digit year; use the usual 70/30 rule.
		year := int(y)
		if year < 70 {
			year += 2000
		} else {
			year += 1900
		}
		h.LastUpdate = fmt.Sprintf("%04d-%02d-%02d", year, m, d)
	}
	if raw[15] != 0 {
		return nil, fmt.Errorf("file is encrypted (encryption flag = 0x%02x)", raw[15])
	}
	if h.HeaderSize < 33 {
		return nil, fmt.Errorf("invalid header size %d", h.HeaderSize)
	}
	if h.RecordSize < 1 {
		return nil, fmt.Errorf("invalid record size %d", h.RecordSize)
	}
	if h.HeaderSize > len(raw) {
		return nil, fmt.Errorf("header size %d exceeds file size %d", h.HeaderSize, len(raw))
	}

	// Field descriptors: 32 bytes each from offset 32, terminated by a
	// single 0x0D. The header size (not 32+n*32+1) decides where data
	// starts: Visual FoxPro appends a 263-byte backlink after the
	// terminator.
	term := -1
	for off := 32; off < h.HeaderSize; off += 32 {
		if raw[off] == 0x0D {
			term = off
			break
		}
	}
	if term < 0 {
		return nil, fmt.Errorf("field descriptor area not terminated by 0x0D")
	}
	offset := 0
	for off := 32; off < term; off += 32 {
		d := raw[off : off+32]
		name := strings.TrimRight(strings.TrimRight(string(d[0:11]), "\x00"), " ")
		f := Field{
			Name:    name,
			Type:    d[11],
			Length:  int(d[16]),
			Decimal: int(d[17]),
			Offset:  offset,
		}
		if f.Length <= 0 {
			return nil, fmt.Errorf("field %q has invalid length %d", name, f.Length)
		}
		if f.Type == 'M' {
			h.HasMemo = true
		}
		h.Fields = append(h.Fields, f)
		offset += f.Length
	}
	if len(h.Fields) == 0 {
		return nil, fmt.Errorf("no field descriptors found (missing 0x0D terminator)")
	}
	return h, nil
}

// Rows decodes all records. Deletion flags are preserved per record.
// Memo failures are reported per record and never abort the table.
// When resolveMemo is false, memo fields are returned as their raw
// pointer text without opening the memo file.
func (t *Table) Rows(resolveMemo bool) ([]Record, error) {
	recSize := t.Header.RecordSize
	rows := make([]Record, 0, t.Header.RecordCount)
	for i := 0; i < t.Header.RecordCount; i++ {
		start := i * recSize
		if start >= len(t.data) {
			break
		}
		rec := t.data[start:]
		if len(rec) < 1 {
			break
		}
		r := Record{Deleted: rec[0] == 0x2A}
		body := rec[1:]
		for _, f := range t.Header.Fields {
			if f.Offset > len(body) {
				r.Values = append(r.Values, "")
				continue
			}
			end := f.Offset + f.Length
			if end > len(body) {
				end = len(body)
			}
			v, mErr := t.decodeValue(f, body[f.Offset:end], resolveMemo)
			if mErr != "" {
				r.MemoErr = mErr
			}
			r.Values = append(r.Values, v)
		}
		rows = append(rows, r)
	}
	return rows, nil
}

// Close releases the table. Kept for API symmetry with future file
// handle use; today the file is read eagerly.
func (t *Table) Close() error {
	return nil
}

// stripNUL cuts a fixed-width text field at the first NUL byte and
// trims trailing spaces, as writers pad with either.
func stripNUL(b []byte) []byte {
	if i := indexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return []byte(strings.TrimRight(string(b), " "))
}

func indexByte(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}

// baseName returns the file path without its extension, used to find
// the companion memo file.
func baseName(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path))
}
