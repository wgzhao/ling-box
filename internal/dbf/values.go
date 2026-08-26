package dbf

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// decodeValue decodes one fixed-width field of a record. The second
// return value is a per-record memo error, empty on success.
func (t *Table) decodeValue(f Field, raw []byte, resolveMemo bool) (string, string) {
	switch f.Type {
	case 'C':
		return t.dec(stripNUL(raw)), ""
	case 'N', 'F':
		return decodeNumeric(raw), ""
	case 'D':
		return decodeDate(raw), ""
	case 'L':
		return decodeLogical(raw), ""
	case 'I':
		return decodeInteger(raw), ""
	case 'T':
		return decodeDateTime(raw), ""
	case 'Y':
		return decodeCurrency(raw), ""
	case 'B':
		return decodeDouble(raw, f.Decimal), ""
	case 'M':
		if resolveMemo {
			return t.decodeMemo(f, raw)
		}
		return strings.TrimSpace(string(raw)), ""
	default:
		// G, P, and unknown types: surface the raw bytes, trimmed of
		// NUL padding, without decoding.
		return string(stripNUL(raw)), ""
	}
}

// JSONValue converts a decoded field value to a JSON-compatible
// value based on the field type: numbers for numeric fields, booleans
// for logical fields, and null for empty values.
func JSONValue(f Field, v string) any {
	switch f.Type {
	case 'L':
		if v == "true" {
			return true
		}
		if v == "false" {
			return false
		}
		return nil
	case 'N', 'F', 'B', 'Y':
		if v == "" {
			return nil
		}
		if fv, err := strconv.ParseFloat(v, 64); err == nil {
			return fv
		}
		return v
	case 'I':
		if v == "" {
			return nil
		}
		if iv, err := strconv.ParseInt(v, 10, 64); err == nil {
			return iv
		}
		return v
	default:
		if v == "" {
			return nil
		}
		return v
	}
}

// decodeNumeric reads an ASCII numeric field: right-justified, with
// possible '*' or '?' overflow sentinels.
func decodeNumeric(raw []byte) string {
	return strings.TrimSpace(string(raw))
}

// decodeDate reads an 8-byte YYYYMMDD field. Empty dates are stored as
// all spaces or all zeros.
func decodeDate(raw []byte) string {
	s := string(raw)
	if strings.TrimSpace(s) == "" || strings.Trim(s, "0") == "" {
		return ""
	}
	if len(s) < 8 {
		return s
	}
	if y, err := strconv.Atoi(s[0:4]); err == nil {
		if m, err := strconv.Atoi(s[4:6]); err == nil {
			if d, err := strconv.Atoi(s[6:8]); err == nil {
				return fmt.Sprintf("%04d-%02d-%02d", y, m, d)
			}
		}
	}
	return s
}

// decodeLogical reads a 1-byte logical field: T/t/Y/y, F/f/N/n, or
// '?'/space when unset.
func decodeLogical(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	switch raw[0] {
	case 'T', 't', 'Y', 'y':
		return "true"
	case 'F', 'f', 'N', 'n':
		return "false"
	default:
		return ""
	}
}

// decodeInteger reads a 4-byte little-endian int32 (VFP type I).
func decodeInteger(raw []byte) string {
	if len(raw) < 4 {
		return ""
	}
	return strconv.FormatInt(int64(int32(binary.LittleEndian.Uint32(raw))), 10)
}

// decodeDateTime reads an 8-byte VFP datetime: little-endian Julian
// day + milliseconds since midnight.
func decodeDateTime(raw []byte) string {
	if len(raw) < 8 {
		return ""
	}
	julian := binary.LittleEndian.Uint32(raw)
	ms := binary.LittleEndian.Uint32(raw[4:8])
	if julian == 0 && ms == 0 {
		return ""
	}
	// VFP's day 0 is 1858-11-17; the Unix epoch is 3506716800 seconds
	// later (40587 days).
	secs := int64(julian)*86400 + int64(ms)/1000 - 3506716800
	return time.Unix(secs, 0).UTC().Format("2006-01-02 15:04:05")
}

// decodeCurrency reads an 8-byte little-endian int64 scaled by 10000
// (VFP type Y).
func decodeCurrency(raw []byte) string {
	if len(raw) < 8 {
		return ""
	}
	v := int64(binary.LittleEndian.Uint64(raw))
	return strconv.FormatFloat(float64(v)/10000, 'f', 4, 64)
}

// decodeDouble reads an 8-byte little-endian float64 (VFP type B),
// formatted with the field's decimal count.
func decodeDouble(raw []byte, decimals int) string {
	if len(raw) < 8 {
		return ""
	}
	v := math.Float64frombits(binary.LittleEndian.Uint64(raw))
	return strconv.FormatFloat(v, 'f', decimals, 64)
}
