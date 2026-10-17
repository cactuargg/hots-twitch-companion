package main

import (
 appconfig "hots-twitch-companion/internal/config"
 app "hots-twitch-companion/internal/app"
	twitch "hots-twitch-companion/internal/twitch"
	discovery "hots-twitch-companion/internal/hots/discovery"
	predstate "hots-twitch-companion/internal/prediction"
	"hots-twitch-companion/internal/storage"
	"hots-twitch-companion/internal/clock"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

const scope = "channel:manage:predictions"
const appVersion = "0.9.6"
const authURL = "https://id.twitch.tv/oauth2"
const apiURL = "https://api.twitch.tv/helix"

// Retain the command-level Config as a defined compatible type while moving
// ownership of its schema and normalization rules to internal/config.
// Existing caller methods will be removed when Engine moves into internal/app.
type Config appconfig.Data

// Existing command HTTP validators still use these named limits.
const defaultPredictionWindow = appconfig.DefaultPredictionWindow
const maxPredictionWindow = appconfig.MaxPredictionWindow

// The release build may embed a public OAuth Client ID via -ldflags.
var defaultTwitchClientID string

func (c Config) twitchClientID() string {return appconfig.Data(c).TwitchClientID(defaultTwitchClientID)}
func (c Config) lang() string {return appconfig.Data(c).Language()}
func (c Config) predictionWindowMinutes() int {return appconfig.Data(c).PredictionWindowMinutes()}
func (c Config) predictionTexts() (string,string,string) {return appconfig.Data(c).PredictionTexts()}
type Token = twitch.Token
type Prediction = predstate.Prediction
// Persisted proof that a real Twitch creation POST may have been sent
// without a confirmed remote ID. Never automatically retry this POST.
type CreationAttempt = predstate.CreationAttempt

type State = predstate.State
type DeviceResponse = twitch.DeviceResponse
type TwitchIdentity = twitch.Identity
type apiPrediction = twitch.APIPrediction
type apiResponse = twitch.APIResponse
type paths struct{ dir, config, secret, state string }

func (p paths) packagePaths() appconfig.Paths {
 return appconfig.Paths{Dir:p.dir,Config:p.config,Secret:p.secret,State:p.state}
}

func pathsForUser() (paths,error) {
 p,err:=appconfig.PathsForUser()
 if err!=nil {return paths{},err}
 return paths{dir:p.Dir,config:p.Config,secret:p.Secret,state:p.State},nil
}
func saveJSON(path string, v any) error {return storage.SaveJSON(path,v)}
func loadJSON(path string, v any) error {return storage.LoadJSON(path,v)}
func readConfig(p paths) (Config,error) {
 c,err:=appconfig.Read(p.packagePaths())
 return Config(c),err
}
func writeConfig(p paths,c Config) error {return appconfig.Write(p.packagePaths(),appconfig.Data(c))}
func loadState(p paths) (State, error)    { var s State; e := loadJSON(p.state, &s); return s, e }
func saveState(p paths, s State) error    { return saveJSON(p.state, s) }

type ReplayFile = discovery.ReplayFile

func defaultReplayRoots() []string { return discovery.DefaultReplayRoots() }
func replayRoots(cfg Config) []string { return discovery.ReplayRoots(cfg.ReplayRoot) }
func replaysAfter(cfg Config, after time.Time, max int) []ReplayFile {
	return discovery.ReplaysAfter(cfg.ReplayRoot,after,max)
}
func findLobby() (string,string,time.Time,error) {
	file,key,mod,err:=discovery.FindLobby()
	if errors.Is(err,discovery.ErrLobbyNotFound) {
		return "", "", time.Time{}, errors.New(tr(langEn,"lobby.not_found"))
	}
	return file,key,mod,err
}

type ReplayScan struct {
 Code string `json:"code"`
 File string `json:"file,omitempty"`
 Detail string `json:"detail,omitempty"`
 CheckedAt int64 `json:"checkedAt,omitempty"`
}

type Engine struct {
	mu      sync.Mutex
	actionMu sync.Mutex
	p       paths
	store   StateStore
	clock   clock.Clock
	predictions PredictionAPI
	cfg     Config
	twitch  *Twitch
	state   State
	scan    ReplayScan
	running bool
	stop    context.CancelFunc
	log      func(string)
	logEvent func(level, code, message string)
}

func newEngine(p paths, cfg Config, log func(string)) (*Engine, error) {
	return newEngineWithStore(p, cfg, log, nil)
}

// newEngineWithStore allows failure-injection tests to exercise durable state
// transitions without modifying the OS filesystem. Production uses fileStateStore.
func newEngineWithStore(p paths, cfg Config, log func(string), store StateStore) (*Engine, error) {
	e := &Engine{p: p, cfg: cfg, log: log, clock: clock.System{}}
	if store == nil {
		// Use the engine's path so existing callers that redirect e.p.state
		// to simulate disk errors keep exercising the production store.
		store = fileStateStore{p: &e.p}
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
	twitch, err := newTwitch(p, cfg.twitchClientID(), cfg.lang())
	if err != nil {
		return nil, err
	}
	e.state, e.twitch, e.predictions = s, twitch, twitch
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
func (e *Engine) safeLiveChange(cfg Config) bool {
 return app.LiveConfigChangeAllowed(appconfig.Data(e.cfg), appconfig.Data(cfg))
}
func (e *Engine) updateConfig(cfg Config) error {
	// All scanner/settlement actions read cfg and twitch while holding
	// actionMu. Serialize live configuration changes with those actions.
	e.actionMu.Lock()
	defer e.actionMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running && !e.safeLiveChange(cfg) {
		return errors.New(tr(e.cfg.lang(), "config.stop_watch_first"))
	}
	if e.state.Active != nil && e.state.Active.DryRun != !cfg.RealMode {
		return errors.New(tr(e.cfg.lang(), "config.mode_conflict_active"))
	}
	// An unresolved real Prediction belongs to a specific Twitch OAuth app.
	// Changing it would strand the remote Prediction even after reauth.
	if (e.state.Active != nil && !e.state.Active.DryRun || e.state.UnconfirmedCreation != nil) && cfg.twitchClientID() != e.cfg.twitchClientID() {
		return errors.New("Finish or cancel the active Twitch Prediction before changing the OAuth Client ID")
	}
	// Do not rebuild an authorized Twitch client for language or wizard
	// changes: an in-flight token refresh could otherwise update the old
	// instance after the replacement loaded outdated credentials from disk.
	if e.cfg.twitchClientID() == cfg.twitchClientID() {
		if err := writeConfig(e.p, cfg); err != nil {
			return err
		}
		e.twitch.mu.Lock()
		e.twitch.lang = cfg.lang()
		e.twitch.mu.Unlock()
		e.cfg = cfg
		return nil
	}
	// Validate new dependencies *before* persisting the new settings.
	// A corrupt DPAPI file or unreadable token must not leave a config on
	// disk that the running engine never accepted.
	t, err := newTwitch(e.p, cfg.twitchClientID(), cfg.lang())
	if err != nil {
		return err
	}
	if err := writeConfig(e.p, cfg); err != nil {
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
func (e *Engine) installTwitchToken(ctx context.Context, expectedClientID string, token Token) (TwitchIdentity, error) {
	if token.AccessToken == "" || token.RefreshToken == "" {
		return TwitchIdentity{}, errors.New("Twitch authorization returned incomplete tokens")
	}
	e.mu.Lock()
	lang := e.cfg.lang()
	e.mu.Unlock()
	me, err := validateTwitchIdentity(ctx, token.AccessToken, expectedClientID, lang)
	if err != nil {
		return TwitchIdentity{}, err
	}
	e.actionMu.Lock()
	defer e.actionMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cfg.twitchClientID() != expectedClientID {
		return TwitchIdentity{}, errors.New("Twitch OAuth application changed during authorization; please reconnect")
	}
	if active := e.state.Active; active != nil && !active.DryRun &&
		(active.BroadcasterID == "" || active.BroadcasterID != me.UserID) {
		return TwitchIdentity{}, errors.New("Finish or cancel the active Twitch Prediction before connecting another channel")
	}
	if uncertain := e.state.UnconfirmedCreation; uncertain != nil && uncertain.BroadcasterID != me.UserID {
		return TwitchIdentity{}, errors.New("Reconcile the unconfirmed Prediction before connecting another Twitch channel")
	}
	t := e.twitch
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.saveToken(token); err != nil {
		return TwitchIdentity{}, err
	}
	t.broadcasterID, t.login = me.UserID, me.Login
	return me, nil
}

// acknowledgeUnconfirmedCreation is the only path that can clear an
// uncertain creation without a known Prediction ID. The user must explicitly
// confirm the check, and Twitch must verify that this same broadcaster has
// no ACTIVE/LOCKED Prediction. No matching by title or blind adoption.
func (e *Engine) acknowledgeUnconfirmedCreation(ctx context.Context) error {
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
	identity, err := twitch.preflightCreate(ctx)
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

func (e *Engine) updateAndSave(s State) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.store.Save(s); err != nil { return err }
	e.state = s
	return nil
}
func (e *Engine) doCreate(ctx context.Context, s *State, lobbyKey string) error {
	return e.doCreatePrepared(ctx, s, lobbyKey, "")
}

func (e *Engine) doCreatePrepared(ctx context.Context, s *State, lobbyKey, broadcasterID string) error {
	if s.Active != nil {
		return nil
	}
	var p *Prediction
	var er error
	if !e.cfg.RealMode {
		p = &Prediction{ID: "SIMULATION", DryRun: true, StartedAt: max(e.clock.Now().UnixMilli(), s.LastSettlementAt+1)}
		e.log(tr(e.cfg.lang(), "log.sim_created"))
	} else {
		title, winTitle, lossTitle := e.cfg.predictionTexts()
		if broadcasterID != "" {
			p, er = e.predictions.createPrepared(ctx, broadcasterID, e.cfg.predictionWindowMinutes(), title, winTitle, lossTitle)
		} else {
			p, er = e.predictions.create(ctx, e.cfg.predictionWindowMinutes(), title, winTitle, lossTitle)
		}
		if er != nil {
			return er
		}
		e.log(tr(e.cfg.lang(), "log.prediction_created", p.ID))
	}
	next, err := confirmedCreationState(*s, p, bindPredictionMatch(s.LastLobby, lobbyKey, e.cfg.PlayerToonHandle))
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
func (e *Engine) doSettle(ctx context.Context, s *State, outcome string, source string) error {
 return e.settlePrediction(ctx, s, outcome, source)
}
// manualOutcome is serialized with the replay scanner. The caller supplies the
// exact active prediction identity so a delayed click can never settle a newer game.
func (e *Engine) manualOutcome(ctx context.Context, outcome, expectedID string, expectedStartedAt int64) error {
 if outcome != "win" && outcome != "loss" && outcome != "cancel" && outcome != "sync" {
  return errors.New(tr(e.cfg.lang(), "manual.invalid"))
 }
 if expectedID == "" || expectedStartedAt <= 0 {
  return errors.New(tr(e.cfg.lang(), "manual.stale"))
 }
 e.actionMu.Lock()
 defer e.actionMu.Unlock()
 e.mu.Lock()
 s := e.state
 e.mu.Unlock()
 if s.Active == nil {
  return errors.New(tr(e.cfg.lang(), "settle.none_active"))
 }
 if s.Active.ID != expectedID || s.Active.StartedAt != expectedStartedAt {
  return errors.New(tr(e.cfg.lang(), "manual.stale"))
 }
 if outcome == "sync" {
  if s.Active.DryRun {
   return errors.New(tr(e.cfg.lang(), "manual.sync_simulation"))
  }
  remote, err := e.predictions.prediction(ctx, s.Active.BroadcasterID, s.Active.ID)
  if err != nil { return err }
  for _, p := range remote.Data {
   if p.ID != s.Active.ID { continue }
   if p.Status != "RESOLVED" && p.Status != "CANCELED" {
    return errors.New(tr(e.cfg.lang(), "manual.still_open"))
   }
   // Use the same immutable completion transition as automatic settlement.
   // The result mapping also stays shared with recovery from an external
   // Twitch resolution; do not maintain a second outcome state machine.
   next, err := completedPredictionState(s, remoteOutcome(p, s.Active), "Twitch", e.clock.Now())
   if err != nil { return err }
   if err := e.updateAndSave(next); err != nil { return err }
   e.report("info", "prediction_synced", tr(e.cfg.lang(), "manual.synced", p.Status))
   return nil
  }
  return errors.New(tr(e.cfg.lang(), "manual.remote_missing"))
 }
 if err := e.doSettle(ctx, &s, outcome, "manual"); err != nil {
  e.report("error", "prediction_settle", tr(e.cfg.lang(), "manual.failed", err.Error()))
  return err
 }
 e.report("info", "prediction_manual", tr(e.cfg.lang(), "manual.completed", s.LastResult))
 return nil
}

func (e *Engine) start() error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return errors.New(tr(e.cfg.lang(), "automation.already_running"))
	}
	if e.state.UnconfirmedCreation != nil {
		e.mu.Unlock()
		return errors.New("Reconcile the unconfirmed Twitch Prediction before starting automation")
	}
	if e.cfg.PlayerName == "" && e.cfg.PlayerToonHandle == "" {
		e.mu.Unlock()
		return errors.New(tr(e.cfg.lang(), "automation.need_player"))
	}
	if e.cfg.twitchClientID() == "" && e.cfg.RealMode {
		e.mu.Unlock()
		return errors.New(tr(e.cfg.lang(), "automation.need_client_id"))
	}
	if e.state.Active != nil && e.state.Active.DryRun == e.cfg.RealMode {
		e.mu.Unlock()
		return errors.New(tr(e.cfg.lang(), "automation.mode_conflict"))
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.running = true
	e.stop = cancel
	e.mu.Unlock()
	go e.watch(ctx)
	return nil
}
func (e *Engine) stopWatch() {
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
		lang := e.cfg.lang()
		e.mu.Unlock()
		e.log(tr(lang, "log.watch_stopped"))
	}()
	// Snapshot startup values under the engine lock. A safe live settings
	// change can otherwise replace cfg/twitch during OAuth validation.
	e.mu.Lock()
	config, twitch := e.cfg, e.twitch
	e.mu.Unlock()
	lang := config.lang()
	e.log(tr(lang, "log.watch_started", tr(lang, map[bool]string{true: "mode.real", false: "mode.simulation"}[config.RealMode])))
	if config.RealMode {
		if _, err := twitch.identity(ctx); err != nil {
			e.report("error", "twitch_auth", "Twitch: " + err.Error())
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
 if ctx.Err() != nil { return }
 e.mu.Lock()
 s := e.state
 e.mu.Unlock()
 e.scanLifecycle(ctx, &s, started, e.clock.Now(), lastError, lastLog)
}

func (e *Engine) setReplayScan(code, file, detail string) {
 e.mu.Lock()
 e.scan = ReplayScan{Code: code, File: file, Detail: detail, CheckedAt: e.clock.Now().UnixMilli()}
 e.mu.Unlock()
}

// scanReplay applies the same correlation gate for background and manual rechecks.
func (e *Engine) scanReplay(ctx context.Context, s *State, now time.Time, lastError *string, lastLog *time.Time) {
 if s.Active == nil { return }
 match, code, file, detail := correlatePrediction(e.cfg, s.Active, now)
 if match == nil {
  if code == "correlation_ambiguous" && s.Active.Match != nil && !s.Active.Match.Ambiguous {
   s.Active = ambiguousMatch(s.Active)
   if err := e.updateAndSave(*s); err != nil { e.mu.Lock(); e.state = *s; e.mu.Unlock(); e.report("error", "correlation_save", err.Error()) }
  }
  e.setReplayScan(code, file, detail)
  return
 }
 if err := ctx.Err(); err != nil { return }
 if err := e.doSettle(ctx, s, match.Player.Result, match.Path); err != nil {
  e.setReplayScan("settle_error", filepath.Base(match.Path), err.Error())
  e.report("error", "prediction_settle", tr(e.cfg.lang(), "log.cancel_error", err.Error()))
  return
 }
 e.setReplayScan("settled", filepath.Base(match.Path), "")
 e.log(tr(e.cfg.lang(), "log.settled", s.LastResult))
}
// recheckReplay only evaluates completed replay files. It never creates a new
// Prediction and refuses stale UI requests for a different active game.
func (e *Engine) recheckReplay(ctx context.Context, expectedID string, expectedStartedAt int64) (ReplayScan, error) {
 e.actionMu.Lock()
 defer e.actionMu.Unlock()
 e.mu.Lock()
 s := e.state
 e.mu.Unlock()
 if s.Active == nil || s.Active.ID != expectedID || s.Active.StartedAt != expectedStartedAt || expectedID == "" {
  return ReplayScan{}, errors.New(tr(e.cfg.lang(), "manual.stale"))
 }
 lastError := ""
 lastLog := time.Time{}
 e.scanReplay(ctx, &s, e.clock.Now(), &lastError, &lastLog)
 e.mu.Lock()
 status := e.scan
 e.mu.Unlock()
 return status, nil
}
