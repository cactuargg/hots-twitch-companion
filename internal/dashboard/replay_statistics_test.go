package dashboard

import (
	"encoding/json"
	"hots-twitch-companion/internal/replaystats"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalStatisticsImportAllParticipantsWithoutChangingHistory(t *testing.T) {
	d := overlayDashboard(t)
	a := d.engine.(*overlayApplication)
	a.c.PlayerToonHandle = "2-Hero-1-999999999"
	a.c.ReplayRoot = t.TempDir()
	raw, err := os.ReadFile("../../testdata/battlelobby/HanamuraSL1_89566.StormReplay")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one.StormReplay", "copy.StormReplay"} {
		file := filepath.Join(a.c.ReplayRoot, name)
		if err := os.WriteFile(file, raw, 0600); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-time.Minute)
		_ = os.Chtimes(file, at, at)
	}
	d.importHistory(a.c)
	summary, err := d.statistics.Summary("all", "")
	if err != nil || summary.Matches != 1 || summary.Observations != 10 || len(summary.Players) != 10 {
		t.Fatalf("not all players indexed: %+v %v", summary, err)
	}
	state, _ := d.overlays.store.Snapshot()
	if len(state.Matches) != 0 || len(state.Sessions) != 0 {
		t.Fatal("statistics import changed personal history")
	}
	d.importHistory(a.c)
	again, _ := d.statistics.Summary("all", "")
	if again.Matches != 1 || again.Observations != 10 {
		t.Fatal("duplicate observations")
	}
	persisted := replaystats.Open(filepath.Join(d.p.Dir, "replay-statistics.json"))
	restored, err := persisted.Summary("all", "")
	if err != nil || restored.Observations != 10 {
		t.Fatal("statistics lost on restart", err)
	}
	w := httptest.NewRecorder()
	d.statisticsAPI(w, httptest.NewRequest("GET", "/api/statistics?kind=players&mode=50091", nil))
	var response struct {
		Rows  []replaystats.Row `json:"rows"`
		Total int               `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || w.Code != 200 || response.Total != 10 || len(response.Rows) != 10 {
		t.Fatalf("statistics API: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	d.statisticsAPI(w, httptest.NewRequest("GET", "/api/statistics?kind=heroes&mode=50091&player="+summary.Players[0].Key, nil))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	for _, query := range []string{"mode=oops", "mode=12345", "kind=invalid"} {
		w = httptest.NewRecorder()
		d.statisticsAPI(w, httptest.NewRequest("GET", "/api/statistics?"+query, nil))
		if w.Code != 400 {
			t.Fatal("invalid filter accepted", query)
		}
	}
	// Statistics are private application data, not available on OBS's read-only server.
	w = httptest.NewRecorder()
	d.overlayHandler().ServeHTTP(w, httptest.NewRequest("GET", "http://"+overlayAddress+"/test-token/statistics", nil))
	if w.Code != 404 {
		t.Fatal("statistics exposed to OBS")
	}
}
func TestStatisticsCollectedWithCounterAndMaskDisabled(t *testing.T) {
	d := overlayDashboard(t)
	a := d.engine.(*overlayApplication)
	a.c.PlayerToonHandle = "2-Hero-1-999999999"
	a.c.ReplayRoot = t.TempDir()
	raw, err := os.ReadFile("../../testdata/battlelobby/HanamuraSL1_89566.StormReplay")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(a.c.ReplayRoot, "match.StormReplay")
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(-time.Minute)
	_ = os.Chtimes(file, at, at)
	d.scanOverlayMatches()
	s, err := d.statistics.Summary("all", "")
	if err != nil || s.Observations != 10 {
		t.Fatalf("collection disabled with counter: %+v %v", s, err)
	}
	d.scanOverlayMatches()
	s, _ = d.statistics.Summary("all", "")
	if s.Matches != 1 {
		t.Fatal("poll duplicated observations")
	}
}
