package dbf

import (
	"encoding/binary"
	"path/filepath"
	"strings"
	"testing"
)

// buildDBT assembles a dBASE III .dbt: 512-byte blocks, block 0 is the
// header (next-free-block, little-endian), blocks 1.. hold text.
func buildDBT(t *testing.T, blocks [][]byte) []byte {
	t.Helper()
	out := make([]byte, 512)
	binary.LittleEndian.PutUint32(out[0:4], uint32(len(blocks)+1))
	for i, b := range blocks {
		block := make([]byte, 512)
		copy(block, b)
		out = append(out, block...)
		_ = i
	}
	return out
}

// buildFPT assembles a FoxPro .fpt: 512-byte header (next-free-block
// and block size, big-endian), then data blocks with a big-endian
// signature and length.
func buildFPT(t *testing.T, blockSize int, blocks [][]byte) []byte {
	t.Helper()
	header := make([]byte, blockSize)
	binary.BigEndian.PutUint32(header[0:4], uint32(len(blocks)+1))
	binary.BigEndian.PutUint16(header[6:8], uint16(blockSize))
	out := header
	for _, b := range blocks {
		block := make([]byte, blockSize)
		binary.BigEndian.PutUint32(block[0:4], 1) // type 1 = text
		binary.BigEndian.PutUint32(block[4:8], uint32(len(b)))
		copy(block[8:], b)
		out = append(out, block...)
	}
	return out
}

func TestMemoDBT(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memo.dbf")
	fields := []fieldSpec{{"NAME", 'C', 10, 0}, {"NOTE", 'M', 10, 0}}
	text := append(gbkEnc(t, "你好,备注字段"), 0x1A)
	writeFile(t, path, buildDBF(t, 0x83, 0x00, fields,
		[][]byte{rec(pad(gbkEnc(t, "alice"), 10), pad([]byte("1"), 10))}, 0))
	writeFile(t, filepath.Join(dir, "memo.dbt"), buildDBT(t, [][]byte{text}))

	tbl, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	recs, _ := tbl.Rows(true)
	if recs[0].Values[1] != "你好,备注字段" {
		t.Errorf("dbt memo = %q", recs[0].Values[1])
	}
	if recs[0].MemoErr != "" {
		t.Errorf("unexpected MemoErr: %s", recs[0].MemoErr)
	}
}

func TestMemoFPT(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memo.dbf")
	fields := []fieldSpec{{"NAME", 'C', 10, 0}, {"NOTE", 'M', 10, 0}}
	writeFile(t, path, buildDBF(t, 0xF5, 0x00, fields,
		[][]byte{rec(pad(gbkEnc(t, "bob"), 10), pad([]byte("1"), 10))}, 0))
	writeFile(t, filepath.Join(dir, "memo.fpt"), buildFPT(t, 512, [][]byte{gbkEnc(t, "FoxPro 备注")}))

	tbl, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	recs, _ := tbl.Rows(true)
	if recs[0].Values[1] != "FoxPro 备注" {
		t.Errorf("fpt memo = %q", recs[0].Values[1])
	}
}

func TestMemoVFP4BytePointer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vfp.dbf")
	fields := []fieldSpec{{"NOTE", 'M', 4, 0}}
	ptr := make([]byte, 4)
	binary.LittleEndian.PutUint32(ptr, 1)
	writeFile(t, path, buildDBF(t, 0x30, 0x00, fields, [][]byte{rec(ptr)}, 0))
	writeFile(t, filepath.Join(dir, "vfp.fpt"), buildFPT(t, 512, [][]byte{gbkEnc(t, "VFP 备注")}))

	tbl, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	recs, _ := tbl.Rows(true)
	if recs[0].Values[0] != "VFP 备注" {
		t.Errorf("vfp memo = %q", recs[0].Values[0])
	}
}

func TestMemoEmptyAndMissing(t *testing.T) {
	dir := t.TempDir()
	fields := []fieldSpec{{"NOTE", 'M', 10, 0}}

	// Empty memo pointer (all spaces) with no memo file: no error.
	path := filepath.Join(dir, "empty.dbf")
	writeFile(t, path, buildDBF(t, 0x83, 0x00, fields, [][]byte{rec(pad(nil, 10))}, 0))
	tbl, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	recs, _ := tbl.Rows(true)
	if recs[0].Values[0] != "" || recs[0].MemoErr != "" {
		t.Errorf("empty memo = %q err %q", recs[0].Values[0], recs[0].MemoErr)
	}

	// Referenced memo but no memo file: per-record error, not fatal.
	path2 := filepath.Join(dir, "missing.dbf")
	writeFile(t, path2, buildDBF(t, 0x83, 0x00, fields, [][]byte{rec(pad([]byte("3"), 10))}, 0))
	tbl2, err := Open(path2)
	if err != nil {
		t.Fatal(err)
	}
	recs2, _ := tbl2.Rows(true)
	if recs2[0].Values[0] != "" || !strings.Contains(recs2[0].MemoErr, "memo file") {
		t.Errorf("missing memo = %q err %q", recs2[0].Values[0], recs2[0].MemoErr)
	}
}

func TestMemoBadBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.dbf")
	fields := []fieldSpec{{"NOTE", 'M', 10, 0}}
	writeFile(t, path, buildDBF(t, 0xF5, 0x00, fields, [][]byte{rec(pad([]byte("99"), 10))}, 0))
	// A valid FPT with a picture block (type 0) at block 1.
	writeFile(t, filepath.Join(dir, "bad.fpt"), buildFPT(t, 512, [][]byte{}))

	tbl, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	recs, _ := tbl.Rows(true)
	if recs[0].MemoErr == "" {
		t.Errorf("expected MemoErr for unreachable block, got %q", recs[0].MemoErr)
	}
}

func TestMemoFileTypeDetection(t *testing.T) {
	// .dbt is preferred for dBASE versions, .fpt for FoxPro versions.
	m := &memoFile{version: 0x03}
	if m.isFPT() {
		t.Error("0x03 should use DBT")
	}
	m = &memoFile{version: 0xF5}
	if !m.isFPT() {
		t.Error("0xF5 should use FPT")
	}
	m = &memoFile{version: 0x30}
	if !m.isFPT() {
		t.Error("0x30 should use FPT")
	}

	// FPT block size from the header.
	fpt := buildFPT(t, 1024, [][]byte{})
	writeFile(t, filepath.Join(t.TempDir(), "x.fpt"), fpt)
	m = &memoFile{version: 0xF5, data: fpt}
	if m.blockSize() != 1024 {
		t.Errorf("block size = %d", m.blockSize())
	}
}
