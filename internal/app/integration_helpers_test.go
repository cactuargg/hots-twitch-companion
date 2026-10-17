package app

import (
	"context"
	"hots-twitch-companion/internal/config"
	dashview "hots-twitch-companion/internal/dashboard"
	"hots-twitch-companion/internal/hots/correlation"
	"hots-twitch-companion/internal/hots/discovery"
	predmodel "hots-twitch-companion/internal/prediction"
	"hots-twitch-companion/internal/storage"
	twitchpkg "hots-twitch-companion/internal/twitch"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// The legacy integration suite exercises the real Dashboard/Application boundary.
// These fixture conveniences exist only in tests; the executable has no aliases
// or forwarding types for the former monolithic command.
const appVersion = "0.9.6"

var defaultTwitchClientID string
var httpClient = &http.Client{Timeout: 20 * time.Second}

func requestJSON(ctx context.Context, method, endpoint string, form url.Values, body any, token, id string, dst any) error {
	return twitchpkg.HTTPRequest(httpClient)(ctx, method, endpoint, form, body, token, id, dst)
}
func newTwitch(p config.Paths, id, lang string) (*twitchpkg.Client, error) {
	return twitchpkg.New(id, lang, twitchpkg.FileTokens{Path: p.Secret}, requestJSON)
}
func newEngine(p config.Paths, c config.Data, log func(string)) (*Engine, error) {
	return newEngineWithStore(p, c, log, nil)
}
func newEngineWithStore(p config.Paths, c config.Data, log func(string), store storage.StateStore) (*Engine, error) {
	e, err := New(p, c, Options{DefaultClientID: defaultTwitchClientID, Log: log, Store: store, Request: requestJSON})
	if err == nil && store == nil {
		e.store = fileStateStore{p: &e.p}
	}
	return e, err
}

type fileStateStore struct{ p *config.Paths }

func (s fileStateStore) Load() (predmodel.State, error) {
	return storage.FileStateStore{Path: s.p.State}.Load()
}
func (s fileStateStore) Save(state predmodel.State) error {
	return storage.FileStateStore{Path: s.p.State}.Save(state)
}
func loadState(p config.Paths) (predmodel.State, error) {
	return storage.FileStateStore{Path: p.State}.Load()
}
func saveState(p config.Paths, s predmodel.State) error {
	return storage.FileStateStore{Path: p.State}.Save(s)
}
func applyProfileReset(p config.Paths) error {
	return storage.ApplyProfileReset(p.Dir, p.Config, p.State, p.Secret)
}
func migrateLiveProfile(p config.Paths, c config.Data) error {
	if err := storage.MigrateLegacySimulation(p.State); err != nil {
		return err
	}
	return config.Write(p, c)
}
func replaysAfter(c config.Data, at time.Time, max int) []discovery.ReplayFile {
	return discovery.ReplaysAfter(c.ReplayRoot, at, max)
}
func defaultReplayRoots() []string { return discovery.DefaultReplayRoots() }
func correlationReplayFiles(c config.Data, after time.Time) ([]discovery.ReplayFile, error) {
	return correlation.CorrelationReplayFiles(c.ReplayRoot, after)
}
func correlatePrediction(c config.Data, p *predmodel.Prediction, now time.Time) (*correlation.CorrelatedReplay, string, string, string) {
	return correlation.CorrelatePrediction(c.ReplayRoot, p, now)
}
func findLobby() (string, string, time.Time, error) { return (DiskSource{}).FindLobby() }

type dashboardFixture struct {
	*dashview.Dashboard
	engine     *Engine
	p          config.Paths
	shutdown   <-chan struct{}
	hideToTray func() error
}

func newDashboard(p config.Paths, c config.Data) (*dashboardFixture, error) {
	e, err := newEngine(p, c, func(string) {})
	if err != nil {
		return nil, err
	}
	d := dashview.New(e, p, appVersion)
	f := &dashboardFixture{Dashboard: d, engine: e, p: p, shutdown: d.Done()}
	d.SetHideToTray(func() error { return f.hideToTray() })
	e.logEvent = d.LogEvent
	e.log = func(message string) { d.LogEvent("info", "", message) }
	registerTestTwitch(e)
	return f, nil
}
func manualDashboard(t *testing.T, c config.Data) *dashboardFixture {
	t.Helper()
	d, err := newDashboard(config.NewPaths(t.TempDir()), c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(d.Close)
	return d
}
func (d *dashboardFixture) routes(addr, key string) http.Handler     { return d.Routes(addr, key) }
func (d *dashboardFixture) status() map[string]any                   { return d.Status() }
func (d *dashboardFixture) config() config.Data                      { return d.Config() }
func (d *dashboardFixture) currentReplays() []dashview.ReplaySummary { return d.CurrentReplays() }
func (d *dashboardFixture) checkReplayRoot(root string) (map[string]any, error) {
	return d.CheckReplayRoot(root)
}
func (d *dashboardFixture) requestProfileReset() error                { return d.RequestProfileReset() }
func (d *dashboardFixture) nativeAssetHandler() (http.Handler, error) { return d.NativeAssetHandler() }
func (d *dashboardFixture) closeRequested() bool                      { return d.CloseRequested() }
func (d *dashboardFixture) runDesktop(open func(string) error) error  { return d.RunDesktop(open) }
func (d *dashboardFixture) logEvent(level, code, message string)      { d.LogEvent(level, code, message) }

type replaySummary = dashview.ReplaySummary
type replayRootEntry = dashview.ReplayRootEntry
type event = dashview.Event
