package dashboard

import (
	"context"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/hots/correlation"
	"hots-twitch-companion/internal/prediction"
	twitchpkg "hots-twitch-companion/internal/twitch"
	"net/http"
	"time"
)

// Application exposes operations, never Engine fields or locks. It is owned by
// this consumer and implemented by internal/app without a dependency on UI.
type Application interface {
	Config() config.Data
	Snapshot() (bool, config.Data, prediction.State, correlation.Scan)
	ClientID() string
	Connection() twitchpkg.Connection
	Identity(context.Context) (twitchpkg.Identity, error)
	StartDevice(context.Context) (twitchpkg.DeviceResponse, error)
	PollDevice(context.Context, twitchpkg.DeviceResponse, func(string)) (twitchpkg.Token, error)
	InstallTwitchToken(context.Context, string, twitchpkg.Token) (twitchpkg.Identity, error)
	UpdateConfig(config.Data) error
	AcknowledgeUnconfirmedCreation(context.Context) error
	RecheckReplay(context.Context, string, int64) (correlation.Scan, error)
	ManualOutcome(context.Context, string, string, int64) error
	Start() error
	StopWatch()
	RequestProfileReset() error
}

func (d *Dashboard) Routes(addr, secret string) http.Handler {
	return NewHTTPHandler(addr, secret, d.handleAPIRoute, func() { d.mu.Lock(); d.lastActivity = time.Now(); d.mu.Unlock() })
}
func (d *Dashboard) RequestProfileReset() error {
	if err := d.engine.RequestProfileReset(); err != nil {
		return err
	}
	d.Close()
	return nil
}
func (d *Dashboard) Close() {
	d.mu.Lock()
	d.shutdownOnce.Do(func() { close(d.shutdown) })
	cancel := d.backgroundCancel
	d.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if d.updates != nil {
		d.updates.Close()
	}
	if d.engine != nil {
		d.engine.StopWatch()
	}
}
func (d *Dashboard) Done() <-chan struct{}           { return d.shutdown }
func (d *Dashboard) ProfileDir() string              { return d.p.Dir }
func (d *Dashboard) SetHideToTray(hide func() error) { d.hideToTray = hide }
func (d *Dashboard) SetUIReady(ready func() error)   { d.uiReady = ready }
func (d *Dashboard) NotifyUIReady() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.uiReady != nil && !d.uiReadyConfirmed {
		if err := d.uiReady(); err != nil {
			return err
		}
		d.uiReadyConfirmed = true
	}
	return nil
}

func (d *Dashboard) updateStartupPending() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.uiReady != nil && !d.uiReadyConfirmed
}
func (d *Dashboard) CloseRequested() bool {
	select {
	case <-d.shutdown:
		return false
	default:
	}
	running, _, state, _ := d.engine.Snapshot()
	overlaysActive := false
	if d.overlays != nil {
		s, err := d.overlays.store.Snapshot()
		overlaysActive = err == nil && (s.Preferences.Enabled || s.Preferences.MaskEnabled)
	}
	return running || state.Active != nil || overlaysActive
}
