// Package ipcalc implements IPv4 subnet calculation, ported from
// Krischan Jodies' ipcalc (http://jodies.de/ipcalc). It computes the
// network, broadcast, Cisco wildcard mask, and host range for a given
// address and netmask, and can display sub- and supernets, deaggregate
// address ranges, and split a network into subnets of requested sizes.
package ipcalc

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// defaultAddr is the fallback address used when the given address is
// invalid, mirroring ipcalc.pl.
const defaultAddr = 0xC0A80101 // 192.168.1.1

// Options controls how a Calc is rendered.
type Options struct {
	Color    bool // emit ANSI color codes
	NoBinary bool // suppress the bitwise output
	Class    bool // only print the natural class bit-count mask
}

// Calc holds the parsed inputs and any address-level parse errors.
type Calc struct {
	Address    uint32
	Address2   uint32   // second address for deaggregation
	Mask1      uint32   // primary netmask
	Mask2      uint32   // secondary netmask (sub-/supernet boundary)
	RangeMode  bool     // deaggregate Address-Address2
	SplitSizes []int    // requested split sizes in hosts
	Errors     []string // INVALID ADDRESS / MASK errors (rendered, not fatal)
}

var (
	dottedRe = regexp.MustCompile(`^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$`)
	bitsRe   = regexp.MustCompile(`^\d{1,2}$`)
	hexRe    = regexp.MustCompile(`^(0x)?[0-9A-Fa-f]{8}$`)
	digitsRe = regexp.MustCompile(`^\d+$`)
)

// Parse converts CLI arguments into a Calc, mirroring ipcalc.pl's
// getopts + argton. With splitMode set, pure-digit arguments are taken
// as split sizes (the -s n1 n2 n3 form) and the rest as address specs.
// With rangeMode set (the -r option), the remaining arguments are
// address1/address2 for deaggregation rather than address/mask. It
// returns an error only for option-level failures (e.g. -s without
// sizes); invalid addresses and masks are recorded in Calc.Errors and
// replaced with fallback values, as the original does.
func Parse(args []string, splitMode, rangeMode bool) (*Calc, error) {
	var specs []string
	var sizes []int
	if splitMode {
		for _, a := range args {
			if digitsRe.MatchString(a) {
				n, err := strconv.Atoi(a)
				if err != nil {
					return nil, fmt.Errorf("argument for -s is missing or invalid")
				}
				sizes = append(sizes, n)
			} else {
				specs = append(specs, a)
			}
		}
		if len(sizes) == 0 {
			return nil, fmt.Errorf("argument for -s is missing or invalid")
		}
	} else {
		specs = args
	}

	c := &Calc{}
	args2, rangeFromArgs := expandSpecs(specs)
	c.RangeMode = rangeMode || rangeFromArgs

	if len(args2) >= 1 {
		if v, ok := parseArg(args2[0], false); ok {
			c.Address = v
		} else {
			c.Errors = append(c.Errors, fmt.Sprintf("INVALID ADDRESS: %s", args2[0]))
			c.Address = defaultAddr
		}
	} else {
		c.Errors = append(c.Errors, "INVALID ADDRESS: ")
		c.Address = defaultAddr
	}

	if c.RangeMode {
		if len(args2) >= 2 {
			if v, ok := parseArg(args2[1], false); ok {
				c.Address2 = v
			} else {
				c.Errors = append(c.Errors, fmt.Sprintf("INVALID ADDRESS2: %s", args2[1]))
				c.Address2 = defaultAddr
			}
		} else {
			c.Errors = append(c.Errors, "INVALID ADDRESS2: ")
			c.Address2 = defaultAddr
		}
		return c, nil
	}

	// The original defaults the missing netmask to /24 (its help text
	// claims the natural class mask, but the code uses argton(24)).
	mask24 := maskFromBits(24)
	if len(args2) >= 2 {
		if v, ok := parseArg(args2[1], true); ok {
			c.Mask1 = v
		} else {
			c.Errors = append(c.Errors, fmt.Sprintf("INVALID MASK1:   %s", args2[1]))
			c.Mask1 = mask24
		}
	} else {
		c.Mask1 = mask24
	}
	if len(args2) >= 3 {
		if v, ok := parseArg(args2[2], true); ok {
			c.Mask2 = v
		} else {
			c.Errors = append(c.Errors, fmt.Sprintf("INVALID MASK2:   %s", args2[2]))
			c.Mask2 = mask24
		}
	} else {
		c.Mask2 = c.Mask1
	}

	if splitMode {
		c.SplitSizes = sizes
	}
	return c, nil
}

// expandSpecs splits address specifications into plain address/mask
// arguments, mirroring ipcalc.pl's getopts: "a/b" and "a/" split on the
// slash, "a-b" deaggregates a range, and a lone "-" marks range mode.
func expandSpecs(specs []string) (args []string, rangeMode bool) {
	for _, a := range specs {
		switch {
		case a == "-":
			rangeMode = true
		case strings.Contains(a, "/"):
			// A leading slash ("/24") is left intact for parseArg's
			// bit-count handling, as in the original.
			if before, after, found := strings.Cut(a, "/"); found && before != "" {
				if after != "" {
					args = append(args, before, after)
				} else {
					args = append(args, before)
				}
			} else {
				args = append(args, a)
			}
		case strings.Contains(a, "-"):
			if i := strings.LastIndexByte(a, '-'); i > 0 && i+1 < len(a) {
				args = append(args, a[:i], a[i+1:])
				rangeMode = true
			} else {
				args = append(args, a)
			}
		default:
			args = append(args, a)
		}
	}
	return args, rangeMode
}

// parseArg converts an address or mask argument to its 32-bit value,
// mirroring ipcalc.pl's argton: dotted quad, bit-count mask (with an
// optional leading slash), or 8-digit hex. When netmask is true, dotted
// and hex values are validated as netmasks and wildcard (inverted)
// masks are negated.
func parseArg(s string, netmask bool) (uint32, bool) {
	if m := dottedRe.FindStringSubmatch(s); m != nil {
		var n uint32
		for i := 1; i <= 4; i++ {
			v, _ := strconv.Atoi(m[i])
			if v > 255 {
				return 0, false
			}
			n |= uint32(v) << (24 - 8*(i-1))
		}
		if netmask {
			return validateMask(n)
		}
		return n, true
	}

	// bit-count mask, e.g. "24" or "/24"
	t := s
	if rest, found := strings.CutPrefix(t, "/"); found && digitsRe.MatchString(rest) {
		t = rest
	}
	if bitsRe.MatchString(t) {
		n, _ := strconv.Atoi(t)
		if n < 1 || n > 32 {
			return 0, false
		}
		return maskFromBits(n), true
	}

	// hex, e.g. "0xC0A80001" or "C0A80001"
	if hexRe.MatchString(t) {
		v, err := strconv.ParseUint(strings.TrimPrefix(t, "0x"), 16, 32)
		if err != nil {
			return 0, false
		}
		if netmask {
			return validateMask(uint32(v))
		}
		return uint32(v), true
	}
	return 0, false
}

// validateMask checks that m is a contiguous netmask. Wildcard (inverse)
// masks, as used in Cisco ACLs, are recognized and negated.
func validateMask(m uint32) (uint32, bool) {
	if m&(1<<31) == 0 {
		m = ^m
	}
	sawZero := false
	for i := 0; i < 32; i++ {
		if m&(1<<(31-i)) == 0 {
			sawZero = true
		} else if sawZero {
			return 0, false // ones after zeros: not a valid netmask
		}
	}
	return m, true
}

// maskFromBits returns the netmask value for a prefix length.
func maskFromBits(n int) uint32 {
	return ^uint32(0) << (32 - n)
}

// BitCount returns the prefix length of a contiguous netmask.
func BitCount(mask uint32) int {
	n := 0
	for n < 32 && mask&(1<<(31-n)) != 0 {
		n++
	}
	return n
}

// ntoa formats a 32-bit address as dotted quads.
func ntoa(v uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", v>>24, v>>16&0xff, v>>8&0xff, v&0xff)
}

// classMasks are the natural bit-count masks indexed by class (A=1..E=5),
// mirroring ipcalc.pl's @class. D and E keep the original's odd values.
var classMasks = [7]int{0, 8, 16, 24, 4, 5, 5}

// classNumber returns the natural class (1..5) of an address, or 0 when
// no class applies (all leading bits set).
func classNumber(addr uint32) int {
	class := 1
	for addr&(1<<(32-class)) != 0 {
		class++
		if class > 5 {
			return 0
		}
	}
	return class
}

// classLetter returns the class of an address as "A".."E", or "invalid".
func classLetter(addr uint32) string {
	if c := classNumber(addr); c != 0 {
		return string(rune('A' + c - 1))
	}
	return "invalid"
}

// classMask returns the natural class bit-count mask of an address as a
// string ("8", "16", ...), or "invalid", for the -c option.
func classMask(addr uint32) string {
	if c := classNumber(addr); c != 0 {
		return strconv.Itoa(classMasks[c])
	}
	return "invalid"
}

// netblock is a special-purpose address block with a display name.
type netblock struct {
	start, end uint32
	name       string
}

// netblocks mirrors ipcalc.pl's %netblocks table. The blocks are
// disjoint, so lookup order is irrelevant.
var netblocks = []netblock{
	{0xC0A80000, 0xC0A8FFFF, "Private Internet"}, // 192.168.0.0/16
	{0xAC100000, 0xAC1FFFFF, "Private Internet"}, // 172.16.0.0/12
	{0x0A000000, 0x0AFFFFFF, "Private Internet"}, // 10.0.0.0/8
	{0xA9FE0000, 0xA9FEFFFF, "APIPA"},            // 169.254.0.0/16
	{0x7F000000, 0x7FFFFFFF, "Loopback"},         // 127.0.0.0/8
	{0xE0000000, 0xEFFFFFFF, "Multicast"},        // 224.0.0.0/4
}

// blockName returns the name of the special block containing network,
// prefixed with "In Part " when only part of it lies inside.
func blockName(network, mask uint32) string {
	end := network | ^mask
	for _, b := range netblocks {
		match := 0
		if network >= b.start && network <= b.end {
			match++
		}
		if end >= b.start && end <= b.end {
			match++
		}
		if b.start > network && b.end < end {
			match = 1
		}
		if match == 1 {
			return "In Part " + b.name
		}
		if match == 2 {
			return b.name
		}
	}
	return ""
}

// Deaggregate returns the CIDR blocks covering the inclusive address
// range start..end, one per element.
func Deaggregate(start, end uint32) []string {
	var out []string
	base := uint64(start)
	end64 := uint64(end)
	for base <= end64 {
		step := 0
		for step < 32 && base&(1<<step) == 0 {
			if base|((1<<(step+1))-1) > end64 {
				break
			}
			step++
		}
		out = append(out, fmt.Sprintf("%s/%d", ntoa(uint32(base)), 32-step))
		base += 1 << step
	}
	return out
}

// round2PowerOf2 rounds n up to the nearest power of two.
func round2PowerOf2(n int) int {
	p := 1
	for p < n && p <= 1<<30 {
		p <<= 1
	}
	return p
}

// log2Int returns the base-2 logarithm of a power of two.
func log2Int(n int) int {
	log := 0
	for n > 1 {
		n >>= 1
		log++
	}
	return log
}
