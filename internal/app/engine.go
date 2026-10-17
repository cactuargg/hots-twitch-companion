package app

import (
	"context"
	"errors"
	"fmt"
	"hots-twitch-companion/internal/clock"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/hots/correlation"
	"hots-twitch-companion/internal/i18n"
	"hots-twitch-companion/internal/prediction"
	"hots-twitch-companion/internal/storage"
	twitchpkg "hots-twitch-companion/internal/twitch"
	"net/http"
	"path/filepath"
	"sync"
	"time"
)

type Engine struct {
	defaultClientID string
	request         twitchpkg.JSONRequest
	source          GameSource
	mu              sync.Mutex
	actionMu        sync.Mutex
	p               config.Paths
	store           storage.StateStore
	clock           clock.Clock
	predictions     PredictionAPI
	cfg             config.Data
	twitch          *twitchpkg.Client
	state           prediction.State
	scan            correlation.Scan
	running         bool
	stop            context.CancelFunc
	log             func(string)
	logEvent        func(level, code, message string)
}

func New(p config.Paths, cfg config.Data, options Options) (*Engine, error) {
	e := &Engine{p: p, cfg: cfg, log: options.Log, logEvent: options.LogEvent, clock: options.Clock, defaultClientID: options.DefaultClientID, request: options.Request, source: options.Source}
	if e.log == nil {
		e.log = func(string) {}
	}
	if e.clock == nil {
		e.clock = clock.System{}
	}
	if e.request == nil {
		e.request = twitchpkg.HTTPRequest(&http.Client{Timeout: 20 * time.Second})
	}
	if e.source == nil {
		e.source = DiskSource{}
	}
	store := options.Store
	if store == nil {
		// Production persistence uses the existing profile state path.
		store = storage.FileStateStore{Path: p.State}
	}
	e.store = store
	s, err := store.Load()
	if err != nil {
		return nil, err
	}
	if s.LastCreatedLobbySHA == "" {
		if s.Active != nil && s.Active.Match != nil && s.Active.Match.Lobby != nil {
			s.LastCreatedLobbySHA = s.Active.Match.Lobby.SHA256
		} else if s.LastLobby != nil {
			s.LastCreatedLobbySHA = s.LastLobby.SHA256
		}
	}
	if len(s.RecentLobbySHAs) == 0 && s.LastCreatedLobbySHA != "" {
		s.RecentLobbySHAs = []string{s.LastCreatedLobbySHA}
	}
	twitch, err := e.newTwitch(cfg)
	if err != nil {
		return nil, err
	}
	e.state, e.twitch, e.predictions = s, twitch, twitch
	if options.Predictions != nil {
		e.predictions = options.Predictions
	}
	return e, nil
}

// report sends a structured diagnostic when the dashboard is attached, while
// preserving the plain logger for CLI tests and other engine callers.
func (e *Engine) report(level, code, message string) {
	if e.logEvent != nil {
		e.logEvent(level, code, message)
		return
	}
	e.log(message)
}

// safeLiveChange reports whether cfg differs from the stored config only by
// language or wizard completion. Such changes stay allowed while monitoring
// runs so the UI can switch language or finish setup live.
func (e *Engine) safeLiveChange(cfg config.Data) bool {
	return LiveConfigChangeAllowed(e.cfg, cfg)
}
func (e *Engine) UpdateConfig(cfg config.Data) error {
	// All scanner/settlement actions read cfg and twitch while holding
	// actionMu. Serialize live configuration changes with those actions.
	e.actionMu.Lock()
	defer e.actionMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running && !e.safeLiveChange(cfg) {
		return errors.New(i18n.Text(e.cfg.Language(), "config.stop_watch_first"))
	}
	if e.state.Active != nil && e.state.Active.DryRun != !cfg.RealMode {
		return errors.New(i18n.Text(e.cfg.Language(), "config.mode_conflict_active"))
	}
	// An unresolved real Prediction belongs to a specific Twitch OAuth app.
	// Changing it would strand the remote Prediction even after reauth.
	if (e.state.Active != nil && !e.state.Active.DryRun || e.state.UnconfirmedCreation != nil) && cfg.TwitchClientID(e.defaultClientID) != e.cfg.TwitchClientID(e.defaultClientID) {
		return errors.New("Finish or cancel the active Twitch Prediction before changing the OAuth Client ID")
	}
	// Do not rebuild an authorized Twitch client for language or wizard
	// changes: an in-flight token refresh could otherwise update the old
	// instance after the replacement loaded outdated credentials from disk.
	if e.cfg.TwitchClientID(e.defaultClientID) == cfg.TwitchClientID(e.defaultClientID) {
		if err := config.Write(e.p, cfg); err != nil {
			return err
		}
		e.twitch.SetLanguage(cfg.Language())
		e.cfg = cfg
		return nil
	}
	// Validate new dependencies *before* persisting the new settings.
	// A corrupt DPAPI file or unreadable token must not leave a config on
	// disk that the running engine never accepted.
	t, err := e.newTwitch(cfg)
	if err != nil {
		return err
	}
	if err := config.Write(e.p, cfg); err != nil {
		return err
	}
	e.cfg = cfg
	e.twitch = t
	e.predictions = t
	return nil
}

// installTwitchToken validates a new device-flow token *before* persisting it.
// actionMu serializes with manual/replay settlement, so the active Prediction
// cannot change between checking ownership and replacing the credentials.
// Reauthorization of the same broadcaster is allowed during a prediction.
func (e *Engine) InstallTwitchToken(ctx context.Context, expectedClientID string, token twitchpkg.Token) (twitchpkg.Identity, error) {
	if token.AccessToken == "" || token.RefreshToken == "" {
		return twitchpkg.Identity{}, errors.New("Twitch authorization returned incomplete tokens")
	}
	e.mu.Lock()
	lang := e.cfg.Language()
	e.mu.Unlock()
	me, err := twitchpkg.ValidateIdentity(ctx, token.AccessToken, expectedClientID, lang, e.request)
	if err != nil {
		return twitchpkg.Identity{}, err
	}
	e.actionMu.Lock()
	defer e.actionMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cfg.TwitchClientID(e.defaultClientID) != expectedClientID {
		return twitchpkg.Identity{}, errors.New("Twitch OAuth application changed during authorization; please reconnect")
	}
	if active := e.state.Active; active != nil && !active.DryRun &&
		(active.BroadcasterID == "" || active.BroadcasterID != me.UserID) {
		return twitchpkg.Identity{}, errors.New("Finish or cancel the active Twitch Prediction before connecting another channel")
	}
	if uncertain := e.state.UnconfirmedCreation; uncertain != nil && uncertain.BroadcasterID != me.UserID {
		return twitchpkg.Identity{}, errors.New("Reconcile the unconfirmed Prediction before connecting another Twitch channel")
	}
	t := e.twitch
	if err := t.AcceptValidatedToken(token, me); err != nil {
		return twitchpkg.Identity{}, err
	}
	return me, nil
}

// acknowledgeUnconfirmedCreation is the only path that can clear an
// uncertain creation without a known Prediction ID. The user must explicitly
// confirm the check, and Twitch must verify that this same broadcaster has
// no ACTIVE/LOCKED Prediction. No matching by title or blind adoption.
func (e *Engine) AcknowledgeUnconfirmedCreation(ctx context.Context) error {
	e.actionMu.Lock()
	defer e.actionMu.Unlock()
	e.mu.Lock()
	s := e.state
	cfg := e.cfg
	twitch := e.twitch
	e.mu.Unlock()
	if s.UnconfirmedCreation == nil || s.Active != nil || !cfg.RealMode {
		return errors.New("There is no unconfirmed Twitch creation to reconcile")
	}
	identity, err := twitch.PreflightCreate(ctx)
	if err != nil {
		return fmt.Errorf("Cannot confirm Twitch is clear: %w", err)
	}
	if identity.UserID != s.UnconfirmedCreation.BroadcasterID {
		return errors.New("Authorized Twitch channel does not match the unconfirmed creation")
	}
	next := s
	next.UnconfirmedCreation = nil
	if err := e.updateAndSave(next); err != nil {
		return err
	}
	e.setReplayScan("checking", "", "")
	return nil
}

func (e *Engine) updateAndSave(s prediction.State) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.store.Save(s); err != nil {
		return err
	}
	e.state = s
	return nil
}
func (e *Engine) doCreate(ctx context.Context, s *prediction.State, lobbyKey string) error {
	return e.doCreatePrepared(ctx, s, lobbyKey, "")
}

func (e *Engine) doCreatePrepared(ctx context.Context, s *prediction.State, lobbyKey, broadcasterID string) error {
	if s.Active != nil {
		return nil
	}
	var p *prediction.Prediction
	var er error
	if !e.cfg.RealMode {
		p = &prediction.Prediction{ID: "SIMULATION", DryRun: true, StartedAt: max(e.clock.Now().UnixMilli(), s.LastSettlementAt+1)}
		e.log(i18n.Text(e.cfg.Language(), "log.sim_created"))
	} else {
		title, winTitle, lossTitle := e.cfg.PredictionTexts()
		if broadcasterID != "" {
			p, er = e.predictions.CreatePrepared(ctx, broadcasterID, e.cfg.PredictionWindowMinutes(), title, winTitle, lossTitle)
		} else {
			p, er = e.predictions.Create(ctx, e.cfg.PredictionWindowMinutes(), title, winTitle, lossTitle)
		}
		if er != nil {
			return er
		}
		e.log(i18n.Text(e.cfg.Language(), "log.prediction_created", p.ID))
	}
	next, err := prediction.ConfirmedCreation(*s, p, correlation.BindPredictionMatch(s.LastLobby, lobbyKey, e.cfg.PlayerToonHandle))
	if err != nil {
		return err
	}
	// Keep the confirmed ID in memory even if the follow-up state write
	// fails. The durable attempted marker prevents a duplicate POST.
	*s = next
	e.setReplayScan("waiting_replay", "", "")
	if err := e.updateAndSave(next); err != nil {
		e.mu.Lock()
		e.state = next
		e.mu.Unlock()
		return err
	}
	return nil
}
func (e *Engine) doSettle(ctx context.Context, s *prediction.State, outcome string, source string) error {
	return e.settlePrediction(ctx, s, outcome, source)
}

// manualOutcome is serialized with the replay scanner. The caller supplies the
// exact active prediction identity so a delayed click can never settle a newer game.
func (e *Engine) ManualOutcome(ctx context.Context, outcome, expectedID string, expectedStartedAt int64) error {
	if outcome != "win" && outcome != "loss" && outcome != "cancel" && outcome != "sync" {
		return errors.New(i18n.Text(e.cfg.Language(), "manual.invalid"))
	}
	if expectedID == "" || expectedStartedAt <= 0 {
		return errors.New(i18n.Text(e.cfg.Language(), "manual.stale"))
	}
	e.actionMu.Lock()
	defer e.actionMu.Unlock()
	e.mu.Lock()
	s := e.state
	e.mu.Unlock()
	if s.Active == nil {
		return errors.New(i18n.Text(e.cfg.Language(), "settle.none_active"))
	}
	if s.Active.ID != expectedID || s.Active.StartedAt != expectedStartedAt {
		return errors.New(i18n.Text(e.cfg.Language(), "manual.stale"))
	}
	if outcome == "sync" {
		if s.Active.DryRun {
			return errors.New(i18n.Text(e.cfg.Language(), "manual.sync_simulation"))
		}
		remote, err := e.predictions.Prediction(ctx, s.Active.BroadcasterID, s.Active.ID)
		if err != nil {
			return err
		}
		for _, p := range remote.Data {
			if p.ID != s.Active.ID {
				continue
			}
			if p.Status != "RESOLVED" && p.Status != "CANCELED" {
				return errors.New(i18n.Text(e.cfg.Language(), "manual.still_open"))
			}
			// Use the same immutable completion transition as automatic settlement.
			// The result mapping also stays shared with recovery from an external
			// Twitch resolution; do not maintain a second outcome state machine.
			next, err := prediction.Completed(s, twitchpkg.RemoteOutcome(p, s.Active), "Twitch", e.clock.Now())
			if err != nil {
				return err
			}
			if err := e.updateAndSave(next); err != nil {
				return err
			}
			e.report("info", "prediction_synced", i18n.Text(e.cfg.Language(), "manual.synced", p.Status))
			return nil
		}
		return errors.New(i18n.Text(e.cfg.Language(), "manual.remote_missing"))
	}
	if err := e.doSettle(ctx, &s, outcome, "manual"); err != nil {
		e.report("error", "prediction_settle", i18n.Text(e.cfg.Language(), "manual.failed", err.Error()))
		return err
	}
	e.report("info", "prediction_manual", i18n.Text(e.cfg.Language(), "manual.completed", s.LastResult))
	return nil
}

func (e *Engine) Start() error {
	e.mu.Lock()
	reason := MonitorStartBlockReason(e.running, e.cfg, e.cfg.TwitchClientID(e.defaultClientID) != "", e.state)
	if reason != "" {
		lang := e.cfg.Language()
		e.mu.Unlock()
		if reason == UnconfirmedCreationBlock {
			return errors.New("Reconcile the unconfirmed Twitch Prediction before starting automation")
		}
		return errors.New(i18n.Text(lang, reason))
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.running = true
	e.stop = cancel
	e.mu.Unlock()
	go e.watch(ctx)
	return nil
}
func (e *Engine) StopWatch() {
	e.mu.Lock()
	if e.stop != nil {
		e.stop()
	}
	e.mu.Unlock()
}
func (e *Engine) watch(ctx context.Context) {
	defer func() {
		e.mu.Lock()
		e.running = false
		lang := e.cfg.Language()
		e.mu.Unlock()
		e.log(i18n.Text(lang, "log.watch_stopped"))
	}()
	// Snapshot startup values under the engine lock. A safe live settings
	// change can otherwise replace cfg/twitch during OAuth validation.
	e.mu.Lock()
	config, twitch := e.cfg, e.twitch
	e.mu.Unlock()
	lang := config.Language()
	e.log(i18n.Text(lang, "log.watch_started", i18n.Text(lang, map[bool]string{true: "mode.real", false: "mode.simulation"}[config.RealMode])))
	if config.RealMode {
		if _, err := twitch.Identity(ctx); err != nil {
			e.report("error", "twitch_auth", "Twitch: "+err.Error())
			return
		}
	}
	started := e.clock.Now()
	lastError := ""
	lastLog := time.Time{}
	// Immediately check any persisted active Prediction on monitoring restart.
	e.scanTick(ctx, started, &lastError, &lastLog)
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.scanTick(ctx, started, &lastError, &lastLog)
		}
	}
}

// scanTick and manualOutcome share actionMu so a manual decision cannot race
// the replay scanner or accidentally restore a previously settled prediction.
func (e *Engine) scanTick(ctx context.Context, started time.Time, lastError *string, lastLog *time.Time) {
	e.actionMu.Lock()
	defer e.actionMu.Unlock()
	if ctx.Err() != nil {
		return
	}
	e.mu.Lock()
	s := e.state
	e.mu.Unlock()
	e.scanLifecycle(ctx, &s, started, e.clock.Now(), lastError, lastLog)
}

func (e *Engine) setReplayScan(code, file, detail string) {
	e.mu.Lock()
	e.scan = correlation.Scan{Code: code, File: file, Detail: detail, CheckedAt: e.clock.Now().UnixMilli()}
	e.mu.Unlock()
}

// scanReplay applies the same correlation gate for background and manual rechecks.
func (e *Engine) scanReplay(ctx context.Context, s *prediction.State, now time.Time, lastError *string, lastLog *time.Time) {
	if s.Active == nil {
		return
	}
	match, code, file, detail := e.source.Correlate(e.cfg.ReplayRoot, s.Active, now)
	if match == nil {
		if code == "correlation_ambiguous" && s.Active.Match != nil && !s.Active.Match.Ambiguous {
			s.Active = correlation.AmbiguousMatch(s.Active)
			if err := e.updateAndSave(*s); err != nil {
				e.mu.Lock()
				e.state = *s
				e.mu.Unlock()
				e.report("error", "correlation_save", err.Error())
			}
		}
		e.setReplayScan(code, file, detail)
		return
	}
	if err := ctx.Err(); err != nil {
		return
	}
	if err := e.doSettle(ctx, s, match.Player.Result, match.Path); err != nil {
		e.setReplayScan("settle_error", filepath.Base(match.Path), err.Error())
		e.report("error", "prediction_settle", i18n.Text(e.cfg.Language(), "log.cancel_error", err.Error()))
		return
	}
	e.setReplayScan("settled", filepath.Base(match.Path), "")
	e.log(i18n.Text(e.cfg.Language(), "log.settled", s.LastResult))
}

// recheckReplay only evaluates completed replay files. It never creates a new
// Prediction and refuses stale UI requests for a different active game.
func (e *Engine) RecheckReplay(ctx context.Context, expectedID string, expectedStartedAt int64) (correlation.Scan, error) {
	e.actionMu.Lock()
	defer e.actionMu.Unlock()
	e.mu.Lock()
	s := e.state
	e.mu.Unlock()
	if s.Active == nil || s.Active.ID != expectedID || s.Active.StartedAt != expectedStartedAt || expectedID == "" {
		return correlation.Scan{}, errors.New(i18n.Text(e.cfg.Language(), "manual.stale"))
	}
	lastError := ""
	lastLog := time.Time{}
	e.scanReplay(ctx, &s, e.clock.Now(), &lastError, &lastLog)
	e.mu.Lock()
	status := e.scan
	e.mu.Unlock()
	return status, nil
}
