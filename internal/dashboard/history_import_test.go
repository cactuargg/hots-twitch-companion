package dashboard

import (
	"hots-twitch-companion/internal/hots/replay"
	"hots-twitch-companion/internal/sessions"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistoryImportDeduplicatesAndPreservesSessionScore(t *testing.T) {
	d := overlayDashboard(t)
	raw, err := os.ReadFile("../../testdata/battlelobby/HanamuraSL1_89566.StormReplay")
	if err != nil {
		t.Fatal(err)
	}
	member, err := replay.ReadMPQMember(raw, "replay.details")
	if err != nil {
		t.Fatal(err)
	}
	details, err := replay.ParseDetails(member)
	if err != nil {
		t.Fatal(err)
	}
	a := d.engine.(*overlayApplication)
	a.c.PlayerToonHandle = details.Players[0].ToonHandle
	a.c.ReplayRoot = t.TempDir()
	player := playerKey(a.c)
	for _, name := range []string{"match.StormReplay", "copy.StormReplay"} {
		p := filepath.Join(a.c.ReplayRoot, name)
		if err := os.WriteFile(p, raw, 0600); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-20 * time.Second)
		_ = os.Chtimes(p, at, at)
	}
	if err := d.overlays.store.Update(func(s *sessions.State) error {
		s.Preferences.Enabled = true
		if err := sessions.New(s, player, sessions.Now()-100); err != nil {
			return err
		}
		sessions.Add(s, sessions.Match{ID: "existing", Player: player, StartedAt: sessions.Now(), Result: "win", OriginalResult: "win", Manual: true})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	d.importHistory(a.c)
	s, err := d.overlays.store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Matches) != 2 {
		t.Fatalf("want one existing and one imported: %+v", s.Matches)
	}
	m := s.Matches[1]
	if !m.Historical || m.NeedsAssignment || m.SessionID != "" || m.Hero == "" || m.Map == "" || len(m.Players) != 10 {
		t.Fatalf("bad imported match: %+v", m)
	}
	if wins, losses := sessions.Score(s, sessions.Current(&s, player).ID); wins != 1 || losses != 0 {
		t.Fatalf("import changed score: %d-%d", wins, losses)
	}
	d.importHistory(a.c)
	s, _ = d.overlays.store.Snapshot()
	if len(s.Matches) != 2 || !s.Matches[0].Manual || s.Matches[0].Result != "win" {
		t.Fatal("reimport duplicated or overwrote history")
	}
	for _, name := range []string{"match.StormReplay", "copy.StormReplay"} {
		b, err := os.ReadFile(filepath.Join(a.c.ReplayRoot, name))
		if err != nil || string(b) != string(raw) {
			t.Fatal("import modified a replay")
		}
	}
}
