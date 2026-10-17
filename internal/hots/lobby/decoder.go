package lobby

// Layout adapted from HeroesToolChest/Heroes.StormReplayParser (MIT).
// See testdata/battlelobby/README.md for the pinned source and limitations.
import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

const MaxBytes = 8 << 20

type LobbyPlayer struct {
	Slot       int    `json:"slot"`
	SlotType   string `json:"slotType"`
	Role       string `json:"role"`
	Hero       string `json:"hero"`
	ToonHandle string `json:"toonHandle,omitempty"`
}

type BattleLobby struct {
	Build      int           `json:"build"`
	RandomSeed uint32        `json:"randomSeed"` // Random value, never a match identifier.
	AMMID      uint32        `json:"ammId"`
	Custom     bool          `json:"custom"`
	TeamSize   string        `json:"teamSize"`
	Players    []LobbyPlayer `json:"players"`
}

// The format consumes low bits first within a byte, then concatenates chunks
// in big-endian order. FourCC strings use the resulting integer in big-endian byte order.
type lobbyReader struct {
	b   []byte
	pos int
	err error
}

func (r *lobbyReader) skip(n int) {
	if r.err != nil {
		return
	}
	if n < 0 || n > len(r.b)*8-r.pos {
		r.err = io.ErrUnexpectedEOF
		return
	}
	r.pos += n
}
func (r *lobbyReader) bits(n int) uint64 {
	if r.err != nil {
		return 0
	}
	if n < 0 || n > 64 || n > len(r.b)*8-r.pos {
		r.err = io.ErrUnexpectedEOF
		return 0
	}
	var v uint64
	for n > 0 {
		take := min(n, 8-r.pos%8)
		v = v<<take | uint64(r.b[r.pos/8]>>uint(r.pos%8))&((1<<take)-1)
		r.pos += take
		n -= take
	}
	return v
}
func (r *lobbyReader) yes() bool { return r.bits(1) != 0 }
func (r *lobbyReader) aligned(n int) []byte {
	r.skip((8 - r.pos%8) % 8)
	start := r.pos / 8
	r.skip(n * 8)
	if r.err != nil {
		return nil
	}
	return r.b[start : start+n]
}
func (r *lobbyReader) blob(width, extra int) string {
	n := int(r.bits(width)) + extra
	return string(r.aligned(n))
}
func (r *lobbyReader) fourCC() string {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(r.bits(32)))
	return strings.Trim(string(b[:]), "\x00")
}
func (r *lobbyReader) handles() {
	n := int(r.bits(6))
	for i := 0; i < n; i++ {
		r.aligned(40)
	}
}
func (r *lobbyReader) optional(n int) {
	if r.yes() {
		r.skip(n)
	}
}

// Only the modern player-info layout is supported. The cosmetics tail is
// deliberately opaque; success means the roster prefix parsed, not the whole file.
func Parse(raw []byte) (BattleLobby, error) {
	result, err := decodeBattleLobby(raw)
	if err != nil {
		return BattleLobby{}, err
	}
	return result, nil
}
func decodeBattleLobby(raw []byte) (BattleLobby, error) {
	var out BattleLobby
	if len(raw) == 0 || len(raw) > MaxBytes {
		return out, errors.New("battlelobby: invalid size")
	}
	r := lobbyReader{b: raw}
	fail := func(message string) (BattleLobby, error) {
		return BattleLobby{}, fmt.Errorf("battlelobby at bit %d: %s", r.pos, message)
	}
	n := int(r.bits(6))
	for i := 0; i < n; i++ {
		r.blob(10, 0)
	}
	r.handles()
	// Attribute definitions and selected values are indexed by namespace + ID.
	type key struct{ ns, id uint64 }
	attrs := map[key][]string{}
	n = int(r.bits(9))
	for i := 0; i < n && r.err == nil; i++ {
		k := key{r.bits(32), r.bits(32)}
		if _, ok := attrs[k]; ok {
			return fail("duplicate attribute")
		}
		values := []string{}
		count := int(r.bits(12))
		for j := 0; j < count && r.err == nil; j++ {
			values = append(values, r.fourCC())
			for x := 0; x < 2; x++ {
				if r.yes() {
					r.optional(32)
					r.skip(6 + 16)
				}
			}
			r.optional(6 + 32 + 16 + 16)
			r.skip(3)
		}
		attrs[k] = values
		if r.yes() {
			r.skip(25)
			count = int(r.bits(5))
			for j := 0; j < count && r.err == nil; j++ {
				r.skip(5 + 32 + 32)
				c := int(r.bits(12))
				r.skip(c * 32)
			}
		}
		r.skip(15 + 32)
		count = int(r.bits(8))
		r.skip(count * (5 + 165 + 15))
		r.skip(11 + 1 + 9)
	}
	r.skip(30 + 16 + 81)
	slots := [16]LobbyPlayer{}
	for i := range slots {
		slots[i].Slot = i
	}
	selected := map[key]bool{}
	n = int(r.bits(9))
	if n != len(attrs) {
		return fail("attribute counts differ")
	}
	for i := 0; i < n && r.err == nil; i++ {
		k := key{r.bits(32), r.bits(32)}
		values, ok := attrs[k]
		if !ok || selected[k] {
			return fail("unknown or repeated attribute selection")
		}
		selected[k] = true
		choice := r.bits(8)
		count := 1
		if choice == 1 {
			count = int(r.bits(5))
			if count > 16 {
				return fail("too many slots")
			}
		} else if choice != 0 {
			return fail("unsupported attribute choice")
		}
		for j := 0; j < count && r.err == nil; j++ {
			index := int(r.bits(11))
			r.skip(1)
			if index >= len(values) {
				return fail("attribute index out of bounds")
			}
			v := values[index]
			if choice == 0 {
				switch k.id {
				case 2001:
					out.TeamSize = v
				case 3009:
					out.Custom = v == "Priv"
				}
			} else {
				switch k.id {
				case 500:
					slots[j].SlotType = v
				case 3007:
					slots[j].Role = v
				case 4002:
					slots[j].Hero = v
				}
			}
		}
	}
	r.handles()
	n = int(r.bits(5))
	for i := 0; i < n && r.err == nil; i++ {
		r.skip(32)
		r.handles()
	}
	r.skip(128)
	n = int(r.bits(5))
	r.skip(n * (32 + 32 + 4))
	// Mod metadata, including optional hero data. No cache files are opened.
	n = int(r.bits(7))
	for i := 0; i < n && r.err == nil; i++ {
		r.skip(64 + 16 + 16 + 24 + 7 + 15 + 2 + 74)
		r.handles()
		r.skip(1)
		r.skip(8 + 32 + 32)
		r.blob(7, 2)
		r.skip(8 + 32 + 32 + 64)
		r.optional(32)
		r.optional(23)
		r.skip(1)
		r.optional(32)
		r.skip(3 + 30 + 3 + 16 + 8 + 8 + 34)
		r.optional(54)
		r.skip(1 + 5 + 5 + 5)
		r.skip(64 + 32 + 1 + 21 + 32 + 28 + 36)
		if r.yes() {
			r.skip(11)
			c := int(r.bits(4))
			r.skip(c * (6 + 32 + 16 + 16 + 23))
			r.skip(4)
			choice := r.bits(4)
			if choice == 5 {
				r.skip(40 + 16 + 79 + 160 + 105)
			} else if choice > 5 {
				return fail("unsupported mod choice")
			}
			c = int(r.bits(7))
			r.skip(c * 23)
			r.skip(11)
			r.optional(38 + 16 + 16)
			r.skip(1)
			c = int(r.bits(5))
			r.skip(c * 32)
			r.skip(23)
		}
		c := int(r.bits(6))
		r.skip(c * 64)
		r.skip(32)
	}
	r.handles()
	// Collection entries are byte aligned; the parallel presence matrix has 16 slots.
	n = int(r.bits(16))
	for i := 0; i < n && r.err == nil; i++ {
		r.aligned(8)
	}
	if int(r.bits(32)) != n {
		return fail("collection counts differ")
	}
	r.aligned(n * 32)
	n = int(r.bits(8))
	r.skip(n * 32) // disabled heroes, layout >= 85027
	out.RandomSeed = uint32(r.bits(32))
	r.skip(32)
	n = int(r.bits(5))
	if n < 1 || n > 16 {
		return fail("invalid client count")
	}
	seen := map[int]bool{}
	toons := map[string]bool{}
	for i := 0; i < n && r.err == nil; i++ {
		r.skip(32)
		slot := int(r.bits(5))
		if slot > 15 || seen[slot] {
			return fail("invalid or repeated client slot")
		}
		seen[slot] = true
		region := r.bits(8)
		program := r.bits(32)
		realm := r.bits(32)
		id := r.bits(64)
		if program != 1214607983 {
			return fail("unsupported toon program")
		}
		toon := fmt.Sprintf("%d-Hero-%d-%d", region, realm, id)
		if toons[toon] {
			return fail("duplicate toon")
		}
		toons[toon] = true
		slots[slot].ToonHandle = toon
		r.skip(8)
		if r.fourCC() != "Hero" {
			return fail("invalid shortcut program")
		}
		r.skip(32)
		r.blob(7, 2)
		r.skip(1 + 2)
		if r.yes() {
			r.skip(2 + 8)
			if r.fourCC() != "Hero" {
				return fail("invalid repeated shortcut")
			}
			r.skip(32)
			r.blob(7, 2)
			r.skip(1)
		}
		r.skip(198)
		r.optional(64)
		r.skip(4)
		build := int(r.bits(32))
		// Extend this list only with a real fixture verifying the new layout.
		knownBuild := build == 89566 || build == 89754 || build == 90670 || build == 93640 || build == 98348
		if !knownBuild || (out.Build != 0 && out.Build != build) {
			return fail("unsupported or inconsistent client build")
		}
		out.Build = build
		if !r.yes() {
			c := int(r.bits(12))
			r.skip(c + 1)
		}
		r.skip(4)
		r.optional(64) // silence flags, staff flag, party
		if r.yes() {
			tag := r.blob(7, 0)
			if tag != "" && !strings.Contains(tag, "#") {
				return fail("invalid battletag")
			}
		}
		r.skip(32 + 1)
	}
	r.skip(64)
	if r.yes() {
		r.skip(8 + 32 + 32)
		out.AMMID = uint32(r.bits(32))
		r.skip(32)
	} else {
		r.skip(16)
	}
	if r.err != nil {
		return fail(r.err.Error())
	}
	// Keep observers and computers; absence of a ToonHandle is meaningful.
	for _, p := range slots {
		if p.SlotType == "Humn" && p.ToonHandle == "" {
			return fail("human slot has no client identity")
		}
		if p.SlotType == "Humn" || p.SlotType == "Comp" {
			out.Players = append(out.Players, p)
		}
	}
	if len(out.Players) == 0 {
		return fail("no occupied slots")
	}
	return out, nil
}
