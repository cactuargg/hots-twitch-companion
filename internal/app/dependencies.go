package app

import (
	"context"
	"errors"
	"hots-twitch-companion/internal/clock"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/hots/correlation"
	"hots-twitch-companion/internal/hots/discovery"
	"hots-twitch-companion/internal/hots/lobby"
	"hots-twitch-companion/internal/i18n"
	"hots-twitch-companion/internal/prediction"
	"hots-twitch-companion/internal/storage"
	twitchpkg "hots-twitch-companion/internal/twitch"
	"path/filepath"
	"time"
)

// Options are per application instance. No mutable process-wide HTTP or OAuth defaults.
type Options struct {
	DefaultClientID string
	Store           storage.StateStore
	Clock           clock.Clock
	Predictions     PredictionAPI
	Request         twitchpkg.JSONRequest
	Source          GameSource
	Log             func(string)
	LogEvent        func(level, code, message string)
}

// GameSource permits deterministic scans without changing the payout evidence gate.
type GameSource interface {
	FindLobby() (string, string, time.Time, error)
	Capture(string) *lobby.Snapshot
	Correlate(string, *prediction.Prediction, time.Time) (*correlation.CorrelatedReplay, string, string, string)
}
type DiskSource struct{}

func (DiskSource) FindLobby() (string, string, time.Time, error) {
	file, key, modified, err := discovery.FindLobby()
	if errors.Is(err, discovery.ErrLobbyNotFound) {
		err = errors.New(i18n.Text(i18n.En, "lobby.not_found"))
	}
	return file, key, modified, err
}
func (DiskSource) Capture(path string) *lobby.Snapshot { return lobby.Capture(path) }
func (DiskSource) Correlate(root string, p *prediction.Prediction, now time.Time) (*correlation.CorrelatedReplay, string, string, string) {
	return correlation.CorrelatePrediction(root, p, now)
}

func (e *Engine) newTwitch(cfg config.Data) (*twitchpkg.Client, error) {
	return twitchpkg.New(cfg.TwitchClientID(e.defaultClientID), cfg.Language(), twitchpkg.FileTokens{Path: e.p.Secret}, e.request)
}
func (e *Engine) Config() config.Data { e.mu.Lock(); defer e.mu.Unlock(); return e.cfg }
func (e *Engine) ClientID() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cfg.TwitchClientID(e.defaultClientID)
}
func (e *Engine) client() *twitchpkg.Client        { e.mu.Lock(); defer e.mu.Unlock(); return e.twitch }
func (e *Engine) Connection() twitchpkg.Connection { return e.client().Connection() }
func (e *Engine) Identity(ctx context.Context) (twitchpkg.Identity, error) {
	return e.client().Identity(ctx)
}
func (e *Engine) StartDevice(ctx context.Context) (twitchpkg.DeviceResponse, error) {
	return e.client().StartDevice(ctx)
}
func (e *Engine) PollDevice(ctx context.Context, d twitchpkg.DeviceResponse, log func(string)) (twitchpkg.Token, error) {
	return e.client().PollDevice(ctx, d, log)
}

// Snapshot copies all mutable durable state before returning it to another owner.
// Lock order: actionMu -> mu -> Twitch's private lock. UI snapshots only need mu.
func (e *Engine) Snapshot() (bool, config.Data, prediction.State, correlation.Scan) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running, e.cfg, prediction.CloneState(e.state), e.scan
}

// RequestProfileReset writes the marker under the same action lock as all
// prediction writes. The caller closes the UI only after this succeeds.
func (e *Engine) RequestProfileReset() error {
	e.actionMu.Lock()
	defer e.actionMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running || e.state.Active != nil || e.state.UnconfirmedCreation != nil {
		return errors.New(i18n.Text(e.cfg.Language(), "reset.busy"))
	}
	return storage.SaveJSON(filepath.Join(e.p.Dir, storage.ResetMarker), true)
}
