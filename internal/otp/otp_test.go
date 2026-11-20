package otp

import (
	"testing"
	"time"
)

const rfcSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ" // "12345678901234567890"

func TestHOTPRFC4226Vectors(t *testing.T) {
	want := []string{
		"755224", "287082", "359152", "969429", "338314",
		"254676", "287922", "162583", "399871", "520489",
	}
	for counter, expected := range want {
		got, err := HOTP(rfcSecret, uint64(counter), Config{})
		if err != nil {
			t.Fatalf("counter %d: %v", counter, err)
		}
		if got != expected {
			t.Errorf("HOTP(%d) = %s, want %s", counter, got, expected)
		}
	}
}

func TestTOTPRFC6238Vectors(t *testing.T) {
	tests := []struct {
		unix int64
		want string
	}{
		{59, "94287082"},
		{1111111109, "07081804"},
		{1111111111, "14050471"},
		{1234567890, "89005924"},
		{2000000000, "69279037"},
		{20000000000, "65353130"},
	}
	for _, tt := range tests {
		got, err := TOTP(rfcSecret, time.Unix(tt.unix, 0), Config{Digits: 8})
		if err != nil {
			t.Fatalf("t=%d: %v", tt.unix, err)
		}
		if got.Code != tt.want {
			t.Errorf("TOTP(%d) = %s, want %s", tt.unix, got.Code, tt.want)
		}
	}
}

func TestTOTPRemaining(t *testing.T) {
	tests := []struct {
		unix int64
		want int
	}{
		{0, 30},
		{1, 29},
		{29, 1},
		{30, 30},
		{59, 1},
	}
	for _, tt := range tests {
		got, err := TOTP(rfcSecret, time.Unix(tt.unix, 0), Config{})
		if err != nil {
			t.Fatal(err)
		}
		if got.Remaining != tt.want {
			t.Errorf("t=%d remaining = %d, want %d", tt.unix, got.Remaining, tt.want)
		}
		if got.Counter != uint64(tt.unix/30) {
			t.Errorf("t=%d counter = %d", tt.unix, got.Counter)
		}
	}
}

func TestSecretFormatting(t *testing.T) {
	want := "94287082"
	variants := []string{
		"gezdgnbvgy3tqojqgezdgnbvgy3tqojq",
		"GEZD GNBV GY3T QOJQ GEZD GNBV GY3T QOJQ",
		"GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ====",
	}
	for _, secret := range variants {
		got, err := TOTP(secret, time.Unix(59, 0), Config{Digits: 8})
		if err != nil {
			t.Fatalf("%q: %v", secret, err)
		}
		if got.Code != want {
			t.Errorf("%q: code = %s, want %s", secret, got.Code, want)
		}
	}
}

func TestParseURI(t *testing.T) {
	u, err := ParseURI("otpauth://totp/ACME%20Co:john@example.com?secret=JBSWY3DPEHPK3PXP&issuer=ACME%20Co&algorithm=SHA1&digits=6&period=30")
	if err != nil {
		t.Fatalf("ParseURI: %v", err)
	}
	if u.Type != "totp" || u.Secret != "JBSWY3DPEHPK3PXP" || u.Digits != 6 || u.Period != 30 || u.Algorithm != "SHA1" {
		t.Errorf("uri = %+v", u)
	}
	if u.Label != "ACME Co:john@example.com" || u.Issuer != "ACME Co" {
		t.Errorf("label/issuer = %q / %q", u.Label, u.Issuer)
	}

	h, err := ParseURI("otpauth://hotp/ACME:alice?secret=JBSWY3DPEHPK3PXP&counter=5")
	if err != nil {
		t.Fatalf("ParseURI hotp: %v", err)
	}
	if h.Type != "hotp" || h.Counter != 5 {
		t.Errorf("hotp uri = %+v", h)
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		cfg    Config
	}{
		{"empty", "", Config{}},
		{"bad base32", "!!!!", Config{}},
		{"digits", rfcSecret, Config{Digits: 99}},
		{"period", rfcSecret, Config{Period: -1}},
		{"algorithm", rfcSecret, Config{Algorithm: "MD5"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := TOTP(tt.secret, time.Unix(0, 0), tt.cfg); err == nil {
				t.Error("expected error")
			}
		})
	}

	uris := []string{
		"https://example.com",
		"otpauth://totp/x",
		"otpauth://totp/x?secret=!!!!",
		"otpauth://hotp/x?secret=JBSWY3DPEHPK3PXP",
		"otpauth://totp/x?secret=JBSWY3DPEHPK3PXP&digits=99",
		"otpauth://totp/x?secret=JBSWY3DPEHPK3PXP&period=0",
	}
	for _, raw := range uris {
		if _, err := ParseURI(raw); err == nil {
			t.Errorf("ParseURI(%q): expected error", raw)
		}
	}
}
