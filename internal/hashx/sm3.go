package hashx

import (
	"encoding/binary"
	"hash"
	"math/bits"
)

// SM3 constants (GB/T 32905-2016).
const (
	sm3Size      = 32
	sm3BlockSize = 64
)

var sm3IV = [8]uint32{
	0x7380166f, 0x4914b2b9, 0x172442d7, 0xda8a0600,
	0xa96f30bc, 0x163138aa, 0xe38dee4d, 0xb0fb0e4e,
}

type sm3Digest struct {
	h   [8]uint32
	x   [sm3BlockSize]byte
	nx  int
	len uint64
}

// NewSM3 returns a hash.Hash implementing the SM3 digest.
func NewSM3() hash.Hash {
	d := &sm3Digest{}
	d.Reset()
	return d
}

func (d *sm3Digest) Reset() {
	d.h = sm3IV
	d.nx = 0
	d.len = 0
	d.x = [sm3BlockSize]byte{}
}

func (d *sm3Digest) Size() int      { return sm3Size }
func (d *sm3Digest) BlockSize() int { return sm3BlockSize }

func (d *sm3Digest) Write(p []byte) (int, error) {
	nn := len(p)
	d.len += uint64(nn)

	if d.nx > 0 {
		n := copy(d.x[d.nx:], p)
		d.nx += n
		if d.nx == sm3BlockSize {
			d.block(d.x[:])
			d.nx = 0
		}
		p = p[n:]
	}
	for len(p) >= sm3BlockSize {
		d.block(p[:sm3BlockSize])
		p = p[sm3BlockSize:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return nn, nil
}

func (d *sm3Digest) Sum(in []byte) []byte {
	// Work on a copy so Sum does not disturb subsequent Writes.
	c := *d
	c.pad()
	out := make([]byte, sm3Size)
	for i, v := range c.h {
		binary.BigEndian.PutUint32(out[i*4:], v)
	}
	return append(in, out...)
}

func (d *sm3Digest) pad() {
	n := d.nx
	d.x[n] = 0x80
	n++
	if n > 56 {
		for i := n; i < sm3BlockSize; i++ {
			d.x[i] = 0
		}
		d.block(d.x[:])
		n = 0
	}
	for i := n; i < 56; i++ {
		d.x[i] = 0
	}
	binary.BigEndian.PutUint64(d.x[56:], d.len<<3)
	d.block(d.x[:])
}

func (d *sm3Digest) block(p []byte) {
	var w [68]uint32
	var w1 [64]uint32

	for i := 0; i < 16; i++ {
		w[i] = binary.BigEndian.Uint32(p[i*4:])
	}
	for j := 16; j < 68; j++ {
		w[j] = sm3P1(w[j-16]^w[j-9]^bits.RotateLeft32(w[j-3], 15)) ^
			bits.RotateLeft32(w[j-13], 7) ^ w[j-6]
	}
	for j := 0; j < 64; j++ {
		w1[j] = w[j] ^ w[j+4]
	}

	a, b, c, dd := d.h[0], d.h[1], d.h[2], d.h[3]
	e, f, g, hh := d.h[4], d.h[5], d.h[6], d.h[7]

	for j := 0; j < 64; j++ {
		a12 := bits.RotateLeft32(a, 12)
		ss1 := bits.RotateLeft32(a12+e+bits.RotateLeft32(sm3Tj(j), j), 7)
		ss2 := ss1 ^ a12
		tt1 := sm3FF(j, a, b, c) + dd + ss2 + w1[j]
		tt2 := sm3GG(j, e, f, g) + hh + ss1 + w[j]

		dd = c
		c = bits.RotateLeft32(b, 9)
		b = a
		a = tt1

		hh = g
		g = bits.RotateLeft32(f, 19)
		f = e
		e = sm3P0(tt2)
	}

	d.h[0] ^= a
	d.h[1] ^= b
	d.h[2] ^= c
	d.h[3] ^= dd
	d.h[4] ^= e
	d.h[5] ^= f
	d.h[6] ^= g
	d.h[7] ^= hh
}

func sm3Tj(j int) uint32 {
	if j < 16 {
		return 0x79cc4519
	}
	return 0x7a879d8a
}

func sm3FF(j int, x, y, z uint32) uint32 {
	if j < 16 {
		return x ^ y ^ z
	}
	return (x & y) | (x & z) | (y & z)
}

func sm3GG(j int, x, y, z uint32) uint32 {
	if j < 16 {
		return x ^ y ^ z
	}
	return (x & y) | (^x & z)
}

func sm3P0(x uint32) uint32 {
	return x ^ bits.RotateLeft32(x, 9) ^ bits.RotateLeft32(x, 17)
}

func sm3P1(x uint32) uint32 {
	return x ^ bits.RotateLeft32(x, 15) ^ bits.RotateLeft32(x, 23)
}
