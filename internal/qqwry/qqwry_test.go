package qqwry

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net/netip"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// testEntry pairs a start IP with the index of its record in the record
// zone.
type testEntry struct {
	start uint32
	rec   int
}

// ipv4 converts a dotted IPv4 string to its big-endian uint32 value.
func ipv4(s string) uint32 {
	return binary.BigEndian.Uint32(netip.MustParseAddr(s).AsSlice())
}

// u32le encodes v as 4 little-endian bytes.
func u32le(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

// le24b encodes v as 3 little-endian bytes.
func le24b(v uint32) []byte {
	return []byte{byte(v), byte(v >> 8), byte(v >> 16)}
}

// gbkBytes encodes s in GBK.
func gbkBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, _, err := transform.Bytes(simplifiedchinese.GBK.NewEncoder(), []byte(s))
	if err != nil {
		t.Fatalf("gbk encode %q: %v", s, err)
	}
	return b
}

// directRecord builds a record holding a 4-byte end IP followed by the
// country and area strings with no redirect modes.
func directRecord(t *testing.T, end uint32, country, area string) []byte {
	t.Helper()
	var b bytes.Buffer
	b.Write(u32le(end))
	b.Write(gbkBytes(t, country))
	b.WriteByte(0)
	b.Write(gbkBytes(t, area))
	b.WriteByte(0)
	return b.Bytes()
}

// buildDat builds a modern-layout qqwry.dat from raw record bytes and
// index entries: an 8-byte header holding the index zone offsets, the
// record zone, then the index zone.
func buildDat(t *testing.T, records [][]byte, entries []testEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.Write(make([]byte, 8)) // header placeholder
	offs := make([]uint32, len(records))
	for i, r := range records {
		offs[i] = uint32(buf.Len())
		buf.Write(r)
	}
	indexOff := uint32(buf.Len())
	for _, e := range entries {
		var b [7]byte
		binary.LittleEndian.PutUint32(b[:4], e.start)
		copy(b[4:], le24b(offs[e.rec]))
		buf.Write(b[:])
	}
	data := buf.Bytes()
	binary.LittleEndian.PutUint32(data[0:4], indexOff)
	binary.LittleEndian.PutUint32(data[4:8], indexOff+uint32(len(entries)-1)*7)
	return data
}

// buildClassicDat builds a classic-layout qqwry.dat: an 8-byte header
// holding the first and last start IPs, the index zone, then the record
// zone. The start IPs must be spaced evenly so that the header-derived
// record count matches len(starts).
func buildClassicDat(t *testing.T, starts []uint32, records [][]byte) []byte {
	t.Helper()
	recBase := uint32(8 + 7*len(starts))
	var recs bytes.Buffer
	offs := make([]uint32, len(records))
	for i, r := range records {
		offs[i] = recBase + uint32(recs.Len())
		recs.Write(r)
	}
	var idx bytes.Buffer
	for i, s := range starts {
		var b [7]byte
		binary.LittleEndian.PutUint32(b[:4], s)
		copy(b[4:], le24b(offs[i]))
		idx.Write(b[:])
	}
	data := make([]byte, 8)
	data = append(data, idx.Bytes()...)
	data = append(data, recs.Bytes()...)
	binary.LittleEndian.PutUint32(data[0:4], starts[0])
	binary.LittleEndian.PutUint32(data[4:8], starts[len(starts)-1])
	return data
}

func mustOpen(t *testing.T, data []byte) *DB {
	t.Helper()
	db, err := openBytes(data)
	if err != nil {
		t.Fatalf("openBytes: %v", err)
	}
	return db
}

func TestOpenModern(t *testing.T) {
	data := buildDat(t, [][]byte{
		directRecord(t, ipv4("0.255.255.255"), "IANA", "保留地址"),
	}, []testEntry{{start: ipv4("0.0.0.1"), rec: 0}})
	db := mustOpen(t, data)
	if db.indexCount != 1 {
		t.Errorf("indexCount = %d, want 1", db.indexCount)
	}
	// record size: 4 (end IP) + 4 ("IANA") + 1 + 8 ("保留地址") + 1 = 18
	if db.indexOff != 8+18 {
		t.Errorf("indexOff = %d, want 26", db.indexOff)
	}
}

func TestQuery(t *testing.T) {
	data := buildDat(t, [][]byte{
		directRecord(t, ipv4("0.255.255.255"), "IANA", "保留地址"),
		directRecord(t, ipv4("8.8.8.255"), "美国", ""),
		directRecord(t, ipv4("114.114.114.255"), "江苏省南京市", "电信CZ88.NET"),
		directRecord(t, ipv4("1.2.3.255"), "中国", "0"),
	}, []testEntry{
		{start: ipv4("0.0.0.1"), rec: 0},
		{start: ipv4("1.2.3.0"), rec: 3},
		{start: ipv4("8.8.8.0"), rec: 1},
		{start: ipv4("114.114.114.0"), rec: 2},
	})
	db := mustOpen(t, data)
	tests := []struct {
		name    string
		ip      string
		want    string
		wantErr error
	}{
		{name: "first range start", ip: "0.0.0.1", want: "IANA 保留地址"},
		{name: "inside first range", ip: "0.1.2.3", want: "IANA 保留地址"},
		{name: "range start boundary", ip: "8.8.8.0", want: "美国"},
		{name: "range end boundary", ip: "8.8.8.255", want: "美国"},
		{name: "gbk decode and cz88 strip", ip: "114.114.114.114", want: "江苏省南京市 电信"},
		{name: "zero area dropped", ip: "1.2.3.4", want: "中国"},
		{name: "gap between ranges", ip: "9.9.9.9", wantErr: ErrNotFound},
		{name: "below first range", ip: "0.0.0.0", wantErr: ErrNotFound},
		{name: "above last range", ip: "255.255.255.255", wantErr: ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec, err := db.QueryStr(tc.ip)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("QueryStr(%q) error = %v, want %v", tc.ip, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("QueryStr(%q): %v", tc.ip, err)
			}
			if got := rec.Location(); got != tc.want {
				t.Errorf("QueryStr(%q).Location() = %q, want %q", tc.ip, got, tc.want)
			}
		})
	}
}

func TestQueryRecordFields(t *testing.T) {
	data := buildDat(t, [][]byte{
		directRecord(t, ipv4("8.8.8.255"), "美国", ""),
	}, []testEntry{{start: ipv4("8.8.8.0"), rec: 0}})
	db := mustOpen(t, data)
	rec, err := db.QueryStr("8.8.8.8")
	if err != nil {
		t.Fatalf("QueryStr: %v", err)
	}
	if rec.StartIP.String() != "8.8.8.0" || rec.EndIP.String() != "8.8.8.255" {
		t.Errorf("StartIP/EndIP = %s/%s, want 8.8.8.0/8.8.8.255", rec.StartIP, rec.EndIP)
	}
}

func TestQueryInvalid(t *testing.T) {
	data := buildDat(t, [][]byte{
		directRecord(t, ipv4("255.255.255.255"), "IANA", ""),
	}, []testEntry{{start: ipv4("0.0.0.1"), rec: 0}})
	db := mustOpen(t, data)
	for _, ip := range []string{"abc", "1.2.3", "1.2.3.4.5", "::1", "fe80::1"} {
		if _, err := db.QueryStr(ip); err == nil {
			t.Errorf("QueryStr(%q): expected error, got nil", ip)
		}
	}
}

func TestRedirectModes(t *testing.T) {
	// record zone contents (offsets assigned by buildDat):
	//  0: country string "美国"
	//  1: mode 0x02 record, area "谷歌" inline at +8
	//  2: mode 0x01 target: country "广东省深圳市", area "电信" inline
	//  3: mode 0x01 record pointing at record 2
	//  4: nested target: 0x02 + country pointer, area "微软" at +4
	//  5: mode 0x01 record pointing at record 4
	country0 := append(gbkBytes(t, "美国"), 0)
	mode1Target := append(gbkBytes(t, "广东省深圳市"), 0)
	mode1Target = append(mode1Target, gbkBytes(t, "电信")...)
	mode1Target = append(mode1Target, 0)
	rec2 := append(append(append(u32le(ipv4("255.255.255.255")), 0x02), le24b(8)...), append(gbkBytes(t, "谷歌"), 0)...)
	rec1 := append(append(u32le(ipv4("255.255.255.255")), 0x01), le24b(uint32(8+len(country0)+len(rec2)))...)
	nested := append([]byte{0x02}, le24b(8)...)
	nested = append(nested, gbkBytes(t, "微软")...)
	nested = append(nested, 0)
	rec0 := append(append(u32le(ipv4("255.255.255.255")), 0x01), le24b(uint32(8+len(country0)+len(rec2)+len(mode1Target)+len(rec1)))...)
	data := buildDat(t, [][]byte{country0, rec2, mode1Target, rec1, nested, rec0}, []testEntry{
		{start: ipv4("1.2.3.0"), rec: 5},
		{start: ipv4("8.8.8.0"), rec: 1},
		{start: ipv4("114.114.114.0"), rec: 3},
	})
	db := mustOpen(t, data)
	tests := []struct {
		name string
		ip   string
		want string
	}{
		{name: "mode 0x02", ip: "8.8.8.8", want: "美国 谷歌"},
		{name: "mode 0x01", ip: "114.114.114.114", want: "广东省深圳市 电信"},
		{name: "nested 0x01 to 0x02", ip: "1.2.3.4", want: "美国 微软"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec, err := db.QueryStr(tc.ip)
			if err != nil {
				t.Fatalf("QueryStr(%q): %v", tc.ip, err)
			}
			if got := rec.Location(); got != tc.want {
				t.Errorf("QueryStr(%q).Location() = %q, want %q", tc.ip, got, tc.want)
			}
		})
	}
}

func TestOpenClassic(t *testing.T) {
	starts := []uint32{ipv4("8.8.8.8"), ipv4("8.8.8.15"), ipv4("8.8.8.22")}
	data := buildClassicDat(t, starts, [][]byte{
		directRecord(t, ipv4("8.8.8.14"), "美国", "Level3"),
		directRecord(t, ipv4("8.8.8.21"), "美国", ""),
		directRecord(t, ipv4("255.255.255.255"), "IANA", ""),
	})
	db := mustOpen(t, data)
	tests := []struct {
		ip   string
		want string
	}{
		{"8.8.8.10", "美国 Level3"},
		{"8.8.8.16", "美国"},
		{"8.8.8.100", "IANA"},
		{"8.8.8.7", ""}, // below the first start IP
	}
	for _, tc := range tests {
		rec, err := db.QueryStr(tc.ip)
		if tc.want == "" {
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("QueryStr(%q) error = %v, want ErrNotFound", tc.ip, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("QueryStr(%q): %v", tc.ip, err)
			continue
		}
		if got := rec.Location(); got != tc.want {
			t.Errorf("QueryStr(%q).Location() = %q, want %q", tc.ip, got, tc.want)
		}
	}
}

func TestOpenErrors(t *testing.T) {
	// unsorted index zone
	unsorted := buildDat(t, [][]byte{
		directRecord(t, ipv4("255.255.255.255"), "A", ""),
		directRecord(t, ipv4("255.255.255.255"), "B", ""),
	}, []testEntry{
		{start: ipv4("10.0.0.0"), rec: 0},
		{start: ipv4("1.0.0.0"), rec: 1},
	})
	// record offset pointing into the header
	badOff := make([]byte, 8+11+7) // 8 header + 11-byte record + 7-byte index
	binary.LittleEndian.PutUint32(badOff[0:4], 19)
	binary.LittleEndian.PutUint32(badOff[4:8], 19)
	copy(badOff[19:26], append(u32le(ipv4("1.0.0.0")), le24b(0)...))
	tests := []struct {
		name string
		data []byte
	}{
		{"too short", []byte{1, 2, 3, 4, 5}},
		{"garbage header", make([]byte, 64)},
		{"unsorted index", unsorted},
		{"record offset out of bounds", badOff},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := openBytes(tc.data); err == nil {
				t.Error("openBytes: expected error, got nil")
			}
		})
	}
}
