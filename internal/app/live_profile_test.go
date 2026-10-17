package app

import (
	"bytes"
	"hots-twitch-companion/internal/config"
	dashview "hots-twitch-companion/internal/dashboard"
	predmodel "hots-twitch-companion/internal/prediction"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestLiveProfileMigrationDropsOnlyLegacyDryRun(t *testing.T) {
	for _, dry := range []bool{false, true} {
		d := manualDashboard(t, config.Data{})
		if err := saveState(d.p, predmodel.State{Active: &predmodel.Prediction{ID: "saved", DryRun: dry}, LastCreatedLobbySHA: "keep"}); err != nil {
			t.Fatal(err)
		}
		before, _ := os.ReadFile(d.p.State)
		cfg, err := config.Read(d.p)
		if err != nil || !cfg.RealMode {
			t.Fatal("profile must use Twitch")
		}
		if err := migrateLiveProfile(d.p, cfg); err != nil {
			t.Fatal(err)
		}
		s, err := loadState(d.p)
		if err != nil {
			t.Fatal(err)
		}
		if s.LastCreatedLobbySHA != "keep" {
			t.Fatal("consumed lobby lost")
		}
		if dry && s.Active != nil {
			t.Fatal("legacy simulation retained")
		}
		if !dry {
			after, _ := os.ReadFile(d.p.State)
			if !bytes.Equal(before, after) {
				t.Fatal("real prediction state changed")
			}
		}
	}
}

func TestStartRequiresExplicitConsentAndConnectedTwitch(t *testing.T) {
	d := manualDashboard(t, config.Data{RealMode: true, PlayerName: "Tester", ClientID: "public"})
	h, _ := d.nativeAssetHandler()
	for _, body := range []string{`{}`, `{"confirmedRealMode":false}`, `{"confirmedRealMode":true}`} {
		r := httptest.NewRequest("POST", "http://"+dashview.DesktopHost+"/api/automation/start", strings.NewReader(body))
		r.Header.Set("X-Companion", "1")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 || d.engine.running {
			t.Fatal("unconfirmed/disconnected start accepted")
		}
	}
}

func TestConfigCannotEnableSimulationOrStartMonitoring(t *testing.T) {
	d := manualDashboard(t, config.Data{})
	h, _ := d.nativeAssetHandler()
	r := httptest.NewRequest("POST", "http://"+dashview.DesktopHost+"/api/config", strings.NewReader(`{"realMode":false,"playerName":"Tester"}`))
	r.Header.Set("X-Companion", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !d.config().RealMode || d.engine.running {
		t.Fatalf("config behavior: %d", w.Code)
	}
}
