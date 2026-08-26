package ipcalc

import (
	"slices"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		split       bool
		rng         bool
		wantAddr    uint32
		wantMask1   uint32
		wantMask2   uint32
		wantAddr2   uint32
		wantRange   bool
		wantSplit   []int
		wantErrors  []string
		wantParseOK bool
	}{
		{
			name:      "cidr",
			args:      []string{"192.168.0.1/24"},
			wantAddr:  0xC0A80001,
			wantMask1: maskFromBits(24),
			wantMask2: maskFromBits(24),
		},
		{
			name:      "default mask is /24",
			args:      []string{"10.0.0.1"},
			wantAddr:  0x0A000001,
			wantMask1: maskFromBits(24),
			wantMask2: maskFromBits(24),
		},
		{
			name:      "dotted mask and second mask",
			args:      []string{"192.168.0.1", "255.255.255.0", "255.255.255.128"},
			wantAddr:  0xC0A80001,
			wantMask1: maskFromBits(24),
			wantMask2: maskFromBits(25),
		},
		{
			name:      "wildcard mask is negated",
			args:      []string{"192.168.0.1", "0.0.63.255"},
			wantAddr:  0xC0A80001,
			wantMask1: maskFromBits(18),
			wantMask2: maskFromBits(18),
		},
		{
			name:      "bit-count mask as argument",
			args:      []string{"192.168.0.1", "24"},
			wantAddr:  0xC0A80001,
			wantMask1: maskFromBits(24),
			wantMask2: maskFromBits(24),
		},
		{
			name:      "hex address",
			args:      []string{"0xC0A80001/24"},
			wantAddr:  0xC0A80001,
			wantMask1: maskFromBits(24),
			wantMask2: maskFromBits(24),
		},
		{
			name:      "hex without prefix",
			args:      []string{"C0A80001/24"},
			wantAddr:  0xC0A80001,
			wantMask1: maskFromBits(24),
			wantMask2: maskFromBits(24),
		},
		{
			name:      "leading slash is a mask",
			args:      []string{"/24"},
			wantAddr:  0xFFFFFF00,
			wantMask1: maskFromBits(24),
			wantMask2: maskFromBits(24),
		},
		{
			name:      "trailing slash",
			args:      []string{"192.168.0.1/"},
			wantAddr:  0xC0A80001,
			wantMask1: maskFromBits(24),
			wantMask2: maskFromBits(24),
		},
		{
			name:       "invalid mask falls back to /24",
			args:       []string{"192.168.0.1/33"},
			wantAddr:   0xC0A80001,
			wantMask1:  maskFromBits(24),
			wantMask2:  maskFromBits(24),
			wantErrors: []string{"INVALID MASK1:   33"},
		},
		{
			name:       "mask zero is invalid",
			args:       []string{"192.168.0.1/0"},
			wantAddr:   0xC0A80001,
			wantMask1:  maskFromBits(24),
			wantMask2:  maskFromBits(24),
			wantErrors: []string{"INVALID MASK1:   0"},
		},
		{
			name:       "non-contiguous netmask is invalid",
			args:       []string{"192.168.0.1", "255.255.0.255"},
			wantAddr:   0xC0A80001,
			wantMask1:  maskFromBits(24),
			wantMask2:  maskFromBits(24),
			wantErrors: []string{"INVALID MASK1:   255.255.0.255"},
		},
		{
			name:       "invalid address falls back",
			args:       []string{"300.1.2.3"},
			wantAddr:   defaultAddr,
			wantMask1:  maskFromBits(24),
			wantMask2:  maskFromBits(24),
			wantErrors: []string{"INVALID ADDRESS: 300.1.2.3"},
		},
		{
			name:      "dash range as one argument",
			args:      []string{"192.168.1.10-192.168.2.5"},
			wantAddr:  0xC0A8010A,
			wantAddr2: 0xC0A80205,
			wantRange: true,
		},
		{
			name:      "dash range as three arguments",
			args:      []string{"192.168.1.10", "-", "192.168.2.5"},
			wantAddr:  0xC0A8010A,
			wantAddr2: 0xC0A80205,
			wantRange: true,
		},
		{
			name:      "range flag with two addresses",
			args:      []string{"192.168.1.10", "192.168.2.5"},
			rng:       true,
			wantAddr:  0xC0A8010A,
			wantAddr2: 0xC0A80205,
			wantRange: true,
		},
		{
			name:       "range flag without second address",
			args:       []string{"192.168.0.1"},
			rng:        true,
			wantAddr:   0xC0A80001,
			wantAddr2:  defaultAddr,
			wantRange:  true,
			wantErrors: []string{"INVALID ADDRESS2: "},
		},
		{
			name:      "split sizes",
			args:      []string{"10.0.0.0/24", "100", "50", "25"},
			split:     true,
			wantAddr:  0x0A000000,
			wantMask1: maskFromBits(24),
			wantMask2: maskFromBits(24),
			wantSplit: []int{100, 50, 25},
		},
		{
			name:      "split sizes before the address",
			args:      []string{"100", "50", "10.0.0.0/24"},
			split:     true,
			wantAddr:  0x0A000000,
			wantMask1: maskFromBits(24),
			wantMask2: maskFromBits(24),
			wantSplit: []int{100, 50},
		},
		{
			name:        "split without sizes is an option error",
			args:        []string{"10.0.0.0/24"},
			split:       true,
			wantParseOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := Parse(tt.args, tt.split, tt.rng)
			if err != nil {
				if !tt.wantParseOK {
					return
				}
				t.Fatalf("Parse: %v", err)
			}
			if c.Address != tt.wantAddr || c.Mask1 != tt.wantMask1 || c.Mask2 != tt.wantMask2 ||
				c.Address2 != tt.wantAddr2 || c.RangeMode != tt.wantRange {
				t.Errorf("Parse(%v) = addr %s mask1 %s mask2 %s addr2 %s range %v, want %s/%s/%s/%s/%v",
					tt.args, ntoa(c.Address), ntoa(c.Mask1), ntoa(c.Mask2), ntoa(c.Address2), c.RangeMode,
					ntoa(tt.wantAddr), ntoa(tt.wantMask1), ntoa(tt.wantMask2), ntoa(tt.wantAddr2), tt.wantRange)
			}
			if !slices.Equal(c.SplitSizes, tt.wantSplit) {
				t.Errorf("SplitSizes = %v, want %v", c.SplitSizes, tt.wantSplit)
			}
			if !slices.Equal(c.Errors, tt.wantErrors) {
				t.Errorf("Errors = %q, want %q", c.Errors, tt.wantErrors)
			}
		})
	}
}

func TestDeaggregate(t *testing.T) {
	got := Deaggregate(0xC0A8010A, 0xC0A80205) // 192.168.1.10 - 192.168.2.5
	want := []string{
		"192.168.1.10/31",
		"192.168.1.12/30",
		"192.168.1.16/28",
		"192.168.1.32/27",
		"192.168.1.64/26",
		"192.168.1.128/25",
		"192.168.2.0/30",
		"192.168.2.4/31",
	}
	if !slices.Equal(got, want) {
		t.Errorf("Deaggregate = %v, want %v", got, want)
	}
	if got := Deaggregate(0x0A000000, 0x0A000000); !slices.Equal(got, []string{"10.0.0.0/32"}) {
		t.Errorf("single address = %v", got)
	}
	if got := Deaggregate(0, 0xFFFFFFFF); !slices.Equal(got, []string{"0.0.0.0/0"}) {
		t.Errorf("full range = %v", got)
	}
}

func TestClassOption(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{"10.0.0.1", "8"},
		{"172.16.0.1", "16"},
		{"192.168.0.1", "24"},
		{"224.0.0.1", "4"}, // class D keeps the original's odd value
		{"240.0.0.1", "5"}, // class E keeps the original's odd value
		{"255.255.255.255", "invalid"},
		{"0.0.0.0", "8"},
		{"foo", "24"}, // invalid address falls back to 192.168.1.1
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			c, err := Parse([]string{tt.addr}, false, false)
			if err != nil {
				t.Fatal(err)
			}
			if got := c.Render(Options{Class: true}); got != tt.want {
				t.Errorf("class(%s) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}

func TestRenderBasic(t *testing.T) {
	c, err := Parse([]string{"192.168.0.1/24"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := `Address:   192.168.0.1          11000000.10101000.00000000. 00000001
Netmask:   255.255.255.0 = 24   11111111.11111111.11111111. 00000000
Wildcard:  0.0.0.255            00000000.00000000.00000000. 11111111
=>
Network:   192.168.0.0/24       11000000.10101000.00000000. 00000000
HostMin:   192.168.0.1          11000000.10101000.00000000. 00000001
HostMax:   192.168.0.254        11000000.10101000.00000000. 11111110
Broadcast: 192.168.0.255        11000000.10101000.00000000. 11111111
Hosts/Net: 254                   Class C, Private Internet

`
	if got := c.Render(Options{}); got != want {
		t.Errorf("Render:\n%q\nwant:\n%q", got, want)
	}
}

func TestRenderSubnets(t *testing.T) {
	c, err := Parse([]string{"192.168.0.1", "255.255.255.0", "255.255.255.128"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := `Address:   192.168.0.1          11000000.10101000.00000000. 00000001
Netmask:   255.255.255.0 = 24   11111111.11111111.11111111. 00000000
Wildcard:  0.0.0.255            00000000.00000000.00000000. 11111111
=>
Network:   192.168.0.0/24       11000000.10101000.00000000. 00000000
HostMin:   192.168.0.1          11000000.10101000.00000000. 00000001
HostMax:   192.168.0.254        11000000.10101000.00000000. 11111110
Broadcast: 192.168.0.255        11000000.10101000.00000000. 11111111
Hosts/Net: 254                   Class C, Private Internet

Subnets after transition from /24 to /25

Netmask:   255.255.255.128 = 25 11111111.11111111.11111111.1 0000000
Wildcard:  0.0.0.127            00000000.00000000.00000000.0 1111111

 1.
Network:   192.168.0.0/25       11000000.10101000.00000000.0 0000000
HostMin:   192.168.0.1          11000000.10101000.00000000.0 0000001
HostMax:   192.168.0.126        11000000.10101000.00000000.0 1111110
Broadcast: 192.168.0.127        11000000.10101000.00000000.0 1111111
Hosts/Net: 126                   Class C, Private Internet

 2.
Network:   192.168.0.128/25     11000000.10101000.00000000.1 0000000
HostMin:   192.168.0.129        11000000.10101000.00000000.1 0000001
HostMax:   192.168.0.254        11000000.10101000.00000000.1 1111110
Broadcast: 192.168.0.255        11000000.10101000.00000000.1 1111111
Hosts/Net: 126                   Class C, Private Internet


Subnets:   2
Hosts:     252
`
	if got := c.Render(Options{}); got != want {
		t.Errorf("Render:\n%q\nwant:\n%q", got, want)
	}
}

func TestRenderSupernet(t *testing.T) {
	c, err := Parse([]string{"10.0.0.1", "255.255.255.0", "255.255.0.0"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := `Address:   10.0.0.1             00001010.00000000.00000000. 00000001
Netmask:   255.255.255.0 = 24   11111111.11111111.11111111. 00000000
Wildcard:  0.0.0.255            00000000.00000000.00000000. 11111111
=>
Network:   10.0.0.0/24          00001010.00000000.00000000. 00000000
HostMin:   10.0.0.1             00001010.00000000.00000000. 00000001
HostMax:   10.0.0.254           00001010.00000000.00000000. 11111110
Broadcast: 10.0.0.255           00001010.00000000.00000000. 11111111
Hosts/Net: 254                   Class A, Private Internet

Supernet

Netmask:   255.255.0.0 = 16     11111111.11111111. 00000000.00000000
Wildcard:  0.0.255.255          00000000.00000000. 11111111.11111111

Network:   10.0.0.0/16          00001010.00000000. 00000000.00000000
HostMin:   10.0.0.1             00001010.00000000. 00000000.00000001
HostMax:   10.0.255.254         00001010.00000000. 11111111.11111110
Broadcast: 10.0.255.255         00001010.00000000. 11111111.11111111
Hosts/Net: 65534                 Class A, Private Internet

`
	if got := c.Render(Options{}); got != want {
		t.Errorf("Render:\n%q\nwant:\n%q", got, want)
	}
}

func TestRenderSplit(t *testing.T) {
	c, err := Parse([]string{"10.0.0.0/24", "100", "50", "25"}, true, false)
	if err != nil {
		t.Fatal(err)
	}
	want := `Address:   10.0.0.0             00001010.00000000.00000000. 00000000
Netmask:   255.255.255.0 = 24   11111111.11111111.11111111. 00000000
Wildcard:  0.0.0.255            00000000.00000000.00000000. 11111111
=>
Network:   10.0.0.0/24          00001010.00000000.00000000. 00000000
HostMin:   10.0.0.1             00001010.00000000.00000000. 00000001
HostMax:   10.0.0.254           00001010.00000000.00000000. 11111110
Broadcast: 10.0.0.255           00001010.00000000.00000000. 11111111
Hosts/Net: 254                   Class A, Private Internet

1. Requested size: 100 hosts
Netmask:   255.255.255.128 = 25 11111111.11111111.11111111.1 0000000
Network:   10.0.0.0/25          00001010.00000000.00000000.0 0000000
HostMin:   10.0.0.1             00001010.00000000.00000000.0 0000001
HostMax:   10.0.0.126           00001010.00000000.00000000.0 1111110
Broadcast: 10.0.0.127           00001010.00000000.00000000.0 1111111
Hosts/Net: 126                   Class A, Private Internet

2. Requested size: 50 hosts
Netmask:   255.255.255.192 = 26 11111111.11111111.11111111.11 000000
Network:   10.0.0.128/26        00001010.00000000.00000000.10 000000
HostMin:   10.0.0.129           00001010.00000000.00000000.10 000001
HostMax:   10.0.0.190           00001010.00000000.00000000.10 111110
Broadcast: 10.0.0.191           00001010.00000000.00000000.10 111111
Hosts/Net: 62                    Class A, Private Internet

3. Requested size: 25 hosts
Netmask:   255.255.255.224 = 27 11111111.11111111.11111111.111 00000
Network:   10.0.0.192/27        00001010.00000000.00000000.110 00000
HostMin:   10.0.0.193           00001010.00000000.00000000.110 00001
HostMax:   10.0.0.222           00001010.00000000.00000000.110 11110
Broadcast: 10.0.0.223           00001010.00000000.00000000.110 11111
Hosts/Net: 30                    Class A, Private Internet

Needed size:  224 addresses.
Used network: 10.0.0.0/24
Unused:
10.0.0.224/27
`
	if got := c.Render(Options{}); got != want {
		t.Errorf("Render:\n%q\nwant:\n%q", got, want)
	}
}

func TestRenderSlash31(t *testing.T) {
	c, err := Parse([]string{"192.168.0.1/31"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := `Address:   192.168.0.1          11000000.10101000.00000000.0000000 1
Netmask:   255.255.255.254 = 31 11111111.11111111.11111111.1111111 0
Wildcard:  0.0.0.1              00000000.00000000.00000000.0000000 1
=>
Network:   192.168.0.0/31       11000000.10101000.00000000.0000000 0
HostMin:   192.168.0.0          11000000.10101000.00000000.0000000 0
HostMax:   192.168.0.1          11000000.10101000.00000000.0000000 1
Hosts/Net: 2                     Class C, Private Internet, PtP Link RFC 3021

`
	if got := c.Render(Options{}); got != want {
		t.Errorf("Render:\n%q\nwant:\n%q", got, want)
	}
}

func TestRenderSlash32(t *testing.T) {
	c, err := Parse([]string{"192.168.0.1/32"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	// The original ipcalc leaves a trailing space after bit 32.
	want := "Address:   192.168.0.1          11000000.10101000.00000000.00000001 \n" +
		"Netmask:   255.255.255.255 = 32 11111111.11111111.11111111.11111111 \n" +
		"Wildcard:  0.0.0.0              00000000.00000000.00000000.00000000 \n" +
		"=>\n" +
		"Hostroute: 192.168.0.1          11000000.10101000.00000000.00000001 \n" +
		"Hosts/Net: 1                     Class C, Private Internet\n\n"
	if got := c.Render(Options{}); got != want {
		t.Errorf("Render:\n%q\nwant:\n%q", got, want)
	}
}

func TestRenderDeaggregate(t *testing.T) {
	c, err := Parse([]string{"192.168.1.10", "-", "192.168.2.5"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := `deaggregate 192.168.1.10 - 192.168.2.5
192.168.1.10/31
192.168.1.12/30
192.168.1.16/28
192.168.1.32/27
192.168.1.64/26
192.168.1.128/25
192.168.2.0/30
192.168.2.4/31
`
	if got := c.Render(Options{}); got != want {
		t.Errorf("Render:\n%q\nwant:\n%q", got, want)
	}
}

func TestRenderErrors(t *testing.T) {
	// Invalid masks and addresses are reported, then the calculation
	// proceeds with fallback values, as in the original.
	c, err := Parse([]string{"192.168.0.1/33"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := `INVALID MASK1:   33

Address:   192.168.0.1          11000000.10101000.00000000. 00000001
Netmask:   255.255.255.0 = 24   11111111.11111111.11111111. 00000000
Wildcard:  0.0.0.255            00000000.00000000.00000000. 11111111
=>
Network:   192.168.0.0/24       11000000.10101000.00000000. 00000000
HostMin:   192.168.0.1          11000000.10101000.00000000. 00000001
HostMax:   192.168.0.254        11000000.10101000.00000000. 11111110
Broadcast: 192.168.0.255        11000000.10101000.00000000. 11111111
Hosts/Net: 254                   Class C, Private Internet

`
	if got := c.Render(Options{}); got != want {
		t.Errorf("Render:\n%q\nwant:\n%q", got, want)
	}

	c, err = Parse([]string{"300.1.2.3"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Render(Options{}); !strings.HasPrefix(got, "INVALID ADDRESS: 300.1.2.3\n\nAddress:   192.168.1.1") {
		t.Errorf("invalid address output = %q", got)
	}
}

func TestRenderNoBinary(t *testing.T) {
	c, err := Parse([]string{"192.168.0.1/24"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := "Address:   192.168.0.1          \n" +
		"Netmask:   255.255.255.0 = 24   \n" +
		"Wildcard:  0.0.0.255            \n" +
		"=>\n" +
		"Network:   192.168.0.0/24       \n" +
		"HostMin:   192.168.0.1          \n" +
		"HostMax:   192.168.0.254        \n" +
		"Broadcast: 192.168.0.255        \n" +
		"Hosts/Net: 254                   Class C, Private Internet\n\n"
	if got := c.Render(Options{NoBinary: true}); got != want {
		t.Errorf("Render:\n%q\nwant:\n%q", got, want)
	}
}

func TestRenderColor(t *testing.T) {
	c, err := Parse([]string{"192.168.0.1/24"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	got := c.Render(Options{Color: true})
	for code, name := range map[string]string{
		colorQuads:  "quads",
		colorBinary: "binary",
		colorMask:   "netmask",
		colorClass:  "class",
		colorNorm:   "reset",
	} {
		if !strings.Contains(got, code) {
			t.Errorf("color output missing %s code %q", name, code)
		}
	}
	// The subnet view marks the new bits in green.
	c, err = Parse([]string{"192.168.0.1", "255.255.255.0", "255.255.255.128"}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Render(Options{Color: true}); !strings.Contains(got, colorSubnet) {
		t.Errorf("subnet view missing green new-bit color")
	}
}
