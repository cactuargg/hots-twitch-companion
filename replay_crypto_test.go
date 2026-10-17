package main

import "strings"

// Independent test-only table encryption for MPQ fixture generation. The
// production MPQ decoder now resides in internal/hots/replay.
var crypt [0x500]uint32

func init() {
	seed := uint64(0x00100001)
	for i := 0; i < 256; i++ {
		for j, p := 0, i; j < 5; j, p = j+1, p+256 {
			seed = (seed*125 + 3) % 0x2aaaab
			hi := uint32(seed&0xffff) << 16
			seed = (seed*125 + 3) % 0x2aaaab
			crypt[p] = hi | uint32(seed&0xffff)
		}
	}
}
func hashMPQ(s string, typ int) uint32 {
	a, b := uint32(0x7fed7fed), uint32(0xeeeeeeee)
	for _, ch := range []byte(strings.ToUpper(strings.ReplaceAll(s, "/", `\`))) {
		a = crypt[(typ<<8)+int(ch)] ^ (a + b)
		b = uint32(ch) + a + b + (b << 5) + 3
	}
	return a
}
