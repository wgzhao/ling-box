// Package plate provides Chinese license plate region lookup data
// and matching logic.
package plate

import (
	"fmt"
	"strings"
)

// Code is one license plate prefix (e.g. 湘A) and the districts it
// belongs to.
type Code struct {
	Code      string
	Districts []string
}

// Province is a provincial-level division with its plate codes.
type Province struct {
	Name    string // full name, e.g. 湖南省
	Short   string // one-character abbreviation, e.g. 湘
	EngName string // pinyin, e.g. hunan
	Codes   []Code
}

// administrativeSuffixes are stripped from a query before matching, so
// 湖南, 湖南省, and 湘 all resolve to 湖南省.
var administrativeSuffixes = []string{"特别行政区", "维吾尔自治区", "壮族自治区", "回族自治区", "自治区", "省", "市"}

// Find resolves a query to a province. The query may be the one-character
// abbreviation (湘), the full name (湖南省), or the name without its
// administrative suffix (湖南).
func Find(query string) (*Province, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty province name")
	}
	for i := range provinces {
		p := &provinces[i]
		if p.Short == query || p.Name == query || p.EngName == query {
			return p, nil
		}
	}
	for i := range provinces {
		p := &provinces[i]
		if stripSuffix(p.Name) == stripSuffix(query) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("unknown province %q; available: %s", query, strings.Join(Names(), ", "))
}

// stripSuffix removes any administrative suffix (省/市/自治区) from name.
func stripSuffix(name string) string {
	for _, s := range administrativeSuffixes {
		if strings.HasSuffix(name, s) {
			return strings.TrimSuffix(name, s)
		}
	}
	return name
}

// All returns all provinces in the source data order.
func All() []*Province {
	out := make([]*Province, len(provinces))
	for i := range provinces {
		out[i] = &provinces[i]
	}
	return out
}

// Names returns the display names of all provinces, for error messages
// and listings.
func Names() []string {
	out := make([]string, len(provinces))
	for i, p := range provinces {
		out[i] = p.Name
	}
	return out
}
