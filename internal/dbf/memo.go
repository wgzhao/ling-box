package dbf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

// memoFile provides access to a memo file, either .fpt (FoxPro 2.x /
// Visual FoxPro) or .dbt (dBASE III/IV). Memo problems never abort the
// table; they surface as per-record MemoErr.
type memoFile struct {
	version byte
	data    []byte
	block   int // block size in bytes (FPT only)
}

// openMemo looks for a companion .fpt or .dbt file next to the DBF and
// returns nil when neither exists or cannot be read.
func openMemo(dbPath string, version byte) (string, *memoFile) {
	base := baseName(dbPath)
	for _, ext := range []string{".fpt", ".FPT", ".dbt", ".DBT"} {
		path := base + ext
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return path, &memoFile{version: version, data: data}
	}
	return "", nil
}

// blockSize returns the memo block size (FPT reads it from the header;
// DBT is always 512).
func (m *memoFile) blockSize() int {
	if m.block > 0 {
		return m.block
	}
	if m.data != nil && (m.version == 0xF5 || m.version == 0x30 || m.version == 0x31 || m.version == 0x32) {
		if len(m.data) >= 8 {
			// FPT header: block size is big-endian at bytes 6-7.
			if bs := int(binary.BigEndian.Uint16(m.data[6:8])); bs > 0 {
				m.block = bs
				return bs
			}
		}
	}
	return 512
}

// isFPT reports whether the memo file is FoxPro .fpt (blocked with
// headers) rather than dBASE .dbt.
func (m *memoFile) isFPT() bool {
	switch m.version {
	case 0xF5, 0x30, 0x31, 0x32:
		return true
	default:
		return false
	}
}

// memoBlockNo extracts the block number from a memo field's bytes.
// Visual FoxPro 3+ uses a 4-byte little-endian pointer; FoxPro 2.x and
// dBASE III use a 10-byte field holding either ASCII digits (dBT) or a
// binary pointer (FPT). Returns 0 when the memo is empty.
func memoBlockNo(f Field, raw []byte) int {
	if f.Length <= 4 {
		if len(raw) >= 4 {
			return int(binary.LittleEndian.Uint32(raw))
		}
		return 0
	}
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return 0
	}
	// ASCII decimal block number (dBASE III .dbt).
	if isDigits(s) {
		n := 0
		for _, c := range s {
			n = n*10 + int(c-'0')
		}
		return n
	}
	// Binary pointer (FoxPro 2.x .fpt).
	if len(raw) >= 4 {
		return int(binary.LittleEndian.Uint32(raw))
	}
	return 0
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// decodeMemo resolves a memo field to its text. The second return is
// a non-fatal error message when the memo file is missing or the block
// cannot be read. An empty memo pointer needs no memo file at all.
func (t *Table) decodeMemo(f Field, raw []byte) (string, string) {
	n := memoBlockNo(f, raw)
	if n <= 0 {
		return "", ""
	}
	if t.memo == nil {
		return "", "memo file (.fpt/.dbt) not found"
	}
	// FoxPro commonly appends a NUL terminator to memo text; strip it.
	trimmed := func(b []byte) []byte { return bytes.TrimRight(b, "\x00") }
	if t.memo.isFPT() {
		text, err := t.memo.readFPT(n)
		if err != nil {
			return "", fmt.Sprintf("memo block %d: %v", n, err)
		}
		return t.dec(trimmed(text)), ""
	}
	text, err := t.memo.readDBT(n)
	if err != nil {
		return "", fmt.Sprintf("memo block %d: %v", n, err)
	}
	return t.dec(trimmed(text)), ""
}
