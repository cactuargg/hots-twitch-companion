package app

import (
	"errors"
	"hots-twitch-companion/internal/config"
	dashview "hots-twitch-companion/internal/dashboard"
	predmodel "hots-twitch-companion/internal/prediction"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNativeTransportRetainsSecurityGuards(t *testing.T) {
	d := manualDashboard(t, config.Data{Lang: "en"})
	h, err := d.nativeAssetHandler()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, method, path, host, origin, companion, site string
		want                                              int
	}{
		{"page", "GET", "/", dashview.DesktopHost, "", "", "", 200},
		{"script", "GET", "/assets/core.js", dashview.DesktopHost, "", "", "", 200},
		{"status", "GET", "/api/status", dashview.DesktopHost, "", "1", "", 200},
		{"host", "GET", "/api/status", "evil.test", "", "1", "", 403},
		{"header", "GET", "/api/status", dashview.DesktopHost, "", "", "", 403},
		{"origin", "POST", "/api/automation/stop", dashview.DesktopHost, "https://evil.test", "1", "same-origin", 403},
		{"site", "POST", "/api/automation/stop", dashview.DesktopHost, "http://" + dashview.DesktopHost, "1", "cross-site", 403},
		{"stop", "POST", "/api/automation/stop", dashview.DesktopHost, "http://" + dashview.DesktopHost, "1", "same-origin", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "http://"+dashview.DesktopHost+tc.path, strings.NewReader("{}"))
			r.Host = tc.host
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("X-Companion", tc.companion)
			r.Header.Set("Sec-Fetch-Site", tc.site)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if tc.name == "page" && (!strings.Contains(w.Body.String(), "HotS Twitch Companion") || !strings.Contains(w.Header().Get("Content-Security-Policy"), "script-src 'self'")) {
				t.Fatal("UI/CSP missing")
			}
			if r.URL.Path != tc.path {
				t.Fatal("adapter mutated caller URL")
			}
		})
	}
}

func TestHideToTrayUsesSameGuardsAndNeverTouchesPrediction(t *testing.T) {
	d := manualDashboard(t, config.Data{})
	d.engine.state.Active = &predmodel.Prediction{ID: "keep"}
	h, err := d.nativeAssetHandler()
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	d.hideToTray = func() error { calls++; return nil }
	for _, tc := range []struct {
		origin, header string
		want           int
	}{
		{"https://evil.test", "1", 403}, {"http://" + dashview.DesktopHost, "", 403}, {"http://" + dashview.DesktopHost, "1", 200},
	} {
		r := httptest.NewRequest("POST", "http://"+dashview.DesktopHost+"/api/window/hide", strings.NewReader("{}"))
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("X-Companion", tc.header)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("hide: %d", w.Code)
		}
	}
	if calls != 1 || d.engine.state.Active.ID != "keep" {
		t.Fatal("hide bypassed guards or changed prediction")
	}
	d.hideToTray = func() error { return errors.New("not ready") }
	r := httptest.NewRequest("POST", "http://"+dashview.DesktopHost+"/api/window/hide", nil)
	r.Header.Set("X-Companion", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 503 {
		t.Fatalf("unavailable tray: %d", w.Code)
	}
}

func TestNativeCloseRequiresConfirmationForMonitoringOrPrediction(t *testing.T) {
	d := manualDashboard(t, config.Data{})
	if d.closeRequested() {
		t.Fatal("idle close should not prompt")
	}
	d.engine.running = true
	if !d.closeRequested() {
		t.Fatal("monitoring close must prompt")
	}
	d.engine.running = false
	d.engine.state.Active = &predmodel.Prediction{ID: "saved"}
	if !d.closeRequested() {
		t.Fatal("active prediction must prompt")
	}
	d.Close()
	if d.closeRequested() {
		t.Fatal("confirmed frontend Quit must not prompt twice")
	}
	if d.engine.state.Active.ID != "saved" {
		t.Fatal("closing changed prediction")
	}
}

func TestNativeQuitKeepsPersistedPrediction(t *testing.T) {
	d := manualDashboard(t, config.Data{})
	saved := predmodel.State{Active: &predmodel.Prediction{ID: "SIMULATION", DryRun: true}}
	if err := d.engine.updateAndSave(saved); err != nil {
		t.Fatal(err)
	}
	h, err := d.nativeAssetHandler()
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "http://"+dashview.DesktopHost+"/api/quit", strings.NewReader("{}"))
	r.Header.Set("Origin", "http://"+dashview.DesktopHost)
	r.Header.Set("X-Companion", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("quit: %d", w.Code)
	}
	select {
	case <-d.shutdown:
	default:
		t.Fatal("quit did not signal shutdown")
	}
	restored, err := loadState(d.p)
	if err != nil || restored.Active == nil || restored.Active.ID != "SIMULATION" {
		t.Fatalf("prediction not preserved: %+v %v", restored, err)
	}
}
