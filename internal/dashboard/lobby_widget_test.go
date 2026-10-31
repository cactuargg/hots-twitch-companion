package dashboard

import (
	"encoding/json"
	"hots-twitch-companion/internal/hots/lobby"
	"hots-twitch-companion/internal/hots/replay"
	"hots-twitch-companion/internal/replaystats"
	"hots-twitch-companion/internal/sessions"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func widgetFixture(t *testing.T) *Dashboard {
	t.Helper()
	d := overlayDashboard(t)
	raw, err := os.ReadFile("../../testdata/battlelobby/HanamuraSL1_89566.StormReplay")
	if err != nil {
		t.Fatal(err)
	}
	b, err := replay.ReadMPQMember(raw, "replay.server.battlelobby")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "replay.server.battlelobby")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	snap := lobby.Capture(path)
	if snap.Status != "roster_parsed" {
		t.Fatal(snap.Error)
	}
	a := d.engine.(*overlayApplication)
	a.c.PlayerName = "Streamer"
	a.c.PlayerToonHandle = snap.Data.Players[0].ToonHandle
	o := d.overlays
	o.widgetLobby = snap
	o.widgetPath = path
	o.phase = "loading_or_game"
	o.lastPlayer = strings.ToLower(a.c.PlayerToonHandle)
	o.checkedAt = time.Now().UnixMilli()
	o.observedAt = snap.ModifiedAt
	if err := o.store.Update(func(s *sessions.State) error { s.Preferences.LobbyWidgetEnabled = true; return nil }); err != nil {
		t.Fatal(err)
	}
	return d
}
func seedWidgetHistory(t *testing.T, d *Dashboard) {
	t.Helper()
	roster := d.overlays.widgetLobby.Data.Players
	var records []replaystats.Record
	for i := 0; i < 3; i++ {
		mode := uint32(50091)
		if i == 2 {
			mode = 50001
		}
		result := "win"
		if i == 1 {
			result = "loss"
		}
		records = append(records, replaystats.Record{ID: string(rune('a' + i)), Mode: mode, Replay: replay.Replay{RecordedAt: 100 + int64(i), Players: []replay.Player{
			{Name: "Streamer", ToonHandle: roster[0].ToonHandle, Hero: lobby.HeroName(roster[0].Hero, "en"), Result: result},
			{Name: "Known <player>", ToonHandle: roster[1].ToonHandle, Hero: lobby.HeroName(roster[1].Hero, "en"), Result: result},
		}}})
	}
	if _, err := d.statistics.Add(records); err != nil {
		t.Fatal(err)
	}
}
func TestLobbyWidgetUsesCurrentHeroModeAndKnownPlayers(t *testing.T) {
	d := widgetFixture(t)
	seedWidgetHistory(t, d)
	s := d.lobbyWidget()
	if s.Phase != "live" || s.HeroPending || s.Streamer.Hero == "" || s.Streamer.OnHero == nil || s.Streamer.OnHero.Games != 2 || s.Streamer.OnHero.WinRate == nil || *s.Streamer.OnHero.WinRate != 50 || len(s.Familiar) != 1 || s.Familiar[0].Meetings != 2 || s.Familiar[0].Overall.Games != 2 {
		t.Fatalf("wrong lobby statistics: %+v", s)
	}
	w := httptest.NewRecorder()
	d.overlayHandler().ServeHTTP(w, httptest.NewRequest("GET", "http://"+overlayAddress+"/test-token/lobby-state", nil))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, secret := range []string{d.overlays.widgetLobby.SHA256, d.overlays.widgetPath, d.overlays.widgetLobby.Data.Players[0].ToonHandle, "sessions", "replays", "playerToonHandle"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatalf("private data exposed: %s", secret)
		}
	}
	for _, route := range []string{"lobby", "lobby-widget.js", "lobby-widget.css"} {
		w = httptest.NewRecorder()
		d.overlayHandler().ServeHTTP(w, httptest.NewRequest("GET", "http://"+overlayAddress+"/test-token/"+route, nil))
		if w.Code != 200 || w.Body.Len() == 0 {
			t.Fatal("missing OBS asset", route, w.Code)
		}
	}
	w = httptest.NewRecorder()
	d.overlayHandler().ServeHTTP(w, httptest.NewRequest("POST", "http://"+overlayAddress+"/test-token/lobby-state", nil))
	if w.Code != 405 {
		t.Fatal("OBS accepted mutation")
	}
}
func TestLobbyWidgetNeverKeepsPreviousMatchOrProfile(t *testing.T) {
	for _, reason := range []string{"stale", "different player", "missing file", "changed file", "completed", "disabled", "ambiguous"} {
		t.Run(reason, func(t *testing.T) {
			d := widgetFixture(t)
			seedWidgetHistory(t, d)
			_ = d.lobbyWidget()
			switch reason {
			case "stale":
				d.overlays.checkedAt = time.Now().Add(-time.Minute).UnixMilli()
			case "different player":
				d.engine.(*overlayApplication).c.PlayerToonHandle = "2-Hero-1-999999"
			case "missing file":
				if err := os.Remove(d.overlays.widgetPath); err != nil {
					t.Fatal(err)
				}
			case "changed file":
				if err := os.WriteFile(d.overlays.widgetPath, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "completed":
				if _, err := d.statistics.Add([]replaystats.Record{{ID: "completed", LobbyHash: d.overlays.widgetLobby.SHA256, Replay: replay.Replay{RecordedAt: 100}}}); err != nil {
					t.Fatal(err)
				}
			case "disabled":
				_ = d.overlays.store.Update(func(s *sessions.State) error { s.Preferences.LobbyWidgetEnabled = false; return nil })
			case "ambiguous":
				d.overlays.phase = "unknown"
			}
			s := d.lobbyWidget()
			if s.Phase == "live" || s.Streamer.Hero != "" || len(s.Familiar) != 0 {
				t.Fatalf("stale lobby displayed for %s: %+v", reason, s)
			}
		})
	}
}
func TestLobbyWidgetUnknownHeroAndEmptyHistoryHaveNoInventedRate(t *testing.T) {
	d := widgetFixture(t)
	s := d.lobbyWidget()
	if s.Streamer.OnHero == nil || s.Streamer.OnHero.Games != 0 || s.Streamer.OnHero.WinRate != nil || len(s.Familiar) != 0 {
		t.Fatalf("invented history: %+v", s)
	}
	d = widgetFixture(t)
	seedWidgetHistory(t, d)
	d.overlays.widgetLobby.Data.AMMID = 50101
	s = d.lobbyWidget()
	if !s.HeroPending || s.Streamer.Hero != "" || s.Streamer.OnHero != nil {
		t.Fatalf("ARAM preselection treated as actual hero: %+v", s)
	}
}
func TestWidgetPreviewOnlyAllowsSameOriginFraming(t *testing.T) {
	d := widgetFixture(t)
	h := d.Routes("127.0.0.1:56789", "secret")
	for _, route := range []string{"assets/lobby-widget.html", "", "assets/lobby-widget.js"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:56789/secret/"+route, nil))
		want := "DENY"
		if route == "assets/lobby-widget.html" {
			want = "SAMEORIGIN"
		}
		if w.Code != 200 || w.Header().Get("X-Frame-Options") != want {
			t.Fatal("unexpected framing policy", route, w.Code, w.Header())
		}
	}
	b, _ := json.Marshal(d.overlaySnapshot(true))
	if !strings.Contains(string(b), "/test-token/lobby") {
		t.Fatal("widget URL missing from settings")
	}
}

func TestLobbyHeroNamesAgreeWithCompletedNonARAMFixtures(t *testing.T) {
	files, err := filepath.Glob("../../testdata/battlelobby/*.StormReplay")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		member, err := replay.ReadMPQMember(raw, "replay.server.battlelobby")
		if err != nil {
			continue
		}
		roster, err := lobby.Parse(member)
		if err != nil || roster.AMMID == 50101 {
			continue
		}
		member, err = replay.ReadMPQMember(raw, "replay.details")
		if err != nil {
			t.Fatal(err)
		}
		details, err := replay.ParseDetails(member)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range roster.Players {
			if p.SlotType != "Humn" || p.Role != "Part" || p.Hero == "" {
				continue
			}
			for _, actual := range details.Players {
				if actual.ToonHandle == p.ToonHandle && actual.Hero != "" {
					wantTeam := "T1"
					if actual.Team == 1 {
						wantTeam = "T2"
					}
					if p.Team != wantTeam {
						t.Fatalf("team mismatch: slot %d has %q, replay team %d", p.Slot, p.Team, actual.Team)
					}
					if !lobby.HeroMatches(p.Hero, actual.Hero) {
						t.Fatalf("hero mismatch in %s: code %q vs actual %q", filepath.Base(file), p.Hero, actual.Hero)
					}
					checked++
				}
			}
		}
	}
	if checked < 10 {
		t.Fatal("insufficient real-fixture hero coverage", checked)
	}
}
