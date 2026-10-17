package lobby

import (
  "os"
  "path/filepath"
  "strings"
  "testing"
)

// White-box tests live next to the binary decoder rather than leaking reader
// internals into the application package.
func TestLobbyReaderWireOrder(t *testing.T) {
	r := lobbyReader{b: []byte{0x08, 0x73, 'a', 'b'}}
	if r.bits(6) != 8 || r.blob(10, 0) != "" || r.err == nil {
		t.Fatal("must reject truncated aligned blob")
	}
	r = lobbyReader{b: []byte{0x08, 0x02, 'a', 'b'}}
	if r.bits(6) != 8 || r.blob(10, 0) != "ab" || r.err != nil {
		t.Fatal("wrong chunk order or alignment")
	}
	r = lobbyReader{b: []byte("Hero")}
	if r.fourCC() != "Hero" {
		t.Fatal("wrong FourCC byte order")
	}
}

func TestBattleLobbyRejectsUnverifiedBuild(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..","..","..","testdata","battlelobby","replay.server.ARAM1_89754.battlelobby"))
	if err != nil {
		t.Fatal(err)
	}
	mutated := append([]byte(nil), raw...)
	matches := 0
	// Locate the independently known client-build values in the fixture's final
	// identity section, then encode an unsupported build in the same wire order.
	for pos := (len(raw) - 8192) * 8; pos+32 <= len(raw)*8; pos++ {
		r := lobbyReader{b: raw, pos: pos}
		if r.bits(32) != 89754 {
			continue
		}
		matches++
		remaining := 32
		at := pos
		for remaining > 0 {
			take := min(remaining, 8-at%8)
			mask := byte((1 << take) - 1)
			shift := uint(at % 8)
			chunk := byte(uint64(999999)>>uint(remaining-take)) & mask
			mutated[at/8] = (mutated[at/8] &^ (mask << shift)) | (chunk << shift)
			at += take
			remaining -= take
		}
	}
	if matches < 10 {
		t.Fatalf("found %d client builds, want at least ten", matches)
	}
	got, err := parseBattleLobby(mutated)
	if err == nil || got.Players != nil || !strings.Contains(err.Error(), "build") {
		t.Fatalf("accepted unverified layout: %+v %v", got, err)
	}
}
