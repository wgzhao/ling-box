package dbf

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// fieldSpec describes one field for buildDBF.
type fieldSpec struct {
	name    string
	typ     byte
	length  int
	decimal int
}

// buildDBF assembles a minimal DBF file: 32-byte header, field
// descriptors, 0x0D terminator, then one record per row. Each row is
// the concatenation of its field values; buildDBF slices it by field
// length and pads each field with spaces to its length. Records are
// prefixed with the 0x20 "not deleted" flag. extraHeader adds padding
// bytes after the 0x0D (e.g. a VFP backlink) which the header size
// must then cover.
func buildDBF(t *testing.T, version, ldid byte, fields []fieldSpec, rows [][]byte, extraHeader int) []byte {
	t.Helper()
	recSize := 1
	for _, f := range fields {
		recSize += f.length
	}
	headerSize := 32 + 32*len(fields) + 1 + extraHeader
	out := make([]byte, 0, headerSize+recSize*len(rows)+1)
	h := make([]byte, 32)
	h[0] = version
	h[1], h[2], h[3] = 26, 8, 25 // last update 2026-08-25
	binary.LittleEndian.PutUint32(h[4:8], uint32(len(rows)))
	binary.LittleEndian.PutUint16(h[8:10], uint16(headerSize))
	binary.LittleEndian.PutUint16(h[10:12], uint16(recSize))
	h[29] = ldid
	out = append(out, h...)
	for _, f := range fields {
		d := make([]byte, 32)
		copy(d[0:11], f.name)
		d[11] = f.typ
		d[16] = byte(f.length)
		d[17] = byte(f.decimal)
		out = append(out, d...)
	}
	out = append(out, 0x0D)
	out = append(out, make([]byte, extraHeader)...)
	for _, row := range rows {
		out = append(out, 0x20)
		out = append(out, row...)
		if len(row) < recSize-1 {
			out = append(out, make([]byte, recSize-1-len(row))...)
		}
	}
	out = append(out, 0x1A)
	return out
}

// rec concatenates field values (each already padded to its field
// width) into one record byte slice.
func rec(fields ...[]byte) []byte {
	var out []byte
	for _, f := range fields {
		out = append(out, f...)
	}
	return out
}

// pad pads v with spaces to width, truncating when wider.
func pad(v []byte, width int) []byte {
	if len(v) >= width {
		return v[:width]
	}
	out := make([]byte, width)
	copy(out, v)
	for i := len(v); i < width; i++ {
		out[i] = ' '
	}
	return out
}

// gbkEnc encodes a string to GBK bytes for fixtures.
func gbkEnc(t *testing.T, s string) []byte {
	t.Helper()
	out, _, err := transform.Bytes(simplifiedchinese.GBK.NewEncoder(), []byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOpenHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.dbf")
	fields := []fieldSpec{{"NAME", 'C', 10, 0}, {"AGE", 'N', 3, 0}, {"BIRTH", 'D', 8, 0}}
	rows := [][]byte{
		rec(pad([]byte("alice"), 10), pad([]byte(" 25"), 3), pad([]byte("20000101"), 8)),
	}
	writeFile(t, path, buildDBF(t, 0x03, 0x00, fields, rows, 0))

	tbl, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	h := tbl.Header
	if h.VersionName != "dBASE III+ (no memo)" {
		t.Errorf("VersionName = %q", h.VersionName)
	}
	if h.RecordCount != 1 || h.RecordSize != 22 || h.HeaderSize != 129 {
		t.Errorf("header = %+v", h)
	}
	if len(h.Fields) != 3 {
		t.Fatalf("fields = %d", len(h.Fields))
	}
	if h.Fields[0].Name != "NAME" || h.Fields[0].Type != 'C' || h.Fields[0].Length != 10 {
		t.Errorf("field0 = %+v", h.Fields[0])
	}
	if h.Fields[2].Offset != 13 {
		t.Errorf("field2 offset = %d", h.Fields[2].Offset)
	}
	if h.LastUpdate != "2026-08-25" {
		t.Errorf("LastUpdate = %q", h.LastUpdate)
	}
	rows2, err := tbl.Rows(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows2) != 1 {
		t.Fatalf("rows = %d", len(rows2))
	}
	if rows2[0].Deleted || rows2[0].Values[0] != "alice" || rows2[0].Values[1] != "25" || rows2[0].Values[2] != "2000-01-01" {
		t.Errorf("row = %+v", rows2[0])
	}
}

func TestOpenVFPBacklink(t *testing.T) {
	// Visual FoxPro appends a 263-byte backlink after the 0x0D; the
	// header size drives the data start.
	dir := t.TempDir()
	path := filepath.Join(dir, "vfp.dbf")
	fields := []fieldSpec{{"X", 'C', 4, 0}}
	rows := [][]byte{rec(pad([]byte("data"), 4))}
	writeFile(t, path, buildDBF(t, 0x30, 0x00, fields, rows, 263))

	tbl, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Header.HeaderSize != 32+32+1+263 {
		t.Errorf("HeaderSize = %d", tbl.Header.HeaderSize)
	}
	recs, _ := tbl.Rows(true)
	if len(recs) != 1 || recs[0].Values[0] != "data" {
		t.Errorf("recs = %+v", recs)
	}
}

func TestDeletedRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "del.dbf")
	fields := []fieldSpec{{"X", 'C', 3, 0}}
	raw := buildDBF(t, 0x03, 0x00, fields, [][]byte{rec(pad([]byte("one"), 3)), rec(pad([]byte("two"), 3))}, 0)
	// Mark the second record deleted (0x2A flag).
	recSize := 4
	raw[32+32+1+recSize] = 0x2A
	writeFile(t, path, raw)

	tbl, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	recs, _ := tbl.Rows(true)
	if len(recs) != 2 || recs[0].Deleted || !recs[1].Deleted {
		t.Errorf("recs = %+v", recs)
	}
	if recs[1].Values[0] != "two" {
		t.Errorf("deleted record value = %q", recs[1].Values[0])
	}
}

func TestGBKEncoding(t *testing.T) {
	dir := t.TempDir()
	fields := []fieldSpec{{"NAME", 'C', 10, 0}}
	chinese := gbkEnc(t, "你好")
	for _, ldid := range []byte{0x4D, 0x7A, 0x00} { // 0x00 -> default GBK
		path := filepath.Join(dir, "gbk.dbf")
		writeFile(t, path, buildDBF(t, 0x03, ldid, fields, [][]byte{rec(pad(chinese, 10))}, 0))
		tbl, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		recs, _ := tbl.Rows(true)
		if recs[0].Values[0] != "你好" {
			t.Errorf("ldid 0x%02x: value = %q", ldid, recs[0].Values[0])
		}
	}
}

func TestEncodingOverride(t *testing.T) {
	dir := t.TempDir()
	fields := []fieldSpec{{"X", 'C', 8, 0}}

	// GB18030: U+20000 encodes as 4 bytes, invalid in GBK.
	g18030 := []byte{0x95, 0x32, 0x82, 0x36}
	path := filepath.Join(dir, "g.dbf")
	writeFile(t, path, buildDBF(t, 0x03, 0x00, fields, [][]byte{rec(pad(g18030, 8))}, 0))
	tbl, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if v := mustRow(t, tbl)[0]; v == "𠀀" {
		t.Errorf("GBK default unexpectedly decoded GB18030 byte: %q", v)
	}
	tbl2, err := openWithEncoding(path, "gb18030")
	if err != nil {
		t.Fatal(err)
	}
	if v := mustRow(t, tbl2)[0]; v != "𠀀" {
		t.Errorf("gb18030 override = %q", v)
	}

	// cp1252 override.
	cp1252 := []byte{0xE9, 0xEF}
	path2 := filepath.Join(dir, "w.dbf")
	writeFile(t, path2, buildDBF(t, 0x03, 0x00, fields, [][]byte{rec(pad(cp1252, 8))}, 0))
	tbl3, err := openWithEncoding(path2, "cp1252")
	if err != nil {
		t.Fatal(err)
	}
	if v := mustRow(t, tbl3)[0]; v != "éï" {
		t.Errorf("cp1252 override = %q", v)
	}
}

func TestOpenErrors(t *testing.T) {
	dir := t.TempDir()

	// Too short.
	writeFile(t, filepath.Join(dir, "tiny.dbf"), make([]byte, 10))
	if _, err := Open(filepath.Join(dir, "tiny.dbf")); err == nil {
		t.Error("tiny file: expected error")
	}

	// Encrypted flag set.
	fields := []fieldSpec{{"X", 'C', 1, 0}}
	raw := buildDBF(t, 0x03, 0x00, fields, nil, 0)
	raw[15] = 0x01
	writeFile(t, filepath.Join(dir, "enc.dbf"), raw)
	if _, err := Open(filepath.Join(dir, "enc.dbf")); err == nil || !strings.Contains(err.Error(), "encrypted") {
		t.Errorf("encrypted: err = %v", err)
	}

	// Missing 0x0D terminator.
	raw2 := buildDBF(t, 0x03, 0x00, fields, nil, 0)
	raw2[32+32] = 0x00 // overwrite the 0x0D
	raw2[4] = 0        // zero records so header size still ends at terminator area
	writeFile(t, filepath.Join(dir, "term.dbf"), raw2)
	if _, err := Open(filepath.Join(dir, "term.dbf")); err == nil || !strings.Contains(err.Error(), "0x0D") {
		t.Errorf("missing terminator: err = %v", err)
	}
}

func TestValueTypes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "types.dbf")
	fields := []fieldSpec{
		{"C1", 'C', 6, 0},
		{"N1", 'N', 6, 2},
		{"NOVF", 'N', 4, 0},
		{"D1", 'D', 8, 0},
		{"DEMPTY", 'D', 8, 0},
		{"L1", 'L', 1, 0},
		{"L2", 'L', 1, 0},
		{"L3", 'L', 1, 0},
		{"I1", 'I', 4, 0},
		{"T1", 'T', 8, 0},
		{"Y1", 'Y', 8, 0},
		{"B1", 'B', 8, 2},
		{"UNK", 'Q', 4, 0},
	}
	var i1 [4]byte
	n := int32(-42)
	binary.LittleEndian.PutUint32(i1[:], uint32(n))
	var t1 [8]byte
	binary.LittleEndian.PutUint32(t1[0:4], 61042) // julian day for 2026-01-02
	binary.LittleEndian.PutUint32(t1[4:8], 45296000)
	var y1 [8]byte
	binary.LittleEndian.PutUint64(y1[:], 1234567)
	var b1 [8]byte
	binary.LittleEndian.PutUint64(b1[:], math.Float64bits(3.14))

	row := [][]byte{rec(
		pad([]byte("ab\x00cd\x00"), 6), // NUL-cut
		pad([]byte(" 12.50"), 6),       // numeric
		pad([]byte("****"), 4),         // overflow sentinel
		pad([]byte("20260102"), 8),
		pad([]byte("        "), 8), // empty date
		pad([]byte("T"), 1),
		pad([]byte("f"), 1),
		pad([]byte("?"), 1),
		i1[:],
		t1[:],
		y1[:],
		b1[:],
		pad([]byte("raw!"), 4),
	)}
	writeFile(t, path, buildDBF(t, 0x30, 0x00, fields, row, 0))
	tbl, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	v := mustRow(t, tbl)
	want := []string{"ab", "12.50", "****", "2026-01-02", "", "true", "false", "", "-42", "2026-01-02 12:34:56", "123.4567", "3.14", "raw!"}
	for i, w := range want {
		if v[i] != w {
			t.Errorf("value[%d] = %q, want %q", i, v[i], w)
		}
	}
}

// mustRow opens the table and returns the first record.
func mustRow(t *testing.T, tbl *Table) []string {
	t.Helper()
	recs, err := tbl.Rows(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) == 0 {
		t.Fatal("no records")
	}
	return recs[0].Values
}

// openWithEncoding opens a DBF with an explicit encoding override.
func openWithEncoding(path, enc string) (*Table, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	h, err := parseHeader(raw)
	if err != nil {
		return nil, err
	}
	cp, dec, err := resolveEncodingWith(h.LanguageID, enc)
	if err != nil {
		return nil, err
	}
	h.CodePage = cp
	end := h.HeaderSize + h.RecordCount*h.RecordSize
	if end > len(raw) {
		end = len(raw)
	}
	return &Table{Header: h, dec: dec, raw: raw, data: raw[h.HeaderSize:end]}, nil
}
