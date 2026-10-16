package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRealBattleLobbies(t *testing.T) {
	for _, tc := range []struct {
		name    string
		build   int
		seed    uint32
		amm     uint32
		clients int
	}{
		{"StormLeague1_89566", 89566, 3685014112, 50091, 10},
		{"ARAM1_89754", 89754, 2002111530, 50101, 10},
		{"Custom1_90670", 90670, 243203581, 0, 1},
		{"ARAM2_93640", 93640, 893217909, 50101, 10},
		{"IndustrialDistrict1_98348", 98348, 427522310, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "battlelobby", "replay.server."+tc.name+".battlelobby"))
			if err != nil {
				t.Fatal(err)
			}
			got, err := parseBattleLobby(raw)
			if err != nil {
				t.Fatal(err)
			}
			if got.Build != tc.build || got.AMMID != tc.amm || (tc.seed != 0 && got.RandomSeed != tc.seed) {
				t.Fatalf("wrong game properties: %+v", got)
			}
			clients := 0
			for _, p := range got.Players {
				if p.ToonHandle != "" {
					clients++
					if !strings.Contains(p.ToonHandle, "-Hero-") {
						t.Fatal(p)
					}
				}
			}
			if clients != tc.clients {
				t.Fatalf("clients %d, want %d", clients, tc.clients)
			}
			if tc.name == "ARAM1_89754" && (got.Players[4].Hero != "Ragn" || !strings.HasPrefix(got.Players[4].ToonHandle, "1-Hero-1-12519")) {
				t.Fatal("ARAM roster differs from upstream assertions")
			}
			if tc.name == "StormLeague1_89566" && (got.TeamSize != "5v5" || got.Players[4].Hero != "Mura") {
				t.Fatal("ranked attributes differ from upstream assertions")
			}
			if tc.name == "Custom1_90670" && (!got.Custom || len(got.Players) != 11 || got.Players[0].Role != "Watc" || got.Players[1].SlotType != "Comp" || got.Players[1].ToonHandle != "") {
				t.Fatal("observer/AI roster differs from upstream assertions")
			}
			if tc.name == "IndustrialDistrict1_98348" && (!got.Custom || got.TeamSize != "5v5" || got.Players[0].SlotType != "Humn" || got.Players[0].ToonHandle != "2-Hero-1-7669040" || got.Players[1].SlotType != "Comp") {
				t.Fatal("98348 layout differs from the verified custom-game roster")
			}
		})
	}
}

func TestBattleLobbyCapturePersistence(t *testing.T) {
	raw, err := os.ReadFile("testdata/battlelobby/replay.server.ARAM1_89754.battlelobby")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "replay.server.battlelobby")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	got := captureBattleLobby(path)
	hash := sha256.Sum256(raw)
	if got.Status != "roster_parsed" || got.Data == nil || got.SHA256 != hex.EncodeToString(hash[:]) || got.Size != int64(len(raw)) || got.ModifiedAt == 0 {
		t.Fatalf("capture: %+v", got)
	}
	p := paths{state: filepath.Join(dir, "state.json")}
	state := State{LastLobby: got, Active: &Prediction{ID: "unchanged"}}
	if err := saveState(p, state); err != nil {
		t.Fatal(err)
	}
	restored, err := loadState(p)
	if err != nil || !reflect.DeepEqual(state, restored) {
		t.Fatalf("restart round trip: %+v, %v", restored, err)
	}
	if err := os.WriteFile(path, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	bad := captureBattleLobby(path)
	if bad.Status != "parse_error" || bad.Data != nil || bad.SHA256 == "" || bad.Error == "" {
		t.Fatalf("stale parsed data: %+v", bad)
	}
	missing := captureBattleLobby(filepath.Join(dir, "missing"))
	if missing.Status != "read_error" || missing.Data != nil {
		t.Fatalf("missing: %+v", missing)
	}
	if err := os.WriteFile(p.state, []byte(`{"active":null,"lastLobbyKey":"legacy"}`), 0600); err != nil {
		t.Fatal(err)
	}
	legacy, err := loadState(p)
	if err != nil || legacy.LastLobby != nil || legacy.LastLobbyKey != "legacy" {
		t.Fatalf("legacy state: %+v %v", legacy, err)
	}
}

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

func TestBattleLobbyRejectsInvalid(t *testing.T) {
	raw, err := os.ReadFile("testdata/battlelobby/replay.server.ARAM1_89754.battlelobby")
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range [][]byte{nil, {}, {0}, make([]byte, 1000), []byte("not a lobby"), raw[:100], raw[:len(raw)/2], make([]byte, maxLobbyBytes+1)} {
		got, err := parseBattleLobby(b)
		if err == nil || len(got.Players) != 0 {
			t.Fatalf("accepted invalid input of %d bytes", len(b))
		}
	}
	for cut := 0; cut < len(raw)/2; cut += 4096 {
		if _, err := parseBattleLobby(raw[:cut]); err == nil {
			t.Fatalf("accepted truncated prefix at %d", cut)
		}
	}
}

func TestBattleLobbyRejectsUnverifiedBuild(t *testing.T) {
	raw, err := os.ReadFile("testdata/battlelobby/replay.server.ARAM1_89754.battlelobby")
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

func TestScannerPersistsLobbyWithoutChangingActivePrediction(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	root := filepath.Join(os.TempDir(), "Heroes of the Storm", "TempWriteReplayP1")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("testdata/battlelobby/replay.server.ARAM1_89754.battlelobby")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "replay.server.battlelobby")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	mod := now.Add(-5 * time.Second)
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
	e, p := scanFixture(t, now.Add(-10*time.Second))
	original := *e.state.Active
	lastError := ""
	lastLog := time.Time{}
	e.scanTick(context.Background(), now.Add(-15*time.Second), &lastError, &lastLog)
	restored, err := loadState(p)
	if err != nil {
		t.Fatal(err)
	}
	if restored.LastLobby == nil || restored.LastLobby.Status != "roster_parsed" || restored.Active == nil || *restored.Active != original {
		t.Fatalf("scanner changed prediction or lost observation: %+v", restored)
	}
	// A later corrupt file replaces the observation, never reuses the old roster.
	if err := os.WriteFile(path, []byte("bad lobby"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
	e.scanTick(context.Background(), now.Add(-15*time.Second), &lastError, &lastLog)
	restored, err = loadState(p)
	if err != nil {
		t.Fatal(err)
	}
	if restored.LastLobby == nil || restored.LastLobby.Status != "parse_error" || restored.LastLobby.Data != nil || restored.Active == nil || *restored.Active != original {
		t.Fatalf("bad lobby changed prediction or retained roster: %+v", restored)
	}
}

func FuzzBattleLobby(f *testing.F) {
	raw, _ := os.ReadFile("testdata/battlelobby/replay.server.ARAM1_89754.battlelobby")
	f.Add(raw)
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		got, err := parseBattleLobby(b)
		if err != nil && len(got.Players) != 0 {
			t.Fatal("partial roster on error")
		}
	})
}
