package replaystats

import (
	"fmt"
	"hots-twitch-companion/internal/hots/lobby"
	"hots-twitch-companion/internal/hots/replay"
	"path/filepath"
	"testing"
)

func estimateFixture(t *testing.T) (*Store, *lobby.BattleLobby) {
	t.Helper()
	s := Open(filepath.Join(t.TempDir(), "stats.json"))
	b := &lobby.BattleLobby{AMMID: 50091, TeamSize: "5v5"}
	for i := 0; i < 10; i++ {
		team := "T1"
		if i >= 5 {
			team = "T2"
		}
		b.Players = append(b.Players, lobby.LobbyPlayer{Slot: i, SlotType: "Humn", Role: "Part", Team: team, Hero: "Jain", ToonHandle: fmt.Sprintf("1-Hero-1-%d", i+1)})
	}
	records := []Record{}
	for n := 0; n < 30; n++ {
		r := Record{ID: fmt.Sprint(n), Mode: 50091, Replay: replay.Replay{RecordedAt: int64(n + 1)}}
		for i, p := range b.Players {
			result := "win"
			if i >= 5 {
				result = "loss"
			}
			r.Replay.Players = append(r.Replay.Players, replay.Player{ToonHandle: p.ToonHandle, Hero: "Jaina", Team: int64(i / 5), Result: result})
		}
		records = append(records, r)
	}
	if _, err := s.Add(records); err != nil {
		t.Fatal(err)
	}
	return s, b
}

func TestEstimateUsesTeamsAndConservativePlayerHistory(t *testing.T) {
	s, b := estimateFixture(t)
	e := s.EstimateLobby(b, b.Players[0].ToonHandle, 100)
	if e == nil || e.Percent <= 50 || e.Percent > 80 || e.Coverage != 10 || e.Matches != 30 {
		t.Fatalf("wrong estimate: %+v", e)
	}
	opposite := s.EstimateLobby(b, b.Players[9].ToonHandle, 100)
	if opposite == nil || opposite.Percent+e.Percent != 100 {
		t.Fatalf("teams not symmetric: %+v %+v", e, opposite)
	}
	// Reordering slots cannot change the inferred team or probability.
	b.Players[0], b.Players[9] = b.Players[9], b.Players[0]
	if got := s.EstimateLobby(b, b.Players[9].ToonHandle, 100); got == nil || got.Percent != e.Percent {
		t.Fatal(got)
	}
}

func TestEstimateRejectsInsufficientAmbiguousAndFutureData(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*lobby.BattleLobby)
		before int64
	}{
		{"before history", func(*lobby.BattleLobby) {}, 1},
		{"too few matches", func(*lobby.BattleLobby) {}, 20},
		{"other mode", func(b *lobby.BattleLobby) { b.AMMID = 50001 }, 100},
		{"no team", func(b *lobby.BattleLobby) { b.Players[0].Team = "" }, 100},
		{"ai", func(b *lobby.BattleLobby) { b.Players[0].SlotType = "Comp" }, 100},
		{"duplicate", func(b *lobby.BattleLobby) { b.Players[0].ToonHandle = b.Players[1].ToonHandle }, 100},
		{"unknown roster", func(b *lobby.BattleLobby) {
			for i := range b.Players {
				b.Players[i].ToonHandle = fmt.Sprintf("1-Hero-1-%d", 100+i)
				b.Players[i].Hero = ""
			}
		}, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, b := estimateFixture(t)
			tc.change(b)
			if got := s.EstimateLobby(b, b.Players[0].ToonHandle, tc.before); got != nil {
				t.Fatalf("unsafe estimate: %+v", got)
			}
		})
	}
}

func TestEstimateARAMIgnoresUnconfirmedHeroes(t *testing.T) {
	s, b := estimateFixture(t)
	s.mu.Lock()
	for i := range s.data.Matches {
		s.data.Matches[i].Mode = 50101
	}
	s.mu.Unlock()
	b.AMMID = 50101
	a := s.EstimateLobby(b, b.Players[0].ToonHandle, 100)
	for i := range b.Players {
		b.Players[i].Hero = "Demo"
	}
	z := s.EstimateLobby(b, b.Players[0].ToonHandle, 100)
	if a == nil || z == nil || a.Percent != z.Percent {
		t.Fatalf("ARAM hero affected estimate: %+v %+v", a, z)
	}
}

func TestEstimateHeroHistoryCanCoverUnknownPeople(t *testing.T) {
	s, b := estimateFixture(t)
	s.mu.Lock()
	for n := range s.data.Matches {
		for i := range s.data.Matches[n].Replay.Players {
			h := "Muradin"
			if i >= 5 {
				h = "Valla"
			}
			s.data.Matches[n].Replay.Players[i].Hero = h
		}
	}
	s.mu.Unlock()
	for i := range b.Players {
		b.Players[i].ToonHandle = fmt.Sprintf("1-Hero-1-%d", 100+i)
		b.Players[i].Hero = "Mura"
		if i >= 5 {
			b.Players[i].Hero = "Demo"
		}
	}
	e := s.EstimateLobby(b, b.Players[0].ToonHandle, 100)
	if e == nil || e.Percent <= 50 || e.Coverage != 10 {
		t.Fatalf("hero data not used: %+v", e)
	}
	for i := range b.Players {
		b.Players[i].Hero = ""
	}
	if s.EstimateLobby(b, b.Players[0].ToonHandle, 100) != nil {
		t.Fatal("unknown people and heroes produced estimate")
	}
}

func TestEstimateRejectsInconsistentHistoricalTeams(t *testing.T) {
	s, b := estimateFixture(t)
	s.mu.Lock()
	for n := range s.data.Matches {
		s.data.Matches[n].Replay.Players[0].Result = "loss"
	}
	s.mu.Unlock()
	if got := s.EstimateLobby(b, b.Players[0].ToonHandle, 100); got != nil {
		t.Fatal(got)
	}
}
