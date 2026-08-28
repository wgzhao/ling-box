// Package qqwry reads qqwry.dat IP geolocation database files (纯真 IP 库).
//
// Two layouts are supported:
//
//   - The modern layout rebuilt from CZDB (as distributed by
//     https://github.com/metowolf/qqwry.dat): an 8-byte header holding the
//     file offsets of the first and last index entries, then the record
//     zone, then the index zone (7 bytes per entry) at the end of the file.
//   - The classic layout: an 8-byte header holding the first and last start
//     IPs, immediately followed by the index zone, then the record zone.
//
// In both layouts every index entry pairs a start IP (4 bytes, little
// endian) with the file offset (3 bytes) of the record that covers the
// range. A record is a 4-byte end IP followed by the country and area
// strings, either inline or through 0x01/0x02 redirect markers. All
// strings are GBK-encoded and null-terminated.
package qqwry

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const (
	indexEntryLen = 7
	redirectMode1 = 0x01
	redirectMode2 = 0x02
)

// ErrNotFound reports that an IP falls outside every range in the database.
var ErrNotFound = errors.New("ip not found in database")

// DB is a loaded qqwry.dat database.
type DB struct {
	data       []byte
	indexOff   uint32 // file offset of the first index entry
	indexCount int    // number of index entries
	stringsEnd uint32 // records must fit before this offset
}

// Record is the geolocation result for one IPv4 address.
type Record struct {
	StartIP netip.Addr
	EndIP   netip.Addr
	Country string
	Area    string
}

// Location joins Country and Area, dropping empty parts.
func (r Record) Location() string {
	parts := make([]string, 0, 2)
	for _, p := range []string{r.Country, r.Area} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, " ")
}

// Open reads a qqwry.dat file and returns a queryable database.
func Open(path string) (*DB, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return openBytes(data)
}

func openBytes(data []byte) (*DB, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("file too short to be a qqwry.dat file (%d bytes)", len(data))
	}
	first := binary.LittleEndian.Uint32(data[0:4])
	second := binary.LittleEndian.Uint32(data[4:8])
	db := &DB{data: data}
	switch {
	case isClassic(data, first, second):
		db.indexOff = 8
		db.indexCount = int((second-first)/7) + 1
		db.stringsEnd = uint32(len(data))
	case isModern(data, first):
		db.indexOff = first
		db.indexCount = (len(data) - int(first)) / 7
		db.stringsEnd = first
	default:
		return nil, fmt.Errorf("not a qqwry.dat file (unrecognized header)")
	}
	if db.indexCount == 0 {
		return nil, fmt.Errorf("qqwry.dat file has an empty index zone")
	}
	if err := db.validateIndex(); err != nil {
		return nil, err
	}
	return db, nil
}

// isClassic reports whether data has the classic layout: the header holds
// the first and last start IPs, and the first index entry (right after the
// header) starts at exactly the header's first IP.
func isClassic(data []byte, first, second uint32) bool {
	if second <= first || (second-first)%7 != 0 || len(data) < 12 {
		return false
	}
	count := (second-first)/7 + 1
	indexEnd := 8 + count*7
	return indexEnd <= uint32(len(data)) && binary.LittleEndian.Uint32(data[8:12]) == first
}

// isModern reports whether data has the modern CZDB-rebuilt layout: the
// header holds the file offset of the first index entry, and the index
// zone runs from there to the end of the file.
func isModern(data []byte, first uint32) bool {
	return first >= 8 && int(first) < len(data) && (len(data)-int(first))%7 == 0
}

// validateIndex sanity-checks the index zone: entries must be sorted by
// start IP and their record offsets must point into the record zone.
func (db *DB) validateIndex() error {
	firstStart, firstOff := db.entry(0)
	lastStart, lastOff := db.entry(db.indexCount - 1)
	if firstStart > lastStart {
		return fmt.Errorf("index zone is not sorted by start ip")
	}
	for _, off := range [2]uint32{firstOff, lastOff} {
		if off < 8 || off+4 >= db.stringsEnd {
			return fmt.Errorf("record offset 0x%x out of bounds", off)
		}
	}
	return nil
}

// entry returns the start IP and record offset of the i-th index entry.
func (db *DB) entry(i int) (startIP, recordOff uint32) {
	pos := int(db.indexOff) + i*indexEntryLen
	buf := db.data[pos : pos+indexEntryLen]
	return binary.LittleEndian.Uint32(buf[0:4]), le24(buf[4:7])
}

// Query returns the record covering ip, or ErrNotFound if ip falls
// outside every range in the database.
func (db *DB) Query(ip netip.Addr) (Record, error) {
	if !ip.Is4() {
		return Record{}, fmt.Errorf("ip %s is not an IPv4 address (qqwry.dat is IPv4-only)", ip)
	}
	return db.query(binary.BigEndian.Uint32(ip.AsSlice()))
}

// QueryStr parses s as an IPv4 address and queries the database.
func (db *DB) QueryStr(s string) (Record, error) {
	ip, err := netip.ParseAddr(s)
	if err != nil {
		return Record{}, fmt.Errorf("invalid ip address %q", s)
	}
	return db.Query(ip)
}

func (db *DB) query(ip uint32) (Record, error) {
	i := db.search(ip)
	if i < 0 {
		return Record{}, ErrNotFound
	}
	start, off := db.entry(i)
	endIP, country, area, err := db.recordAt(off)
	if err != nil {
		return Record{}, err
	}
	if ip > endIP {
		return Record{}, ErrNotFound
	}
	return Record{
		StartIP: uint32ToAddr(start),
		EndIP:   uint32ToAddr(endIP),
		Country: country,
		Area:    area,
	}, nil
}

// search returns the index of the entry with the greatest start IP that is
// not greater than ip, or -1 if every start IP is greater.
func (db *DB) search(ip uint32) int {
	lo, hi := 0, db.indexCount-1
	if start, _ := db.entry(lo); start > ip {
		return -1
	}
	// invariant: entry(lo).start <= ip
	for lo < hi {
		mid := lo + (hi-lo+1)/2
		if start, _ := db.entry(mid); start <= ip {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// recordAt parses the record at off: a 4-byte end IP followed by the
// country and area strings, either inline or through 0x01/0x02 redirects.
func (db *DB) recordAt(off uint32) (endIP uint32, country, area string, err error) {
	if off+4 > db.stringsEnd {
		return 0, "", "", fmt.Errorf("record offset 0x%x out of bounds", off)
	}
	endIP = binary.LittleEndian.Uint32(db.data[off : off+4])
	country, area, err = db.recordStrings(off + 4)
	return
}

// recordStrings parses the country and area strings that follow the record
// header byte at off, following redirect markers as defined by the qqwry
// format.
func (db *DB) recordStrings(off uint32) (country, area string, err error) {
	if off+4 > db.stringsEnd {
		return "", "", fmt.Errorf("record header at 0x%x out of bounds", off)
	}
	mode := db.data[off]
	switch mode {
	case redirectMode1:
		// off+1: 3-byte pointer to the country field
		posC := le24(db.data[off+1 : off+4])
		if posC >= db.stringsEnd {
			return "", "", fmt.Errorf("redirect at 0x%x points out of bounds (0x%x)", off, posC)
		}
		if db.data[posC] == redirectMode2 {
			// nested redirect: posC holds a 0x02 marker followed by the
			// country pointer; the area follows the nested pointer
			if posC+4 > db.stringsEnd {
				return "", "", fmt.Errorf("nested redirect at 0x%x out of bounds", posC)
			}
			posA := le24(db.data[posC+1 : posC+4])
			country, _, err = db.cstr(posA)
			if err != nil {
				return "", "", err
			}
			area, err = db.maybeRedirect(posC + 4)
			return country, area, err
		}
		country, next, err := db.cstr(posC)
		if err != nil {
			return "", "", err
		}
		area, err = db.maybeRedirect(next)
		return country, area, err
	case redirectMode2:
		// off+1: 3-byte pointer to the country field; the area field
		// sits at off+4, right after the marker and pointer
		posA := le24(db.data[off+1 : off+4])
		country, _, err = db.cstr(posA)
		if err != nil {
			return "", "", err
		}
		area, err = db.maybeRedirect(off + 4)
		return country, area, err
	default:
		// the byte at off is the first byte of the country string itself
		country, next, err := db.cstr(off)
		if err != nil {
			return "", "", err
		}
		area, err = db.maybeRedirect(next)
		return country, area, err
	}
}

// maybeRedirect reads the area string at pos, following a 0x01/0x02
// redirect if present. Records without an area field yield an empty string.
func (db *DB) maybeRedirect(pos uint32) (string, error) {
	if pos >= db.stringsEnd {
		return "", nil
	}
	if m := db.data[pos]; m == redirectMode1 || m == redirectMode2 {
		if pos+4 > db.stringsEnd {
			return "", fmt.Errorf("area field at 0x%x out of bounds", pos)
		}
		pos = le24(db.data[pos+1 : pos+4])
	}
	s, _, err := db.cstr(pos)
	return s, err
}

// cstr reads a null-terminated GBK string starting at off, returning the
// decoded string and the offset just past the terminator.
func (db *DB) cstr(off uint32) (string, uint32, error) {
	if off >= db.stringsEnd {
		return "", off, fmt.Errorf("string offset 0x%x out of bounds", off)
	}
	for i := off; i < db.stringsEnd; i++ {
		if db.data[i] == 0 {
			return clean(decodeGBK(db.data[off:i])), i + 1, nil
		}
	}
	return "", db.stringsEnd, fmt.Errorf("unterminated string at offset 0x%x", off)
}

// clean trims space, drops the CZ88.NET banner, and normalizes the "0"
// placeholder used for missing fields.
func clean(s string) string {
	s = strings.TrimSpace(strings.TrimSuffix(s, "CZ88.NET"))
	if s == "0" {
		return ""
	}
	return s
}

// decodeGBK decodes a GBK-encoded byte slice, ignoring invalid sequences.
func decodeGBK(b []byte) string {
	s, _, _ := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), b)
	return string(s)
}

// le24 decodes a little-endian 3-byte offset.
func le24(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16
}

// uint32ToAddr converts a big-endian IPv4 value to a netip address.
func uint32ToAddr(v uint32) netip.Addr {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	return netip.AddrFrom4(b)
}
