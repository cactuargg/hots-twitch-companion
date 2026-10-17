package app

import (
	"hots-twitch-companion/internal/config"
	dashview "hots-twitch-companion/internal/dashboard"
	predmodel "hots-twitch-companion/internal/prediction"
	"hots-twitch-companion/internal/storage"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResetClearsOnlyOwnedDataOnNextLaunch(t *testing.T) {
	d := manualDashboard(t, config.Data{WizardDone: true})
	for _, name := range []string{d.p.Config, d.p.State, d.p.Secret, d.p.Secret + ".new"} {
		if err := os.WriteFile(name, []byte("saved"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cache := filepath.Join(d.p.Dir, "webview2")
	os.MkdirAll(cache, 0700)
	os.WriteFile(filepath.Join(cache, "cache"), []byte("cache"), 0600)
	replay := filepath.Join(d.p.Dir, "keep.StormReplay")
	os.WriteFile(replay, []byte("replay"), 0600)
	if err := d.requestProfileReset(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(d.p.Secret); err != nil {
		t.Fatal("data deleted before process exit", err)
	}
	if err := applyProfileReset(d.p); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{d.p.Config, d.p.State, d.p.Secret, d.p.Secret + ".new", cache, filepath.Join(d.p.Dir, storage.ResetMarker)} {
		if _, err := os.Stat(name); !os.IsNotExist(err) {
			t.Fatalf("not removed: %s", name)
		}
	}
	if _, err := os.Stat(replay); err != nil {
		t.Fatal("replay removed")
	}
	cfg, err := config.Read(d.p)
	if err != nil || cfg.WizardDone || !cfg.RealMode {
		t.Fatal("not a fresh profile")
	}
	if err := applyProfileReset(d.p); err != nil {
		t.Fatal("reset is not idempotent", err)
	}
}

func TestResetRejectsRunningOrActivePrediction(t *testing.T) {
	for _, active := range []bool{false, true} {
		d := manualDashboard(t, config.Data{})
		if active {
			d.engine.state.Active = &predmodel.Prediction{ID: "keep"}
		} else {
			d.engine.running = true
		}
		if err := d.requestProfileReset(); err == nil {
			t.Fatal("unsafe reset accepted")
		}
		if _, err := os.Stat(filepath.Join(d.p.Dir, storage.ResetMarker)); !os.IsNotExist(err) {
			t.Fatal("reset marker created")
		}
	}
}

func TestResetCannotEraseUnconfirmedTwitchCreation(t *testing.T) {
	d := manualDashboard(t, config.Data{RealMode: true})
	state := predmodel.State{UnconfirmedCreation: &predmodel.CreationAttempt{
		LobbySHA: "pending", BroadcasterID: "channel", AttemptedAt: 123,
	}}
	if err := d.engine.updateAndSave(state); err != nil {
		t.Fatal(err)
	}
	if err := d.requestProfileReset(); err == nil {
		t.Fatal("reset erased an unverified Twitch wager")
	}
	persisted, err := loadState(d.p)
	if err != nil || persisted.UnconfirmedCreation == nil {
		t.Fatalf("unconfirmed wager was forgotten: %+v %v", persisted, err)
	}
	if _, err := os.Stat(filepath.Join(d.p.Dir, storage.ResetMarker)); !os.IsNotExist(err) {
		t.Fatal("unsafe reset marker created")
	}
}

func TestResetFailureRetainsMarkerForRetry(t *testing.T) {
	d := manualDashboard(t, config.Data{})
	if err := d.requestProfileReset(); err != nil {
		t.Fatal(err)
	}
	os.Mkdir(d.p.Config, 0700)
	os.WriteFile(filepath.Join(d.p.Config, "block"), []byte("x"), 0600)
	if err := applyProfileReset(d.p); err == nil {
		t.Fatal("expected deletion failure")
	}
	if _, err := os.Stat(filepath.Join(d.p.Dir, storage.ResetMarker)); err != nil {
		t.Fatal("retry marker lost")
	}
	os.RemoveAll(d.p.Config)
	if err := applyProfileReset(d.p); err != nil {
		t.Fatal(err)
	}
}

func TestResetEndpointRequiresGuardsAndExplicitConfirmation(t *testing.T) {
	for _, tc := range []struct {
		body, origin, header string
		want                 int
	}{
		{`{"confirmed":true}`, "https://evil.test", "1", 403},
		{`{"confirmed":true}`, "http://" + dashview.DesktopHost, "", 403},
		{`{"confirmed":false}`, "http://" + dashview.DesktopHost, "1", 400},
		{`{"confirmed":true}`, "http://" + dashview.DesktopHost, "1", 200},
	} {
		d := manualDashboard(t, config.Data{})
		h, _ := d.nativeAssetHandler()
		r := httptest.NewRequest("POST", "http://"+dashview.DesktopHost+"/api/profile/reset", strings.NewReader(tc.body))
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("X-Companion", tc.header)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("reset: got %d want %d: %s", w.Code, tc.want, w.Body.String())
		}
		if w.Code == 200 {
			r = httptest.NewRequest("POST", "http://"+dashview.DesktopHost+"/api/auth", strings.NewReader("{}"))
			r.Header.Set("X-Companion", "1")
			w = httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 503 {
				t.Fatal("new auth accepted after reset")
			}
		}
	}
}
