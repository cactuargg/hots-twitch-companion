// Package replay parses read-only Blizzard versioned replay.details data.
// MPQ archive extraction remains outside this package until the next step.
package replay

import (
	"errors"
	"fmt"
	"io"
)

const maxReplay = 64 << 20

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
	Hero       string `json:"hero,omitempty"`
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
func ParseDetails(raw []byte) (Replay, error) {
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
		// m_hero in the completed replay describes the hero actually played.
		// BattleLobby's hero attribute may still contain a pre-ARAM selection.
		result.Players = append(result.Players, Player{Name: asStr(p[0]), Hero: asStr(p[10]), ToonHandle: handle, Result: out, ResultCode: rc, Team: asInt(p[5])})
	}
	if len(result.Players) == 0 {
		return Replay{}, errors.New("No players in details")
	}
	return result, nil
}
