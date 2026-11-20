package hashx

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestSM3MultiBlockVector(t *testing.T) {
	got, err := Sum([]byte(strings.Repeat("abcd", 16)), "sm3", nil)
	if err != nil {
		t.Fatal(err)
	}
	const want = "debe9ff92275b8a138604889c18e5a4d6fdb70e5387e5765293dcba39c0c5732"
	if got != want {
		t.Errorf("SM3 = %s, want %s", got, want)
	}
}

func TestSM3ChunkedWrites(t *testing.T) {
	data := []byte(strings.Repeat("lingbox-sm3-", 37))
	want, err := Sum(data, "sm3", nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewSM3()
	for _, chunk := range [][]byte{data[:1], data[1:55], data[55:56], data[56:130], data[130:]} {
		if _, err := h.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		t.Errorf("chunked SM3 = %s, want %s", got, want)
	}
}
