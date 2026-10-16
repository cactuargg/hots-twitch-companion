package main

// Read-only MPQ and Blizzard versioned-data parser, based on public file formats.
// Reads replay.details and replay.server.battlelobby for correlation.
// No game-process access or file modifications.
import (
	"bytes"
	"compress/bzip2"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const maxReplay = 64 << 20

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
func decryptMPQ(b []byte, key uint32) ([]byte, error) {
	if len(b)%4 != 0 {
		return nil, errors.New("MPQ: damaged table")
	}
	out := make([]byte, len(b))
	a, c := key, uint32(0xeeeeeeee)
	for i := 0; i < len(b); i += 4 {
		c += crypt[0x400+int(a&255)]
		v := binary.LittleEndian.Uint32(b[i:]) ^ (a + c)
		binary.LittleEndian.PutUint32(out[i:], v)
		a = ((^a << 21) + 0x11111111) | (a >> 11)
		c = v + c + (c << 5) + 3
	}
	return out, nil
}
func checkedSlice(b []byte, off, size uint64) ([]byte, error) {
	if off > uint64(len(b)) || size > uint64(len(b))-off {
		return nil, errors.New("MPQ: member out of bounds")
	}
	return b[int(off):int(off+size)], nil
}
func unpackMPQ(b []byte) ([]byte, error) {
	if len(b) < 1 {
		return nil, errors.New("MPQ: empty compressed block")
	}
	var r io.Reader
	switch b[0] {
	case 0x02:
		zr, e := zlib.NewReader(bytes.NewReader(b[1:]))
		if e != nil {
			return nil, e
		}
		defer zr.Close()
		r = zr
	case 0x10:
		r = bzip2.NewReader(bytes.NewReader(b[1:]))
	default:
		return nil, fmt.Errorf("MPQ: unsupported compression 0x%02x (zlib 0x02 / BZip2 0x10 supported)", b[0])
	}
	lim := io.LimitReader(r, maxReplay+1)
	out, e := io.ReadAll(lim)
	if e != nil {
		return nil, e
	}
	if len(out) > maxReplay {
		return nil, errors.New("MPQ: decompressed member too large")
	}
	return out, nil
}
func readMember(filePath, name string) ([]byte, error) {
	stat, e := os.Stat(filePath)
	if e != nil {
		return nil, e
	}
	if stat.Size() <= 0 || stat.Size() > maxReplay {
		return nil, errors.New("Replay size exceeds 64 MiB limit")
	}
	data, e := os.ReadFile(filePath)
	if e != nil {
		return nil, e
	}
	return readMPQMember(data, name)
}

// All correlation members are decoded from one immutable, bounded file read.
func readMPQMember(data []byte, name string) ([]byte, error) {
	if len(data) < 32 || len(data) > maxReplay {
		return nil, errors.New("Not a valid MPQ")
	}
	start := uint64(0)
	if bytes.Equal(data[:4], []byte{'M', 'P', 'Q', 0x1b}) {
		start = uint64(binary.LittleEndian.Uint32(data[8:]))
	}
	head, e := checkedSlice(data, start, 32)
	if e != nil {
		return nil, e
	}
	if !bytes.Equal(head[:4], []byte{'M', 'P', 'Q', 0x1a}) {
		return nil, errors.New("Not an MPQ archive")
	}
	shift := binary.LittleEndian.Uint16(head[14:])
	ha := binary.LittleEndian.Uint32(head[16:])
	ba := binary.LittleEndian.Uint32(head[20:])
	hn := binary.LittleEndian.Uint32(head[24:])
	bn := binary.LittleEndian.Uint32(head[28:])
	if shift > 10 || hn == 0 || bn == 0 || hn > 65536 || bn > 65536 {
		return nil, errors.New("MPQ: unsupported table dimensions")
	}
	rawHash, e := checkedSlice(data, start+uint64(ha), uint64(hn)*16)
	if e != nil {
		return nil, e
	}
	rawBlock, e := checkedSlice(data, start+uint64(ba), uint64(bn)*16)
	if e != nil {
		return nil, e
	}
	h, e := decryptMPQ(rawHash, hashMPQ("(hash table)", 3))
	if e != nil {
		return nil, e
	}
	blocks, e := decryptMPQ(rawBlock, hashMPQ("(block table)", 3))
	if e != nil {
		return nil, e
	}
	id := -1
	wantedA, wantedB := hashMPQ(name, 1), hashMPQ(name, 2)
	for i := 0; i < int(hn); i++ {
		v := h[i*16:]
		if binary.LittleEndian.Uint32(v) == wantedA && binary.LittleEndian.Uint32(v[4:]) == wantedB {
			id = int(binary.LittleEndian.Uint32(v[12:]))
			break
		}
	}
	if id < 0 || id >= int(bn) {
		return nil, fmt.Errorf("MPQ: member %s not found", name)
	}
	item := blocks[id*16:]
	off := binary.LittleEndian.Uint32(item)
	compressed := binary.LittleEndian.Uint32(item[4:])
	plain := binary.LittleEndian.Uint32(item[8:])
	flags := binary.LittleEndian.Uint32(item[12:])
	if flags&0x80000000 == 0 || plain == 0 || plain > maxReplay {
		return nil, errors.New("MPQ: missing or oversize member")
	}
	if flags&(0x00010000|0x00000100) != 0 {
		return nil, errors.New("MPQ: encrypted or PKWARE member not supported")
	}
	packed, e := checkedSlice(data, start+uint64(off), uint64(compressed))
	if e != nil {
		return nil, e
	}
	var out []byte
	if flags&0x01000000 != 0 {
		if flags&0x200 != 0 && compressed < plain {
			out, e = unpackMPQ(packed)
		} else {
			out = packed
		}
	} else if flags&0x200 == 0 {
		out = packed
	} else {
		sectorSize := uint32(512) << shift
		count := (plain + sectorSize - 1) / sectorSize
		tableCount := count + 1
		if flags&0x04000000 != 0 {
			tableCount++
		}
		if uint64(len(packed)) < uint64(tableCount)*4 {
			return nil, errors.New("MPQ: truncated sector table")
		}
		result := make([]byte, 0, int(plain))
		for i := uint32(0); i < count; i++ {
			lo := binary.LittleEndian.Uint32(packed[i*4:])
			hi := binary.LittleEndian.Uint32(packed[(i+1)*4:])
			if hi < lo {
				return nil, errors.New("MPQ: corrupt sector offsets")
			}
			part, er := checkedSlice(packed, uint64(lo), uint64(hi-lo))
			if er != nil {
				return nil, er
			}
			expected := sectorSize
			if remaining := plain - i*sectorSize; remaining < expected {
				expected = remaining
			}
			if uint32(len(part)) < expected {
				part, er = unpackMPQ(part)
				if er != nil {
					return nil, er
				}
			}
			if uint32(len(part)) != expected {
				return nil, errors.New("MPQ: sector size mismatch")
			}
			result = append(result, part...)
		}
		out = result
	}
	if e != nil {
		return nil, e
	}
	if len(out) != int(plain) {
		return nil, fmt.Errorf("MPQ: decoded size mismatch %d != %d", len(out), plain)
	}
	return out, nil
}

type vReader struct {
	b        []byte
	i, count int
}

func (r *vReader) byte() (byte, error) {
	if r.i >= len(r.b) {
		return 0, io.ErrUnexpectedEOF
	}
	b := r.b[r.i]
	r.i++
	return b, nil
}
func (r *vReader) bs(n int) ([]byte, error) {
	if n < 0 || n > maxReplay || n > len(r.b)-r.i {
		return nil, io.ErrUnexpectedEOF
	}
	b := r.b[r.i : r.i+n]
	r.i += n
	return b, nil
}
func (r *vReader) vint() (int64, error) {
	b, e := r.byte()
	if e != nil {
		return 0, e
	}
	neg := b&1 == 1
	v := uint64((b >> 1) & 0x3f)
	shift := uint(6)
	for b&0x80 != 0 {
		if shift > 60 {
			return 0, errors.New("Versioned integer overflow")
		}
		b, e = r.byte()
		if e != nil {
			return 0, e
		}
		v |= uint64(b&0x7f) << shift
		shift += 7
	}
	if v > uint64(1<<63-1) {
		return 0, errors.New("Versioned integer out of bounds")
	}
	if neg {
		return -int64(v), nil
	}
	return int64(v), nil
}
func (r *vReader) read(depth int) (any, error) {
	r.count++
	if depth > 48 || r.count > 50000 {
		return nil, errors.New("Versioned nesting limit")
	}
	kind, e := r.byte()
	if e != nil {
		return nil, e
	}
	switch kind {
	case 0:
		n, e := r.vint()
		if e != nil {
			return nil, e
		}
		if n < 0 || n > 10000 {
			return nil, errors.New("Versioned array size invalid")
		}
		a := make([]any, int(n))
		for i := range a {
			a[i], e = r.read(depth + 1)
			if e != nil {
				return nil, e
			}
		}
		return a, nil
	case 1:
		bits, e := r.vint()
		if e != nil {
			return nil, e
		}
		if bits < 0 || bits > maxReplay*8 {
			return nil, errors.New("Versioned bit length")
		}
		return r.bs(int((bits + 7) / 8))
	case 2:
		n, e := r.vint()
		if e != nil {
			return nil, e
		}
		return r.bs(int(n))
	case 3:
		n, e := r.vint()
		if e != nil {
			return nil, e
		}
		v, e := r.read(depth + 1)
		return map[int]any{-1: n, 0: v}, e
	case 4:
		b, e := r.byte()
		if e != nil {
			return nil, e
		}
		if b == 0 {
			return nil, nil
		}
		return r.read(depth + 1)
	case 5:
		n, e := r.vint()
		if e != nil {
			return nil, e
		}
		if n < 0 || n > 10000 {
			return nil, errors.New("Versioned field count")
		}
		o := make(map[int]any)
		for i := int64(0); i < n; i++ {
			t, e := r.vint()
			if e != nil {
				return nil, e
			}
			if t < 0 || t > 100000 {
				return nil, errors.New("Versioned tag invalid")
			}
			o[int(t)], e = r.read(depth + 1)
			if e != nil {
				return nil, e
			}
		}
		return o, nil
	case 6:
		b, e := r.byte()
		return b != 0, e
	case 7:
		return r.bs(4)
	case 8:
		return r.bs(8)
	case 9:
		return r.vint()
	default:
		return nil, fmt.Errorf("Unknown versioned type: %d", kind)
	}
}

type Player struct {
	Name       string `json:"name"`
	ToonHandle string `json:"toonHandle"`
	Result     string `json:"result"`
	ResultCode int64  `json:"resultCode"`
	Team       int64  `json:"team"`
}
type Replay struct {
	Map        string   `json:"map"`
	Players    []Player `json:"players"`
	RecordedAt int64    `json:"recordedAt,omitempty"`
}

func asObj(v any) map[int]any { o, _ := v.(map[int]any); return o }
func asArr(v any) []any       { a, _ := v.([]any); return a }
func asStr(v any) string      { b, _ := v.([]byte); return string(b) }
func asInt(v any) int64       { n, _ := v.(int64); return n }
func parseReplay(path string) (Replay, error) {
	raw, e := readMember(path, "replay.details")
	if e != nil {
		return Replay{}, e
	}
	return parseReplayDetails(raw)
}

func parseReplayDetails(raw []byte) (Replay, error) {
	r := vReader{b: raw}
	v, e := r.read(0)
	if e != nil {
		return Replay{}, e
	}
	root := asObj(v)
	if root == nil {
		return Replay{}, errors.New("Replay details missing")
	}
	arr := asArr(root[0])
	if len(arr) == 0 {
		return Replay{}, errors.New("Replay player list missing")
	}
	result := Replay{Map: asStr(root[1])}
	// Blizzard m_timeUTC is a Windows FILETIME, independent of file mtime.
	if ticks := asInt(root[5]); ticks > 116444736000000000 {
		result.RecordedAt = (ticks - 116444736000000000) / 10000
	}
	for _, item := range arr {
		p := asObj(item)
		if p == nil {
			continue
		}
		t := asObj(p[1])
		handle := ""
		if len(t) > 0 {
			handle = fmt.Sprintf("%d-%s-%d-%d", asInt(t[0]), asStr(t[1]), asInt(t[2]), asInt(t[4]))
		}
		rc := asInt(p[8])
		out := "unknown"
		if rc == 1 {
			out = "win"
		} else if rc == 2 {
			out = "loss"
		}
		result.Players = append(result.Players, Player{Name: asStr(p[0]), ToonHandle: handle, Result: out, ResultCode: rc, Team: asInt(p[5])})
	}
	if len(result.Players) == 0 {
		return Replay{}, errors.New("No players in details")
	}
	return result, nil
}
func findOutcome(r Replay, playerName, toon, lang string) (Player, error) {
	var matches []Player
	for _, p := range r.Players {
		if toon != "" && strings.EqualFold(p.ToonHandle, toon) || toon == "" && strings.EqualFold(p.Name, strings.TrimSpace(playerName)) {
			matches = append(matches, p)
		}
	}
	if len(matches) != 1 {
		return Player{}, errors.New(tr(lang, "replay.player_matches", len(matches)))
	}
	if matches[0].Result == "unknown" {
		return Player{}, errors.New(tr(lang, "replay.unknown_outcome"))
	}
	return matches[0], nil
}
