package sessions

import "testing"

func TestReimportRepairsLobbyHeroWithoutChangingUserDecisions(t *testing.T) {
	s := State{Matches: []Match{{ID: "match", Player: "p", SessionID: "chosen", Hero: "Nova", Map: "Lost Cavern", Result: "loss", OriginalResult: "win", Manual: true, Excluded: true}}}
	added := Add(&s, Match{ID: "match", Player: "p", Hero: "Jaina", HeroSource: "replay.details", Historical: true, Result: "win"})
	if added || len(s.Matches) != 1 {
		t.Fatal("repair created a duplicate")
	}
	m := s.Matches[0]
	if m.Hero != "Jaina" || m.HeroSource != "replay.details" || m.Result != "loss" || !m.Manual || !m.Excluded || m.SessionID != "chosen" || m.Map != "Lost Cavern" {
		t.Fatalf("repair changed user decisions: %+v", m)
	}
	// An unknown played hero must clear a misleading lobby hero, never fall back.
	Add(&s, Match{ID: "match", Player: "p", HeroSource: "replay.details", Historical: true})
	if s.Matches[0].Hero != "" {
		t.Fatal("stale lobby hero retained when replay hero is unknown")
	}
}

func TestReimportRosterHeroesUsesHandles(t *testing.T) {
	s := State{Matches: []Match{{ID: "m", Player: "p", Manual: true, Result: "loss", SessionID: "chosen", Players: []Participant{{Name: "Alex", ToonHandle: "one", Result: "win"}, {Name: "Alex", ToonHandle: "two", Result: "loss"}}}}}
	Add(&s, Match{ID: "m", Player: "p", HeroSource: "replay.details", Players: []Participant{{Name: "Alex", ToonHandle: "two", Hero: "Jaina"}, {Name: "Alex", ToonHandle: "one", Hero: "Nova"}}})
	m := s.Matches[0]
	if len(s.Matches) != 1 || m.Players[0].Hero != "Nova" || m.Players[1].Hero != "Jaina" || m.Players[0].Result != "win" || m.Result != "loss" || m.SessionID != "chosen" || !m.Manual {
		t.Fatalf("incorrect roster repair: %+v", m)
	}
}
