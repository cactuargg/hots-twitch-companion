package sessions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestartDedupAndLateReplayAcrossReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	s := Open(path)
	if err := s.Update(func(v *State) error {
		v.Preferences.Enabled = true
		if err := New(v, "p", 100); err != nil {
			return err
		}
		if err := New(v, "p", 200); err != nil {
			return err
		}
		Add(v, Match{ID: "late", Player: "p", StartedAt: 150, Result: "win"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	restored := Open(path)
	v, err := restored.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Matches) != 1 || v.Matches[0].SessionID != v.Sessions[0].ID {
		t.Fatalf("late replay assigned to reset session: %+v", v)
	}
	if Add(&v, Match{ID: "late", Player: "p", StartedAt: 250, Result: "loss"}) {
		t.Fatal("duplicate counted")
	}
	if w, l := Score(v, v.Sessions[0].ID); w != 1 || l != 0 {
		t.Fatal(w, l)
	}
	if w, l := Score(v, Current(&v, "p").ID); w != 0 || l != 0 {
		t.Fatal(w, l)
	}
	if err := Undo(&v, "p"); err != nil {
		t.Fatal(err)
	}
	if len(v.Sessions) != 1 || Current(&v, "p").EndedAt != 0 {
		t.Fatal("undo did not reopen previous session")
	}
}
func TestUnknownStartFiltersBoundaryAndPlayerIsolation(t *testing.T) {
	s := State{}
	_ = New(&s, "p", 100)
	_ = New(&s, "other", 100)
	Add(&s, Match{ID: "a", Player: "p", StartedAt: 120, Mode: 50091, Result: "win"})
	Add(&s, Match{ID: "b", Player: "p", StartedAt: 140, Mode: 50001, Result: "loss"})
	Add(&s, Match{ID: "offline", Player: "p", RecordedAt: 160, Result: "win"})
	if !s.Matches[2].NeedsAssignment || s.Matches[2].SessionID != "" {
		t.Fatal("guessed offline boundary")
	}
	s.Preferences.RankedOnly = true
	if w, l := Score(s, Current(&s, "p").ID); w != 1 || l != 0 {
		t.Fatal(w, l)
	}
	if w, l := Score(s, Current(&s, "other").ID); w != 0 || l != 0 {
		t.Fatal("player mixed")
	}
	if err := Boundary(&s, "p", "b"); err != nil {
		t.Fatal(err)
	}
	if s.Matches[0].SessionID == s.Matches[1].SessionID {
		t.Fatal("boundary failed")
	}
	if err := Boundary(&s, "p", "offline"); err == nil {
		t.Fatal("unknown start used as boundary")
	}
	if err := Undo(&s, "p"); err != nil {
		t.Fatal(err)
	}
	if s.Matches[0].SessionID != s.Matches[1].SessionID {
		t.Fatal("undo lost matches")
	}
}
func TestCorruptAndFailedSaveNeverOverwriteJournal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	_ = os.WriteFile(path, []byte("broken"), 0600)
	s := Open(path)
	if err := s.Update(func(v *State) error { return New(v, "p", 100) }); err == nil {
		t.Fatal("corruption overwritten")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "broken" {
		t.Fatal("journal changed")
	}
	path = filepath.Join(dir, "new.json")
	s = Open(path)
	_ = os.Mkdir(path+".new", 0700)
	if err := s.Update(func(v *State) error { return New(v, "p", 100) }); err == nil {
		t.Fatal("expected save failure")
	}
	v, _ := s.Snapshot()
	if len(v.Sessions) != 0 {
		t.Fatal("uncommitted state published")
	}
}
