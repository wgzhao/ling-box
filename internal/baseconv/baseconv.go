package baseconv

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Base represents a number base.
type Base int

const (
	Bin Base = 2
	Oct Base = 8
	Dec Base = 10
	Hex Base = 16
)

// Result holds all representations of a single number.
type Result struct {
	Input    string `json:"input"`
	FromBase Base   `json:"from_base"`
	Binary   string `json:"binary"`
	Octal    string `json:"octal"`
	Decimal  string `json:"decimal"`
	Hex      string `json:"hex"`
}

// validBases maps string names to their base values.
var validBases = map[string]Base{
	"bin": Bin,
	"oct": Oct,
	"dec": Dec,
	"hex": Hex,
}

// ParseBase parses a base name into a Base value.
func ParseBase(s string) (Base, error) {
	s = strings.ToLower(s)
	if b, ok := validBases[s]; ok {
		return b, nil
	}
	return 0, fmt.Errorf("unsupported base: %q (supported: bin, oct, dec, hex)", s)
}

// Convert converts a number from the given base to all other bases.
func Convert(input string, fromBase Base) (*Result, error) {
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Try signed int64 first; if it overflows, fall back to uint64.
	value, err := tryParseInt(input, fromBase)
	if err == nil {
		return &Result{
			Input:    input,
			FromBase: fromBase,
			Binary:   strconv.FormatInt(value, 2),
			Octal:    strconv.FormatInt(value, 8),
			Decimal:  strconv.FormatInt(value, 10),
			Hex:      strings.ToUpper(strconv.FormatInt(value, 16)),
		}, nil
	}

	// Fall back to unsigned
	uvalue, uerr := tryParseUint(input, fromBase)
	if uerr != nil {
		return nil, fmt.Errorf("invalid %s value %q: %w", baseName(fromBase), input, err)
	}

	return &Result{
		Input:    input,
		FromBase: fromBase,
		Binary:   strconv.FormatUint(uvalue, 2),
		Octal:    strconv.FormatUint(uvalue, 8),
		Decimal:  strconv.FormatUint(uvalue, 10),
		Hex:      strings.ToUpper(strconv.FormatUint(uvalue, 16)),
	}, nil
}

func tryParseInt(input string, fromBase Base) (int64, error) {
	clean := input
	if fromBase == Hex {
		clean = strings.TrimPrefix(input, "0x")
		clean = strings.TrimPrefix(clean, "0X")
	}
	switch fromBase {
	case Bin:
		return strconv.ParseInt(clean, 2, 64)
	case Oct:
		return strconv.ParseInt(clean, 8, 64)
	case Dec:
		return strconv.ParseInt(clean, 10, 64)
	case Hex:
		return strconv.ParseInt(clean, 16, 64)
	default:
		return 0, fmt.Errorf("unsupported base: %d", fromBase)
	}
}

func tryParseUint(input string, fromBase Base) (uint64, error) {
	clean := input
	if fromBase == Hex {
		clean = strings.TrimPrefix(input, "0x")
		clean = strings.TrimPrefix(clean, "0X")
	}
	switch fromBase {
	case Bin:
		return strconv.ParseUint(clean, 2, 64)
	case Oct:
		return strconv.ParseUint(clean, 8, 64)
	case Dec:
		return strconv.ParseUint(clean, 10, 64)
	case Hex:
		return strconv.ParseUint(clean, 16, 64)
	default:
		return 0, fmt.Errorf("unsupported base: %d", fromBase)
	}
}

// ConvertUint64 handles unsigned 64-bit integers (for very large values).
func ConvertUint64(input string, fromBase Base) (*Result, error) {
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	var value uint64
	var err error

	switch fromBase {
	case Bin:
		value, err = strconv.ParseUint(input, 2, 64)
	case Oct:
		value, err = strconv.ParseUint(input, 8, 64)
	case Dec:
		value, err = strconv.ParseUint(input, 10, 64)
	case Hex:
		input = strings.TrimPrefix(input, "0x")
		input = strings.TrimPrefix(input, "0X")
		value, err = strconv.ParseUint(input, 16, 64)
	default:
		return nil, fmt.Errorf("unsupported base: %d", fromBase)
	}

	if err != nil {
		return nil, fmt.Errorf("invalid %s value %q: %w", baseName(fromBase), input, err)
	}

	return &Result{
		Input:    input,
		FromBase: fromBase,
		Binary:   strconv.FormatUint(value, 2),
		Octal:    strconv.FormatUint(value, 8),
		Decimal:  strconv.FormatUint(value, 10),
		Hex:      strings.ToUpper(strconv.FormatUint(value, 16)),
	}, nil
}

// AutoDetect tries to determine the input base from common prefixes:
// 0x → hex, 0b → binary, 0 → octal, otherwise → decimal.
func AutoDetect(input string) (Base, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, fmt.Errorf("empty input")
	}

	if strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X") {
		return Hex, nil
	}
	if strings.HasPrefix(input, "0b") || strings.HasPrefix(input, "0B") {
		return Bin, nil
	}
	if strings.HasPrefix(input, "0") && len(input) > 1 {
		// Could be octal, but also check all digits 0-7
		allOctal := true
		for _, c := range input[1:] {
			if c < '0' || c > '7' {
				allOctal = false
				break
			}
		}
		if allOctal {
			return Oct, nil
		}
	}

	// Check if it's a valid hex string (contains A-F)
	hasHexLetters := false
	for _, c := range input {
		if (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			hasHexLetters = true
			break
		}
	}
	if hasHexLetters {
		return Hex, nil
	}

	return Dec, nil
}

func baseName(b Base) string {
	switch b {
	case Bin:
		return "binary"
	case Oct:
		return "octal"
	case Dec:
		return "decimal"
	case Hex:
		return "hex"
	default:
		return fmt.Sprintf("base-%d", b)
	}
}

// bound-checking helpers for ConvertUint64
const maxUint64 = math.MaxUint64
