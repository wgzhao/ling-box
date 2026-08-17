package plate

import (
	"strings"
	"testing"
)

func TestFindByShort(t *testing.T) {
	p, err := Find("湘")
	if err != nil {
		t.Fatalf("Find(湘): %v", err)
	}
	if p.Name != "湖南省" {
		t.Errorf("Find(湘).Name = %q, want 湖南省", p.Name)
	}
}

func TestFindByFullName(t *testing.T) {
	p, err := Find("湖南省")
	if err != nil {
		t.Fatalf("Find(湖南省): %v", err)
	}
	if p.Short != "湘" {
		t.Errorf("Find(湖南省).Short = %q, want 湘", p.Short)
	}
}

func TestFindBySuffixlessName(t *testing.T) {
	for _, q := range []string{"湖南", "北京", "内蒙古", "新疆"} {
		if _, err := Find(q); err != nil {
			t.Errorf("Find(%s): %v", q, err)
		}
	}
	// 新疆 must resolve to 新疆维吾尔自治区, not merely a partial match.
	p, err := Find("新疆")
	if err != nil {
		t.Fatalf("Find(新疆): %v", err)
	}
	if p.Name != "新疆维吾尔自治区" {
		t.Errorf("Find(新疆).Name = %q, want 新疆维吾尔自治区", p.Name)
	}
	// Whitespace is tolerated.
	if p, err := Find(" 北京 "); err != nil || p.Name != "北京市" {
		t.Errorf("Find( 北京 ) = %v, %v; want 北京市", p, err)
	}
}

func TestFindByEngName(t *testing.T) {
	p, err := Find("hunan")
	if err != nil {
		t.Fatalf("Find(hunan): %v", err)
	}
	if p.Name != "湖南省" {
		t.Errorf("Find(hunan).Name = %q, want 湖南省", p.Name)
	}
}

func TestFindUnknown(t *testing.T) {
	for _, q := range []string{"", "   ", "不存在的省"} {
		if _, err := Find(q); err == nil {
			t.Errorf("Find(%q): want error, got nil", q)
		}
	}
	// The error should list available provinces for guidance.
	_, err := Find("不存在的省")
	if err == nil || !strings.Contains(err.Error(), "北京市") {
		t.Errorf("unknown-province error should list available names, got: %v", err)
	}
}

func TestAll(t *testing.T) {
	all := All()
	if len(all) != 31 {
		t.Errorf("All() = %d provinces, want 31", len(all))
	}
	// Every province must have at least one code, and each code must be
	// prefixed by the province's abbreviation.
	for _, p := range all {
		if len(p.Codes) == 0 {
			t.Errorf("%s has no plate codes", p.Name)
		}
		for _, c := range p.Codes {
			if !strings.HasPrefix(c.Code, p.Short) {
				t.Errorf("%s code %s does not start with %s", p.Name, c.Code, p.Short)
			}
			if len(c.Districts) == 0 {
				t.Errorf("%s code %s has no districts", p.Name, c.Code)
			}
		}
	}
}

func TestNames(t *testing.T) {
	names := Names()
	if len(names) != 31 {
		t.Fatalf("Names() = %d names, want 31", len(names))
	}
	// Names should be unique.
	seen := make(map[string]bool, len(names))
	for _, n := range names {
		if seen[n] {
			t.Errorf("duplicate name %q", n)
		}
		seen[n] = true
	}
}
