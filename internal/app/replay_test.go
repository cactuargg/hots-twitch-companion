package app

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"hots-twitch-companion/internal/hots/replay"
	"hots-twitch-companion/internal/i18n"
	"os"
	"path/filepath"
	"testing"
)

func vintTest(n int64) []byte {
	v := uint64(n)
	first := byte((v & 63) << 1)
	v >>= 6
	if v != 0 {
		first |= 128
	}
	out := []byte{first}
	for v > 0 {
		b := byte(v & 127)
		v >>= 7
		if v != 0 {
			b |= 128
		}
		out = append(out, b)
	}
	return out
}
func blobTest(s string) []byte {
	out := append([]byte{2}, vintTest(int64(len(s)))...)
	return append(out, []byte(s)...)
}
func intTest(n int64) []byte { return append([]byte{9}, vintTest(n)...) }
func structTest(fields ...[]byte) []byte {
	out := append([]byte{5}, vintTest(int64(len(fields)/2))...)
	for _, f := range fields {
		out = append(out, f...)
	}
	return out
}
func playerTest(name string, id, code int64) []byte {
	toon := structTest(vintTest(0), intTest(2), vintTest(1), append([]byte{7}, []byte("Hero")...), vintTest(2), intTest(1), vintTest(4), intTest(id))
	return structTest(vintTest(0), blobTest(name), vintTest(1), toon, vintTest(5), intTest(0), vintTest(8), intTest(code))
}
func detailsTest() []byte {
	a := append([]byte{0}, vintTest(2)...)
	a = append(a, playerTest("Проверка", 10005, 1)...)
	a = append(a, playerTest("Opponent", 10006, 2)...)
	return structTest(vintTest(0), append([]byte{4, 1}, a...), vintTest(1), blobTest("Dragon Shire"), vintTest(10), blobTest(string(bytes.Repeat([]byte("synthetic-padding"), 30))))
}

// Unlike decryption, table encryption feeds the previous plain word into seed2.
func encryptTest(src []byte, key uint32) []byte {
	out := make([]byte, len(src))
	a, c := key, uint32(0xeeeeeeee)
	for i := 0; i < len(src); i += 4 {
		c += crypt[0x400+int(a&255)]
		v := binary.LittleEndian.Uint32(src[i:])
		binary.LittleEndian.PutUint32(out[i:], v^(a+c))
		a = ((^a << 21) + 0x11111111) | (a >> 11)
		c = v + c + (c << 5) + 3
	}
	return out
}
func buildTestMPQ(payload, details []byte) []byte {
	arch := make([]byte, 8)
	binary.LittleEndian.PutUint32(arch, 8)
	binary.LittleEndian.PutUint32(arch[4:], uint32(8+len(payload)))
	arch = append(arch, payload...)
	head := make([]byte, 32)
	copy(head, []byte{'M', 'P', 'Q', 0x1a})
	binary.LittleEndian.PutUint32(head[4:], 32)
	binary.LittleEndian.PutUint32(head[8:], uint32(112+len(arch)))
	binary.LittleEndian.PutUint16(head[12:], 0)
	binary.LittleEndian.PutUint16(head[14:], 3)
	binary.LittleEndian.PutUint32(head[16:], 32)
	binary.LittleEndian.PutUint32(head[20:], 96)
	binary.LittleEndian.PutUint32(head[24:], 4)
	binary.LittleEndian.PutUint32(head[28:], 1)
	h := bytes.Repeat([]byte{255}, 64)
	binary.LittleEndian.PutUint32(h, hashMPQ("replay.details", 1))
	binary.LittleEndian.PutUint32(h[4:], hashMPQ("replay.details", 2))
	binary.LittleEndian.PutUint16(h[8:], 0)
	binary.LittleEndian.PutUint16(h[10:], 0)
	binary.LittleEndian.PutUint32(h[12:], 0)
	b := make([]byte, 16)
	binary.LittleEndian.PutUint32(b, 112)
	binary.LittleEndian.PutUint32(b[4:], uint32(len(arch)))
	binary.LittleEndian.PutUint32(b[8:], uint32(len(details)))
	binary.LittleEndian.PutUint32(b[12:], 0x80000200)
	return bytes.Join([][]byte{head, encryptTest(h, hashMPQ("(hash table)", 3)), encryptTest(b, hashMPQ("(block table)", 3)), arch}, nil)
}
func TestReplayZlibAndBzip2(t *testing.T) {
	details := detailsTest()
	bz, _ := base64.StdEncoding.DecodeString("QlpoOTFBWSZTWaQ1RnwAAA93+vfwREBAAgRAiAAu4dwgQAAABIAUUhFgAAAEIACJCSRTanqGgAGgHpAG9UEVJGhpoAaDTJsoABfJRgTJUGh57RIk8DZGim5nCqTGNh4BAzGeoaZmM5MYLIXQsFaqijbEVlIVuFYit6nkSFZp4ItYrKArhFZ+BQK6egrSV5YisOdJzJIS05UkoiStEk569vaMjdc934d6fxdyRThQkKQ1Rnw=")
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	_, _ = zw.Write(details)
	_ = zw.Close()
	for _, test := range []struct {
		name    string
		payload []byte
	}{{"zlib", append([]byte{2}, z.Bytes()...)}, {"bzip2", append([]byte{0x10}, bz...)}} {
		t.Run(test.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "fixture.StormReplay")
			in := buildTestMPQ(test.payload, details)
			if e := os.WriteFile(file, in, 0600); e != nil {
				t.Fatal(e)
			}
			r, e := replay.Parse(file)
			if e != nil {
				t.Fatal(e)
			}
			if r.Map != "Dragon Shire" || len(r.Players) != 2 {
				t.Fatalf("unexpected replay %+v", r)
			}
			p, e := replay.FindOutcome(r, "Проверка", "", i18n.Ru)
			if e != nil || p.Result != "win" {
				t.Fatalf("wrong winner: %+v, %v", p, e)
			}
			p, e = replay.FindOutcome(r, "", "2-Hero-1-10006", i18n.En)
			if e != nil || p.Result != "loss" {
				t.Fatalf("wrong loser: %+v, %v", p, e)
			}
			after, e := os.ReadFile(file)
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(in, after) {
				t.Fatal("parser modified replay")
			}
		})
	}
}
func TestUnknownCompressionRejected(t *testing.T) {
	_, err := replay.UnpackCompressed([]byte{0x20, 1, 2})
	if err == nil {
		t.Fatal("unknown compression was accepted")
	}
}
