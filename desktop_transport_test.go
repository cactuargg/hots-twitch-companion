package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNativeTransportRetainsSecurityGuards(t *testing.T) {
	d := manualDashboard(t, Config{Lang: "en"})
	h, err := d.nativeAssetHandler()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, method, path, host, origin, companion, site string
		want                                              int
	}{
		{"page", "GET", "/", desktopHost, "", "", "", 200},
		{"script", "GET", "/assets/core.js", desktopHost, "", "", "", 200},
		{"status", "GET", "/api/status", desktopHost, "", "1", "", 200},
		{"host", "GET", "/api/status", "evil.test", "", "1", "", 403},
		{"header", "GET", "/api/status", desktopHost, "", "", "", 403},
		{"origin", "POST", "/api/automation/stop", desktopHost, "https://evil.test", "1", "same-origin", 403},
		{"site", "POST", "/api/automation/stop", desktopHost, "http://" + desktopHost, "1", "cross-site", 403},
		{"stop", "POST", "/api/automation/stop", desktopHost, "http://" + desktopHost, "1", "same-origin", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "http://"+desktopHost+tc.path, strings.NewReader("{}"))
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

func TestNativeCloseRequiresConfirmationForMonitoringOrPrediction(t *testing.T) {
	d := manualDashboard(t, Config{})
	if d.closeRequested() {
		t.Fatal("idle close should not prompt")
	}
	d.engine.running = true
	if !d.closeRequested() {
		t.Fatal("monitoring close must prompt")
	}
	d.engine.running = false
	d.engine.state.Active = &Prediction{ID: "saved"}
	if !d.closeRequested() {
		t.Fatal("active prediction must prompt")
	}
	close(d.shutdown)
	if d.closeRequested() {
		t.Fatal("confirmed frontend Quit must not prompt twice")
	}
	if d.engine.state.Active.ID != "saved" {
		t.Fatal("closing changed prediction")
	}
}

func TestNativeQuitKeepsPersistedPrediction(t *testing.T) {
	d := manualDashboard(t, Config{})
	saved := State{Active: &Prediction{ID: "SIMULATION", DryRun: true}}
	if err := d.engine.updateAndSave(saved); err != nil {
		t.Fatal(err)
	}
	h, err := d.nativeAssetHandler()
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "http://"+desktopHost+"/api/quit", strings.NewReader("{}"))
	r.Header.Set("Origin", "http://"+desktopHost)
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
