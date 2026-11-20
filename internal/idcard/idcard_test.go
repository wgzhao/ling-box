package idcard

import (
	"strings"
	"testing"
	"time"
)

func fixedNow() time.Time {
	return time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
}

func validID(t *testing.T, first17 string) string {
	t.Helper()
	check, err := checkDigit(first17)
	if err != nil {
		t.Fatalf("checkDigit(%q): %v", first17, err)
	}
	return first17 + string(check)
}

func TestValidate18(t *testing.T) {
	info := Validate("11010519491231002X", Options{Now: fixedNow()})
	if !info.Valid {
		t.Fatalf("expected valid: %+v", info)
	}
	if info.Format != "18-digit" || info.Number != "11010519491231002X" {
		t.Errorf("format/number = %q/%q", info.Format, info.Number)
	}
	if info.RegionCode != "110105" || !info.RegionKnown || info.RegionName != "北京市朝阳区" {
		t.Errorf("region = %+v", info)
	}
	if info.Province != "北京市" || info.City != "市辖区" || info.Area != "朝阳区" {
		t.Errorf("region parts = %q/%q/%q", info.Province, info.City, info.Area)
	}
	if info.BirthDate != "1949-12-31" || info.Gender != "female" || info.Sequence != "002" {
		t.Errorf("birth/gender/sequence = %q/%q/%q", info.BirthDate, info.Gender, info.Sequence)
	}
	if info.CheckDigit != "X" || info.ExpectedCheckDigit != "X" {
		t.Errorf("checksum = %q/%q", info.CheckDigit, info.ExpectedCheckDigit)
	}
}

func TestValidateMaleLowercaseAndSpaces(t *testing.T) {
	info := Validate(" 110105194912310038 ", Options{Now: fixedNow()})
	if !info.Valid {
		t.Fatalf("expected valid: %+v", info)
	}
	if info.Gender != "male" {
		t.Errorf("gender = %q", info.Gender)
	}

	lower := Validate("11010519491231002x", Options{Now: fixedNow()})
	if !lower.Valid || lower.CheckDigit != "X" || lower.Number != "11010519491231002X" {
		t.Errorf("lowercase x = %+v", lower)
	}
}

func TestValidate15(t *testing.T) {
	info := Validate("110105491231002", Options{Now: fixedNow()})
	if !info.Valid {
		t.Fatalf("expected valid: %+v", info)
	}
	if info.Format != "15-digit" || info.Number != "11010519491231002X" {
		t.Errorf("format/number = %q/%q", info.Format, info.Number)
	}
	if info.BirthDate != "1949-12-31" || info.Gender != "female" {
		t.Errorf("birth/gender = %q/%q", info.BirthDate, info.Gender)
	}
}

func TestValidateErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"length", "123", "15 or 18"},
		{"checksum", "110105194912310020", "checksum"},
		{"bad date", "11010520250230002X", "birth date"},
		{"future", "11010520990101002X", "future"},
		{"check digit", "11010519491231002A", "check digit"},
		{"15 bad date", "110105491332002", "birth date"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := Validate(tt.input, Options{Now: fixedNow()})
			if info.Valid {
				t.Fatalf("expected invalid: %+v", info)
			}
			if !strings.Contains(strings.Join(info.Errors, " "), tt.want) {
				t.Errorf("errors = %v, want %q", info.Errors, tt.want)
			}
		})
	}
}

func TestUnknownRegion(t *testing.T) {
	id := validID(t, "99999919491231002")
	info := Validate(id, Options{Now: fixedNow()})
	if !info.Valid || info.RegionKnown {
		t.Fatalf("expected valid with warning: %+v", info)
	}
	if len(info.Warnings) == 0 || !strings.Contains(strings.Join(info.Warnings, " "), "unknown region") {
		t.Errorf("warnings = %v", info.Warnings)
	}

	strict := Validate(id, Options{Now: fixedNow(), StrictRegion: true})
	if strict.Valid {
		t.Errorf("strict validation accepted unknown region: %+v", strict)
	}
}

func TestGenerate(t *testing.T) {
	birth := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, gender := range []string{"male", "female", "any"} {
		gens, err := Generate(GenerateOptions{
			Region: "110105",
			Birth:  birth,
			Gender: gender,
			Count:  5,
			Now:    fixedNow(),
		})
		if err != nil {
			t.Fatalf("%s: %v", gender, err)
		}
		if len(gens) != 5 {
			t.Fatalf("%s: got %d", gender, len(gens))
		}
		for _, g := range gens {
			if len(g.Number) != 18 {
				t.Errorf("%s: number = %q", gender, g.Number)
			}
			if g.RegionName != "北京市朝阳区" || g.BirthDate != "1990-01-01" {
				t.Errorf("%s: %+v", gender, g)
			}
			if gender != "any" && g.Gender != gender {
				t.Errorf("%s: gender = %q", gender, g.Gender)
			}
			if gender == "male" && g.Gender != "male" || gender == "female" && g.Gender != "female" {
				t.Errorf("%s: generated gender = %q", gender, g.Gender)
			}

			info := Validate(g.Number, Options{Now: fixedNow()})
			if !info.Valid {
				t.Errorf("generated %q is invalid: %+v", g.Number, info)
			}
		}
	}
}

func TestGenerateErrors(t *testing.T) {
	birth := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []GenerateOptions{
		{Region: "999999", Birth: birth, Count: 1, Now: fixedNow()},
		{Region: "110105", Birth: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC), Count: 1, Now: fixedNow()},
		{Region: "110105", Birth: birth, Gender: "robot", Count: 1, Now: fixedNow()},
		{Region: "110105", Birth: birth, Count: -1, Now: fixedNow()},
		{Region: "110105", Birth: birth, Count: 1001, Now: fixedNow()},
	}
	for _, opts := range tests {
		if _, err := Generate(opts); err == nil {
			t.Errorf("Generate(%+v): expected error", opts)
		}
	}
}

func TestRegionTable(t *testing.T) {
	if len(regions) < 2900 {
		t.Errorf("region table has %d entries, want at least 2900", len(regions))
	}
	if r, ok := LookupRegion("130102"); !ok || r.Name() != "河北省石家庄市长安区" {
		t.Errorf("130102 = %+v, %v", r, ok)
	}
	if r, ok := LookupRegion("110105"); !ok || r.Name() != "北京市朝阳区" {
		t.Errorf("110105 = %+v, %v", r, ok)
	}
	if _, ok := LookupRegion("999999"); ok {
		t.Error("999999 unexpectedly found")
	}
}
