package ipcalc

import (
	"fmt"
	"sort"
	"strings"
)

// ANSI color codes used by the renderer, matching the original ipcalc.
const (
	colorQuads  = "\033[34m"        // dotted quads, blue
	colorNorm   = "\033[m"          // normal
	colorBinary = "\033[33m"        // binary bits, yellow
	colorMask   = "\033[31m"        // netmask bits, red
	colorClass  = "\033[35m"        // class bits, magenta
	colorSubnet = "\033[0m\033[32m" // subnet bits, green
	colorError  = "\033[31m"        // errors, red
)

// renderer writes ipcalc-style output, mirroring the layout and color
// handling of ipcalc.pl's printline/printnet.
type renderer struct {
	b    *strings.Builder
	errs []string
	opt  Options
}

// col returns the ANSI code for c when colors are enabled.
func (r *renderer) col(c string) string {
	if r.opt.Color {
		return c
	}
	return ""
}

// writeErrors prints the accumulated parse errors in red, followed by a
// blank line, as ipcalc.pl does before the calculated output.
func (r *renderer) writeErrors() {
	if len(r.errs) == 0 {
		return
	}
	r.b.WriteString(r.col(colorError))
	for _, e := range r.errs {
		r.b.WriteString(e)
		r.b.WriteByte('\n')
	}
	r.b.WriteString(r.col(colorNorm))
	r.b.WriteByte('\n')
}

// Render produces the full text output for the calculation.
func (c *Calc) Render(o Options) string {
	var b strings.Builder
	r := &renderer{b: &b, errs: c.Errors, opt: o}

	if o.Class {
		b.WriteString(classMask(c.Address)) // no trailing newline, as in the original
		return b.String()
	}

	if c.RangeMode {
		r.writeErrors()
		fmt.Fprintf(&b, "deaggregate %s - %s\n", ntoa(c.Address), ntoa(c.Address2))
		for _, cidr := range Deaggregate(c.Address, c.Address2) {
			b.WriteString(cidr)
			b.WriteByte('\n')
		}
		return b.String()
	}

	r.writeErrors()
	r.line("Address", c.Address, c.Mask1, c.Mask1)
	r.line("Netmask", c.Mask1, c.Mask1, c.Mask1)
	r.line("Wildcard", ^c.Mask1, c.Mask1, c.Mask1)
	b.WriteString("=>\n")

	network := c.Address & c.Mask1
	r.net(network, c.Mask1, c.Mask1)

	switch {
	case len(c.SplitSizes) > 0:
		r.split(network, c.Mask1, c.Mask2, c.SplitSizes)
	case BitCount(c.Mask1) < BitCount(c.Mask2):
		fmt.Fprintf(&b, "Subnets after transition from /%d to /%d\n\n", BitCount(c.Mask1), BitCount(c.Mask2))
		r.subnets(network, c.Mask1, c.Mask2)
	case BitCount(c.Mask1) > BitCount(c.Mask2):
		b.WriteString("Supernet\n\n")
		r.supernet(network, c.Mask1, c.Mask2)
	}
	return b.String()
}

// line renders one labeled value with its binary representation,
// mirroring ipcalc.pl's printline: a label padded to 11 columns, the
// value (plus "/24" or " = 24" suffix) padded to 21 columns, then the
// 32 bits split into octets with a space at the mask boundary. A space
// also follows bit 32, as the original does for /32 masks.
func (r *renderer) line(label string, addr, m1, m2 uint32) {
	mask1 := BitCount(m1)
	mask2 := BitCount(m2)
	additional := ""
	classBits := false
	switch label {
	case "Netmask":
		additional = fmt.Sprintf(" = %d", mask1)
	case "Network":
		classBits = true
		additional = fmt.Sprintf("/%d", mask1)
	case "Hostroute":
		classBits = true
	}

	r.b.WriteString(r.col(colorNorm))
	fmt.Fprintf(r.b, "%-11s", label+":")
	r.b.WriteString(r.col(colorQuads))
	fmt.Fprintf(r.b, "%-21s", ntoa(addr)+additional)

	if !r.opt.NoBinary {
		bitColor := colorBinary
		if label == "Netmask" {
			bitColor = colorMask
		}
		newBits := false
		// active reports the color of the next bit: class bits stay
		// magenta, subnet bits green, everything else the base color.
		active := func() string {
			switch {
			case classBits:
				return colorClass
			case newBits:
				return colorSubnet
			default:
				return bitColor
			}
		}
		if classBits {
			r.b.WriteString(r.col(colorClass))
		} else {
			r.b.WriteString(r.col(bitColor))
		}
		for i := 1; i <= 32; i++ {
			bit := byte('0')
			if addr&(1<<(32-i)) != 0 {
				bit = '1'
			}
			r.b.WriteByte(bit)
			if classBits && bit == '0' {
				classBits = false
				if newBits {
					r.b.WriteString(r.col(colorSubnet))
				} else {
					r.b.WriteString(r.col(bitColor))
				}
			}
			if i%8 == 0 && i < 32 {
				r.b.WriteString(r.col(colorNorm))
				r.b.WriteByte('.')
				r.b.WriteString(r.col(active()))
			}
			if i == mask1 {
				r.b.WriteByte(' ')
			}
			// In sub-/supernet views the bits between the two masks are
			// marked green.
			if (i == mask1 || i == mask2) && mask1 != mask2 {
				if newBits {
					newBits = false
					if !classBits {
						r.b.WriteString(r.col(bitColor))
					}
				} else {
					newBits = true
					if !classBits {
						r.b.WriteString(r.col(colorSubnet))
					}
				}
			}
		}
		r.b.WriteString(r.col(colorNorm))
	}
	r.b.WriteByte('\n')
}

// net renders the Network/HostMin/HostMax/Broadcast block plus the
// Hosts/Net summary, mirroring ipcalc.pl's printnet.
func (r *renderer) net(network, m1, m2 uint32) {
	mask := BitCount(m1)
	broadcast := network | ^m1
	hmin := network + 1
	hmax := broadcast - 1
	hostn := uint32(0)
	if mask == 31 {
		// RFC 3021 point-to-point link: both addresses are usable.
		hmin = network
		hmax = broadcast
		hostn = 2
	} else if mask == 32 {
		hostn = 1
	} else {
		hostn = hmax - hmin + 1
	}

	if mask == 32 {
		r.line("Hostroute", network, m1, m2)
	} else {
		r.line("Network", network, m1, m2)
		r.line("HostMin", hmin, m1, m2)
		r.line("HostMax", hmax, m1, m2)
		if mask < 31 {
			r.line("Broadcast", broadcast, m1, m2)
		}
	}

	r.b.WriteString(r.col(colorNorm))
	r.b.WriteString("Hosts/Net: ")
	r.b.WriteString(r.col(colorQuads))
	fmt.Fprintf(r.b, "%-22d", hostn)
	r.b.WriteString(r.description(network, m1))
	r.b.WriteString("\n\n")
}

// description renders "Class X, <block>, PtP Link RFC 3021" with the
// class part colored, mirroring ipcalc.pl's get_description.
func (r *renderer) description(network, mask uint32) string {
	var parts []string
	class := "Class " + classLetter(network)
	if r.opt.Color {
		parts = append(parts, r.col(colorClass)+class+r.col(colorNorm))
	} else {
		parts = append(parts, class)
	}
	if b := blockName(network, mask); b != "" {
		parts = append(parts, b)
	}
	if BitCount(mask) == 31 {
		parts = append(parts, "PtP Link RFC 3021")
	}
	return strings.Join(parts, ", ")
}

// subnets lists the subnets of network when the mask is shortened to
// mask2, mirroring ipcalc.pl's subnets.
func (r *renderer) subnets(network, mask1, mask2 uint32) {
	b1 := BitCount(mask1)
	b2 := BitCount(mask2)
	r.line("Netmask", mask2, mask2, mask1)
	r.line("Wildcard", ^mask2, mask2, mask1)
	r.b.WriteByte('\n')

	count := 1 << (b2 - b1)
	for subnet := 0; subnet < count; subnet++ {
		net := network | uint32(subnet)<<(32-b2)
		fmt.Fprintf(r.b, " %d.\n", subnet+1)
		r.net(net, mask2, mask1)
		if subnet >= 1000 {
			r.b.WriteString("... stopped at 1000 subnets ...\n")
			break
		}
	}
	hostn := int64(network|^mask2) - int64(network) - 1
	if hostn > -1 {
		fmt.Fprintf(r.b, "\nSubnets:   %d\n", count)
	}
	if hostn < 1 {
		hostn = 1
	}
	fmt.Fprintf(r.b, "Hosts:     %d\n", hostn*int64(count))
}

// supernet displays the supernet of network when the mask is extended to
// mask2, mirroring ipcalc.pl's supernet.
func (r *renderer) supernet(network, mask1, mask2 uint32) {
	network &= mask2
	r.line("Netmask", mask2, mask2, mask1)
	r.line("Wildcard", ^mask2, mask2, mask1)
	r.b.WriteByte('\n')
	r.net(network, mask2, mask1)
}

// split divides network into subnets sized for the requested host
// counts, mirroring ipcalc.pl's split_network. Each size is rounded up
// to a power of two (plus two for the network and broadcast addresses)
// and the subnets are allocated largest-first.
func (r *renderer) split(network, mask1, mask2 uint32, sizes []int) {
	firstAddress := network
	broadcast := network | ^mask1

	type alloc struct{ size, nr int }
	allocs := make([]alloc, len(sizes))
	needed := 0
	for i, s := range sizes {
		size := round2PowerOf2(s + 2)
		allocs[i] = alloc{size, i}
		needed += size
	}
	sort.SliceStable(allocs, func(a, b int) bool { return allocs[a].size > allocs[b].size })

	netAddr := make([]uint32, len(sizes))
	netMask := make([]int, len(sizes))
	for _, a := range allocs {
		netAddr[a.nr] = network
		netMask[a.nr] = 32 - log2Int(a.size)
		network += uint32(a.size)
	}

	for i, s := range sizes {
		fmt.Fprintf(r.b, "%d. Requested size: %d hosts\n", i+1, s)
		m := maskFromBits(netMask[i])
		r.line("Netmask", m, m, mask2)
		r.net(netAddr[i], m, mask2)
	}

	usedMask := 32 - log2Int(round2PowerOf2(needed))
	if usedMask < BitCount(mask1) {
		r.b.WriteString("Network is too small\n")
	}
	fmt.Fprintf(r.b, "Needed size:  %d addresses.\n", needed)
	fmt.Fprintf(r.b, "Used network: %s/%d\n", ntoa(firstAddress), usedMask)
	r.b.WriteString("Unused:\n")
	for _, cidr := range Deaggregate(network, broadcast) {
		r.b.WriteString(cidr)
		r.b.WriteByte('\n')
	}
}
