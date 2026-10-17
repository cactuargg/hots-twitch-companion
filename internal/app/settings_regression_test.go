package app

import (
	"context"
	"errors"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/prediction"
	"hots-twitch-companion/internal/twitch"
	"net/url"
	"testing"
	"time"
)

func TestStatusPreservesFrontendPhaseKeys(t *testing.T) {
	d := manualDashboard(t, config.Data{})
	s := d.Status()
	for _, key := range []string{"monitorPhase", "predictionPhase"} {
		if _, ok := s[key]; !ok {
			t.Errorf("frontend key %q missing", key)
		}
	}
}

func TestChangingNextWindowPreservesActiveCountdownAcrossRestart(t *testing.T) {
	cfg := config.Data{RealMode: true, PredictionWindowMin: 2}
	d := manualDashboard(t, cfg)
	if err := d.engine.updateAndSave(prediction.State{Active: &prediction.Prediction{ID: "existing", StartedAt: time.Now().Add(-30 * time.Second).UnixMilli()}}); err != nil {
		t.Fatal(err)
	}
	before := d.Status()["predictionWindowSecondsLeft"].(int)
	d.engine.running = true
	owner := d.engine.twitch
	cfg.PredictionWindowMin = 30
	cfg.PredictionTitle = "Next game?"
	cfg.PredictionWin = "Yes"
	cfg.PredictionLoss = "No"
	if err := d.engine.UpdateConfig(cfg); err != nil {
		t.Fatal(err)
	}
	after := d.Status()["predictionWindowSecondsLeft"].(int)
	if after-before > 2 {
		t.Fatalf("active countdown changed from %ds to %ds", before, after)
	}
	if d.engine.twitch != owner {
		t.Fatal("live client replaced")
	}
	persisted, err := loadState(d.p)
	if err != nil || persisted.Active == nil || persisted.Active.WindowSeconds != 120 {
		t.Fatalf("legacy duration not saved: %+v %v", persisted, err)
	}
	restarted, err := newEngine(d.p, cfg, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if restarted.state.Active.WindowSeconds != 120 || restarted.cfg.PredictionWindowMinutes() != 30 {
		t.Fatal("restart changed active duration or future config")
	}
}

func TestLegacyWindowSaveFailureBlocksConfigChange(t *testing.T) {
	d := manualDashboard(t, config.Data{RealMode: true, PredictionWindowMin: 2})
	original := prediction.State{Active: &prediction.Prediction{ID: "keep", StartedAt: 123}}
	d.engine.state = original
	d.engine.store = &failureInjectingStateStore{state: original, shouldFail: func(prediction.State) bool { return true }}
	next := d.Config()
	next.PredictionWindowMin = 30
	if err := d.engine.UpdateConfig(next); err == nil {
		t.Fatal("failed state migration accepted")
	}
	if d.Config().PredictionWindowMin != 2 || d.engine.state.Active.WindowSeconds != 0 {
		t.Fatal("failed migration mutated live state")
	}
}

func TestRejectedRefreshClearsConnectedIdentity(t *testing.T) {
	e, err := New(config.NewPaths(t.TempDir()), config.Data{}, Options{Request: func(context.Context, string, string, url.Values, any, string, string, any) error {
		return errors.New("invalid_grant")
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.twitch.AcceptValidatedToken(twitch.Token{AccessToken: "expired", RefreshToken: "revoked", ExpiresIn: 3600, SavedAt: time.Now().Add(-time.Hour).Unix()}, twitch.Identity{Login: "demo", UserID: "channel"}); err != nil {
		t.Fatal(err)
	}
	// Exercise the refresh path used by prediction requests, not only Identity.
	if _, err := e.twitch.Prediction(context.Background(), "channel", "tracked"); err == nil {
		t.Fatal("expected rejected refresh")
	}
	if c := e.Connection(); c.HasToken && c.Login != "" {
		t.Fatalf("rejected credentials still reported connected: %+v", c)
	}
}

type settingsPredictionAPI struct {
	fakePredictionAPI
	window int
	title  string
}

func (f *settingsPredictionAPI) CreatePrepared(ctx context.Context, broadcaster string, window int, title, win, loss string) (*prediction.Prediction, error) {
	f.window, f.title = window, title
	return f.fakePredictionAPI.CreatePrepared(ctx, broadcaster, window, title, win, loss)
}
func TestNewPredictionCapturesSavedFutureSettings(t *testing.T) {
	d := manualDashboard(t, config.Data{RealMode: true, PredictionWindowMin: 2})
	fake := &settingsPredictionAPI{}
	d.engine.predictions = fake
	d.engine.running = true
	cfg := d.Config()
	cfg.PredictionWindowMin = 7
	cfg.PredictionTitle = "Next game?"
	if err := d.engine.UpdateConfig(cfg); err != nil {
		t.Fatal(err)
	}
	s := prediction.State{}
	if err := d.engine.doCreatePrepared(context.Background(), &s, "fixture", "channel"); err != nil {
		t.Fatal(err)
	}
	if fake.window != 7 || fake.title != "Next game?" || s.Active.WindowSeconds != 420 {
		t.Fatalf("new settings not captured: window=%d title=%q active=%+v", fake.window, fake.title, s.Active)
	}
}
