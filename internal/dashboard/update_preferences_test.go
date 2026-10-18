package dashboard

import (
	"bytes"
	"encoding/json"
	"errors"
	"hots-twitch-companion/internal/config"
	"net/http/httptest"
	"testing"
)

func TestUpdatePreferencePersistsAndOldConfigWritesCannotEnableChecks(t *testing.T) {
	p := config.NewPaths(t.TempDir())
	d := noticeDashboard(t, p)
	defer d.Close()
	preference := httptest.NewRecorder()
	d.handleAPIRoute(preference, httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"enabled":false}`)), "updates/preferences")
	if preference.Code != 200 {
		t.Fatal(preference.Body.String())
	}
	cfg, err := config.Read(p)
	if err != nil || !cfg.DisableUpdateChecks {
		t.Fatalf("preference not durable: %+v %v", cfg, err)
	}
	old := cfg
	old.DisableUpdateChecks = false
	old.PlayerName = "new draft"
	old.Lang = "en"
	body, _ := json.Marshal(old)
	saved := httptest.NewRecorder()
	d.handleAPIRoute(saved, httptest.NewRequest("POST", "/", bytes.NewReader(body)), "config")
	if saved.Code != 200 {
		t.Fatal(saved.Body.String())
	}
	if !d.Config().DisableUpdateChecks || d.updates.Snapshot().Enabled {
		t.Fatal("old config snapshot re-enabled update checks")
	}
	denied := httptest.NewRecorder()
	d.handleAPIRoute(denied, httptest.NewRequest("POST", "/", bytes.NewBufferString(`{}`)), "updates/check")
	if denied.Code == 200 {
		t.Fatal("manual check allowed while disabled")
	}
	enabled := httptest.NewRecorder()
	d.handleAPIRoute(enabled, httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"enabled":true}`)), "updates/preferences")
	if enabled.Code != 200 || d.Config().DisableUpdateChecks || !d.updates.Snapshot().Enabled {
		t.Fatal("could not re-enable checks")
	}
}
func TestUpdateInstallRequiresExplicitConfirmation(t *testing.T) {
	d := noticeDashboard(t, config.NewPaths(t.TempDir()))
	defer d.Close()
	recorder := httptest.NewRecorder()
	d.handleAPIRoute(recorder, httptest.NewRequest("POST", "/", bytes.NewBufferString(`{}`)), "updates/install")
	if recorder.Code == 200 {
		t.Fatal("installation accepted without confirmation")
	}
	select {
	case <-d.Done():
		t.Fatal("refused update closed the application")
	default:
	}
}

func TestUpdatedApplicationCannotStartUntilFrontendAcknowledgesReady(t *testing.T) {
	d := noticeDashboard(t, config.NewPaths(t.TempDir()))
	defer d.Close()
	calls := 0
	fail := true
	d.SetUIReady(func() error {
		calls++
		if fail {
			return errors.New("cannot write startup marker")
		}
		return nil
	})
	if d.Status()["updateStartupPending"] != true {
		t.Fatal("missing startup barrier")
	}
	start := httptest.NewRecorder()
	d.handleAPIRoute(start, httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"confirmedRealMode":true}`)), "automation/start")
	if start.Code == 200 {
		t.Fatal("monitoring started before update startup was accepted")
	}
	first := httptest.NewRecorder()
	d.handleAPIRoute(first, httptest.NewRequest("POST", "/", bytes.NewBufferString(`{}`)), "updates/startup-ready")
	if first.Code == 200 || !d.updateStartupPending() {
		t.Fatal("failed startup marker released barrier")
	}
	fail = false
	for range 2 {
		ack := httptest.NewRecorder()
		d.handleAPIRoute(ack, httptest.NewRequest("POST", "/", bytes.NewBufferString(`{}`)), "updates/startup-ready")
		if ack.Code != 200 {
			t.Fatal(ack.Body.String())
		}
	}
	if calls != 2 || d.updateStartupPending() {
		t.Fatal("startup callback did not confirm exactly once")
	}
}
