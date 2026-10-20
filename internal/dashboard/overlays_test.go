package dashboard

import (
	"bytes"
	"encoding/json"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/hots/correlation"
	"hots-twitch-companion/internal/hots/discovery"
	"hots-twitch-companion/internal/hots/lobby"
	"hots-twitch-companion/internal/hots/replay"
	"hots-twitch-companion/internal/sessions"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type overlayApplication struct {
	Application
	c config.Data
}

func (a *overlayApplication) Config() config.Data { return a.c }
func (a *overlayApplication) StopWatch()          {}
func overlayDashboard(t *testing.T) *Dashboard {
	t.Helper()
	d := New(&overlayApplication{c: config.Data{PlayerToonHandle: "2-Hero-1-679757", DisableUpdateChecks: true}}, config.NewPaths(t.TempDir()), "test")
	d.overlays.findLobby = func() (string, string, time.Time, error) { return "", "", time.Time{}, discovery.ErrLobbyNotFound }
	if err := d.overlays.store.Update(func(s *sessions.State) error { s.Token = "test-token"; return nil }); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(d.Close)
	return d
}
func TestOverlayReadOnlyBoundaryAndStableToken(t *testing.T) {
	d := overlayDashboard(t)
	handler := d.overlayHandler()
	for _, tc := range []struct {
		method, path, host string
		code               int
	}{{"GET", "/test-token/score", overlayAddress, 200}, {"GET", "/test-token/mask", overlayAddress, 200}, {"GET", "/test-token/state", overlayAddress, 200}, {"GET", "/test-token/api/config", overlayAddress, 404}, {"POST", "/test-token/state", overlayAddress, 405}, {"GET", "/wrong/state", overlayAddress, 404}, {"GET", "/test-token/state", "evil.test", 403}} {
		r := httptest.NewRequest(tc.method, "http://"+tc.host+tc.path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.code {
			t.Fatalf("%+v: %d %s", tc, w.Code, w.Body.String())
		}
		if strings.HasSuffix(tc.path, "state") && w.Code == 200 {
			for _, secret := range []string{"sessions", "matches", "test-token", "player", "replayRoot"} {
				if strings.Contains(w.Body.String(), `"`+secret+`"`) {
					t.Fatalf("management data leaked: %s", w.Body.String())
				}
			}
		}
	}
	reopened := sessions.Open(filepath.Join(d.p.Dir, "sessions.json"))
	s, err := reopened.Snapshot()
	if err != nil || s.Token != "test-token" {
		t.Fatal("URL token lost on restart")
	}
}
func TestMaskFailClosedAndManualControls(t *testing.T) {
	d := overlayDashboard(t)
	o := d.overlays
	if err := o.store.Update(func(s *sessions.State) error { s.Preferences.MaskEnabled = true; return nil }); err != nil {
		t.Fatal(err)
	}
	check := func(want bool) {
		t.Helper()
		if got := d.overlaySnapshot(false)["maskVisible"]; got != want {
			t.Fatalf("mask=%v want %v", got, want)
		}
	}
	check(true)
	o.lastPlayer = playerKey(d.Config())
	o.phase = "loading_or_game"
	o.observedAt = sessions.Now()
	o.checkedAt = sessions.Now()
	check(false)
	o.checkedAt = sessions.Now() - 20000
	check(true)
	_ = o.store.Update(func(s *sessions.State) error { s.Preferences.MaskMode = "hide"; return nil })
	check(false)
	_ = o.store.Update(func(s *sessions.State) error { s.Preferences.MaskMode = "show"; return nil })
	check(true)
	_ = o.store.Update(func(s *sessions.State) error { s.Preferences.MaskEnabled = false; return nil })
	check(false)
}
func TestRealReplayJournalRecoveryCopiesAndManualAssignment(t *testing.T) {
	d := overlayDashboard(t)
	raw, err := os.ReadFile("../../testdata/battlelobby/HanamuraSL1_89566.StormReplay")
	if err != nil {
		t.Fatal(err)
	}
	detailsBytes, err := replay.ReadMPQMember(raw, "replay.details")
	if err != nil {
		t.Fatal(err)
	}
	details, err := replay.ParseDetails(detailsBytes)
	if err != nil {
		t.Fatal(err)
	}
	a := d.engine.(*overlayApplication)
	a.c.PlayerToonHandle = details.Players[0].ToonHandle
	a.c.ReplayRoot = t.TempDir()
	player := playerKey(a.c)
	for _, name := range []string{"a.StormReplay", "copy.StormReplay"} {
		path := filepath.Join(a.c.ReplayRoot, name)
		_ = os.WriteFile(path, raw, 0600)
		at := time.Now().Add(-20 * time.Second)
		_ = os.Chtimes(path, at, at)
	}
	if err := d.overlays.store.Update(func(s *sessions.State) error {
		s.Preferences.Enabled = true
		return sessions.New(s, player, details.RecordedAt-1)
	}); err != nil {
		t.Fatal(err)
	}
	d.scanOverlayMatches()
	s, err := d.overlays.store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Matches) != 1 || !s.Matches[0].NeedsAssignment {
		t.Fatalf("offline copies not pending/deduplicated: %+v", s.Matches)
	}
	body, _ := json.Marshal(map[string]any{"matchId": correlation.ContentSHA(raw), "sessionId": s.Sessions[0].ID})
	w := httptest.NewRecorder()
	d.handleAPIRoute(w, httptest.NewRequest("POST", "/", bytes.NewReader(body)), "sessions/match")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	d.scanOverlayMatches()
	s, _ = d.overlays.store.Snapshot()
	if len(s.Matches) != 1 || s.Matches[0].NeedsAssignment {
		t.Fatal("duplicate scan lost assignment")
	}
	wins, losses := sessions.Score(s, s.Sessions[0].ID)
	if wins+losses != 1 {
		t.Fatal(wins, losses)
	}
	// A copied old replay is not imported into a newly enabled journal.
	newer := overlayDashboard(t)
	newer.engine = a
	_ = newer.overlays.store.Update(func(s *sessions.State) error {
		s.Preferences.Enabled = true
		return sessions.New(s, player, sessions.Now())
	})
	newer.scanOverlayMatches()
	n, _ := newer.overlays.store.Snapshot()
	if len(n.Matches) != 0 {
		t.Fatal("old replay imported")
	}
}
func TestPreferencesRejectInvalidImageAndSessionCrossPlayer(t *testing.T) {
	d := overlayDashboard(t)
	s, _ := d.overlays.store.Snapshot()
	p := s.Preferences
	p.MaskImage = "data:image/svg+xml;base64,PHN2Zz4="
	if validPreferences(p) == nil {
		t.Fatal("SVG accepted")
	}
	p.MaskImage = "data:image/png;base64,YWJj"
	if validPreferences(p) == nil {
		t.Fatal("fake PNG accepted")
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"preferences":{},"extra":true}`))
	w := httptest.NewRecorder()
	d.handleAPIRoute(w, r, "sessions/preferences")
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}

func TestFreshLobbyLoadsMaskAndReplayRestoresIt(t *testing.T) {
	d := overlayDashboard(t)
	raw, err := os.ReadFile("../../testdata/battlelobby/HanamuraSL1_89566.StormReplay")
	if err != nil {
		t.Fatal(err)
	}
	member, _ := replay.ReadMPQMember(raw, "replay.server.battlelobby")
	l, err := lobby.Parse(member)
	if err != nil {
		t.Fatal(err)
	}
	detailsBytes, _ := replay.ReadMPQMember(raw, "replay.details")
	details, _ := replay.ParseDetails(detailsBytes)
	a := d.engine.(*overlayApplication)
	a.c.PlayerToonHandle = details.Players[0].ToonHandle
	a.c.ReplayRoot = t.TempDir()
	lobbyPath := filepath.Join(t.TempDir(), "battlelobby")
	_ = os.WriteFile(lobbyPath, member, 0600)
	at := time.Now().Add(-20 * time.Second)
	_ = os.Chtimes(lobbyPath, at, at)
	d.overlays.findLobby = func() (string, string, time.Time, error) { return lobbyPath, "key", at, nil }
	_ = d.overlays.store.Update(func(s *sessions.State) error {
		s.Preferences.MaskEnabled = true
		s.Preferences.MaskRankedOnly = l.AMMID == 50091
		return nil
	})
	d.scanOverlayMatches()
	if d.overlaySnapshot(false)["maskVisible"] != false {
		t.Fatal("fresh lobby did not hide mask")
	}
	// Rewriting the same content with a different observation time is ambiguous.
	at = at.Add(time.Second)
	_ = os.Chtimes(lobbyPath, at, at)
	d.scanOverlayMatches()
	if d.overlaySnapshot(false)["maskVisible"] != true {
		t.Fatal("ambiguous lobby did not fail closed")
	}
	s, _ := d.overlays.store.Snapshot()
	if len(s.Observations) != 1 || !s.Observations[0].Ambiguous {
		t.Fatal("ambiguity not persisted")
	}
	// A completed replay returns the mask independently of Twitch.
	replayPath := filepath.Join(a.c.ReplayRoot, "match.StormReplay")
	_ = os.WriteFile(replayPath, raw, 0600)
	old := time.Now().Add(-10 * time.Second)
	_ = os.Chtimes(replayPath, old, old)
	d.scanOverlayMatches()
	if d.overlays.phase != "between_games" || d.overlaySnapshot(false)["maskVisible"] != true {
		t.Fatal("replay did not restore mask")
	}
}
