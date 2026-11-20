package hashx

import (
	"strings"
	"testing"
)

func TestSumKnownVectors(t *testing.T) {
	tests := []struct {
		algo string
		in   string
		want string
	}{
		{"md5", "abc", "900150983cd24fb0d6963f7d28e17f72"},
		{"sha1", "abc", "a9993e364706816aba3e25717850c26c9cd0d89d"},
		{"sha256", "abc", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"sha512", "abc", "ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f"},
		{"crc32", "123456789", "cbf43926"},
		{"sm3", "", "1ab21d8355cfa17f8e61194831e81a8f22bec8c728fefb747ed035eb5082aa2b"},
		{"sm3", "abc", "66c7f0f462eeedd9d1f2d46bdc10e4e24167c4875cf2f7a2297da02b8f4ba8e0"},
	}
	for _, tt := range tests {
		t.Run(tt.algo+"/"+tt.in, func(t *testing.T) {
			got, err := Sum([]byte(tt.in), tt.algo, nil)
			if err != nil {
				t.Fatalf("Sum: %v", err)
			}
			if got != tt.want {
				t.Errorf("Sum = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSumReaderMatchesSum(t *testing.T) {
	data := strings.Repeat("lingbox-", 1000)
	for _, algo := range []string{"md5", "sha256", "crc32", "sm3"} {
		fromBytes, err := Sum([]byte(data), algo, nil)
		if err != nil {
			t.Fatal(err)
		}
		fromReader, err := SumReader(strings.NewReader(data), algo, nil)
		if err != nil {
			t.Fatal(err)
		}
		if fromBytes != fromReader {
			t.Errorf("%s: reader = %s, bytes = %s", algo, fromReader, fromBytes)
		}
	}
}

func TestHMAC(t *testing.T) {
	key := make([]byte, 20)
	for i := range key {
		key[i] = 0x0b
	}
	got, err := Sum([]byte("Hi There"), "hmac-sha256", key)
	if err != nil {
		t.Fatalf("Sum: %v", err)
	}
	const want = "b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7"
	if got != want {
		t.Errorf("HMAC-SHA256 = %s, want %s", got, want)
	}
}

func TestSumErrors(t *testing.T) {
	if _, err := Sum([]byte("x"), "nope", nil); err == nil {
		t.Error("unknown algorithm: expected error")
	}
	if _, err := Sum([]byte("x"), "hmac-sha256", nil); err == nil {
		t.Error("HMAC without a key: expected error")
	}
	if _, err := Sum([]byte("x"), "sha256", []byte("key")); err == nil {
		t.Error("key with non-HMAC algorithm: expected error")
	}
}

func TestAlgorithmsCoverAllPlainHashes(t *testing.T) {
	algs := Algorithms()
	want := map[string]bool{
		"md5": false, "sha1": false, "sha224": false, "sha256": false,
		"sha384": false, "sha512": false, "sm3": false, "crc32": false,
		"hmac-sha1": false, "hmac-sha256": false, "hmac-sha384": false,
		"hmac-sha512": false, "hmac-sm3": false,
	}
	for _, a := range algs {
		if _, ok := want[a]; ok {
			want[a] = true
		}
	}
	for a, seen := range want {
		if !seen {
			t.Errorf("Algorithms() missing %q: %v", a, algs)
		}
	}
}
