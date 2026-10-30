package replaystats

import (
	"hots-twitch-companion/internal/hots/replay"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func fixture(id string, mode uint32, at int64, name string) Record {
	return Record{ID: id, Mode: mode, Replay: replay.Replay{RecordedAt: at, Players: []replay.Player{
		{Name: name, ToonHandle: "2-Hero-1-1", Hero: "Jaina", Result: "win"},
		{Name: "Same name", ToonHandle: "2-Hero-1-2", Hero: "Jaina", Result: "loss"},
		{Name: "Unknown", ToonHandle: "2-Hero-1-3", Hero: "Valla", Result: "unknown"},
		{Name: "AI", ToonHandle: "0--0-0", Hero: "Valla", Result: "win"},
	}}}
}
func TestStatisticsDeduplicatePersistAndKeepStableIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "replay-statistics.json")
	s := Open(path)
	a := fixture("a", 50091, 100, "Same name")
	b := fixture("b", 50101, 200, "Renamed")
	n, err := s.Add([]Record{a, a, b})
	if err != nil || n != 2 {
		t.Fatalf("import: %d %v", n, err)
	}
	n, err = s.Add([]Record{a})
	if err != nil || n != 0 {
		t.Fatalf("reimport: %d %v", n, err)
	}
	all, err := s.Summary("all", "")
	if err != nil || all.Matches != 2 || all.Observations != 4 || all.Skipped != 4 || len(all.Heroes) != 1 || all.Heroes[0].Games != 4 || all.Heroes[0].WinRate != 50 || len(all.Players) != 2 {
		t.Fatalf("unexpected summary: %+v %v", all, err)
	}
	personal, _ := s.Summary("all", "2-Hero-1-1")
	if personal.Heroes[0].Wins != 2 || personal.Heroes[0].WinRate != 100 {
		t.Fatalf("player hero: %+v", personal)
	}
	ranked, _ := s.Summary("50091", "")
	if ranked.Matches != 1 || ranked.Heroes[0].Games != 2 {
		t.Fatalf("mode mixed: %+v", ranked)
	}
	found := false
	for _, p := range all.Players {
		if p.Key == "2-hero-1-1" {
			found = p.Name == "Renamed"
		}
	}
	if !found {
		t.Fatal("latest nickname lost or identical names merged")
	}
	restored := Open(path)
	got, err := restored.Summary("all", "")
	if err != nil || !reflect.DeepEqual(got, all) {
		t.Fatal("restart changed statistics", err)
	}
	// Returned values and Add inputs must never alias stored observations/cache.
	all.Heroes[0].Wins = 999
	a.Replay.Players[0].Name = "mutated"
	r, _ := s.Get("a")
	r.Replay.Players[0].Hero = "mutated"
	again, _ := s.Summary("all", "")
	original, _ := s.Get("a")
	if again.Heroes[0].Wins != 2 || original.Replay.Players[0].Hero != "Jaina" || original.Replay.Players[0].Name != "Same name" {
		t.Fatal("caller mutated store")
	}
	c := fixture("c", 50091, 300, "Renamed")
	if _, err := s.Add([]Record{c}); err != nil {
		t.Fatal(err)
	}
	newer, _ := s.Summary("all", "")
	if newer.Matches != 3 || newer.Heroes[0].Games != 6 {
		t.Fatal("cached statistics did not update")
	}
}
func TestDuplicateIdentityAndUnknownHeroesDoNotInventResults(t *testing.T) {
	s := Open(filepath.Join(t.TempDir(), "stats.json"))
	r := fixture("a", 50091, 100, "Player")
	r.Replay.Players = append(r.Replay.Players, replay.Player{ToonHandle: "2-Hero-1-1", Hero: "Valla", Result: "loss"}, replay.Player{Name: "No hero", ToonHandle: "2-Hero-1-4", Result: "win"})
	if _, err := s.Add([]Record{r}); err != nil {
		t.Fatal(err)
	}
	summary, _ := s.Summary("all", "")
	if summary.Observations != 2 || summary.Skipped != 4 || summary.Heroes[0].Games != 1 || summary.Heroes[0].Wins != 0 {
		t.Fatalf("invented results: %+v", summary)
	}
}
func TestCorruptStatisticsAndSaveFailureDoNotOverwriteData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.json")
	raw := []byte(`{"version":`)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s := Open(path)
	if _, err := s.Add([]Record{fixture("a", 50091, 100, "Player")}); err == nil {
		t.Fatal("corruption overwritten")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(raw) {
		t.Fatal("original corrupt database changed")
	}
	if _, err := s.Summary("all", ""); err == nil {
		t.Fatal("corruption hidden")
	}
	target := filepath.Join(dir, "blocked")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "child"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	s = Open(filepath.Join(dir, "missing.json"))
	s.path = target
	if _, err := s.Add([]Record{fixture("a", 50091, 100, "Player")}); err == nil {
		t.Fatal("expected persistence failure")
	}
	summary, _ := s.Summary("all", "")
	if summary.Matches != 0 {
		t.Fatal("failed save committed in memory")
	}
}
