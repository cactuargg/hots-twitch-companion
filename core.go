package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const scope = "channel:manage:predictions"
const authURL = "https://id.twitch.tv/oauth2"
const apiURL = "https://api.twitch.tv/helix"

type Config struct {
	ClientID            string `json:"clientId"`
	PlayerName          string `json:"playerName"`
	PlayerToonHandle    string `json:"playerToonHandle"`
	ReplayRoot          string `json:"replayRoot"`
	RealMode            bool   `json:"realMode"`
	Lang                string `json:"lang"`
	PredictionWindowMin int    `json:"predictionWindowMin"`
	WizardDone          bool   `json:"wizardDone"`
	PredictionTitle     string `json:"predictionTitle"`
	PredictionWin       string `json:"predictionWin"`
	PredictionLoss      string `json:"predictionLoss"`
}

const defaultPredictionWindow = 2
const maxPredictionWindow = 30

func (c Config) lang() string { return normalizeLang(c.Lang) }

// Duration of the betting window in whole minutes, like the Twitch predictions UI.
func (c Config) predictionWindowMinutes() int {
	if c.PredictionWindowMin < 1 {
		return defaultPredictionWindow
	}
	if c.PredictionWindowMin > maxPredictionWindow {
		return maxPredictionWindow
	}
	return c.PredictionWindowMin
}

// Prediction texts with user overrides; empty fields fall back to localized defaults.
func (c Config) predictionTexts() (title, win, loss string) {
	title = strings.TrimSpace(c.PredictionTitle)
	win = strings.TrimSpace(c.PredictionWin)
	loss = strings.TrimSpace(c.PredictionLoss)
	if title == "" {
		title = tr(c.lang(), "prediction.title")
	}
	if win == "" {
		win = tr(c.lang(), "prediction.win")
	}
	if loss == "" {
		loss = tr(c.lang(), "prediction.loss")
	}
	return title, win, loss
}
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	SavedAt      int64  `json:"savedAt"`
}
type Prediction struct {
	ID            string `json:"id"`
	BroadcasterID string `json:"broadcasterId"`
	WinID         string `json:"winId"`
	LossID        string `json:"lossId"`
	StartedAt     int64  `json:"startedAt"`
	DryRun        bool   `json:"dryRun"`
}
type State struct {
	Active           *Prediction `json:"active"`
	LastLobbyKey     string      `json:"lastLobbyKey"`
	LastSettlementAt int64       `json:"lastSettlementAt"`
	LastResult       string      `json:"lastResult"`
}
type DeviceResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}
type TwitchIdentity struct {
	ClientID string   `json:"client_id"`
	UserID   string   `json:"user_id"`
	Login    string   `json:"login"`
	Scopes   []string `json:"scopes"`
}
type apiResponse struct {
	Data []struct {
		ID               string `json:"id"`
		Status           string `json:"status"`
		Title            string `json:"title"`
		WinningOutcomeID string `json:"winning_outcome_id"`
		Outcomes         []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"outcomes"`
	} `json:"data"`
}
type paths struct{ dir, config, secret, state string }

func pathsForUser() (paths, error) {
	home := os.Getenv("LOCALAPPDATA")
	if home == "" {
		var e error
		home, e = os.UserConfigDir()
		if e != nil {
			return paths{}, e
		}
	}
	dir := filepath.Join(home, "HotSTwitchCompanion")
	return paths{dir, filepath.Join(dir, "config.json"), filepath.Join(dir, "tokens.dpapi"), filepath.Join(dir, "state.json")}, nil
}
func saveJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	tmp := path + ".new"
	if e = os.WriteFile(tmp, b, 0600); e != nil {
		return e
	}
	return os.Rename(tmp, path)
}
func loadJSON(path string, v any) error {
	b, e := os.ReadFile(path)
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
func readConfig(p paths) (Config, error)  { var c Config; e := loadJSON(p.config, &c); return c, e }
func writeConfig(p paths, c Config) error { return saveJSON(p.config, c) }
func loadState(p paths) (State, error)    { var s State; e := loadJSON(p.state, &s); return s, e }
func saveState(p paths, s State) error    { return saveJSON(p.state, s) }

var httpClient = &http.Client{Timeout: 20 * time.Second}

func requestJSON(ctx context.Context, method, urlPath string, form url.Values, body any, token, clientID string, dst any) error {
	var r io.Reader
	if form != nil {
		r = strings.NewReader(form.Encode())
	} else if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			return e
		}
		r = bytes.NewReader(b)
	}
	req, e := http.NewRequestWithContext(ctx, method, urlPath, r)
	if e != nil {
		return e
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		if strings.HasSuffix(urlPath, "/validate") {
			req.Header.Set("Authorization", "OAuth "+token)
		} else {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	if clientID != "" {
		req.Header.Set("Client-Id", clientID)
	}
	res, e := httpClient.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if e != nil {
		return e
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var errBody struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.Unmarshal(raw, &errBody)
		msg := errBody.Message
		if msg == "" {
			msg = errBody.Error
		}
		if msg == "" {
			msg = res.Status
		}
		return fmt.Errorf("Twitch HTTP %d: %s", res.StatusCode, msg)
	}
	if dst != nil && len(raw) > 0 {
		return json.Unmarshal(raw, dst)
	}
	return nil
}
func startDevice(ctx context.Context, clientID string) (DeviceResponse, error) {
	var d DeviceResponse
	e := requestJSON(ctx, "POST", authURL+"/device", url.Values{"client_id": {clientID}, "scopes": {scope}}, nil, "", "", &d)
	return d, e
}
func pollDevice(ctx context.Context, clientID, lang string, d DeviceResponse, log func(string)) (Token, error) {
	if d.Interval < 5 {
		d.Interval = 5
	}
	deadline := time.NewTimer(time.Duration(d.ExpiresIn) * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Duration(d.Interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return Token{}, ctx.Err()
		case <-deadline.C:
			return Token{}, errors.New(tr(lang, "auth.code_expired"))
		case <-ticker.C:
			var t Token
			form := url.Values{"client_id": {clientID}, "scopes": {scope}, "device_code": {d.DeviceCode}, "grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}}
			e := requestJSON(ctx, "POST", authURL+"/token", form, nil, "", "", &t)
			if e == nil {
				if t.AccessToken == "" || t.RefreshToken == "" {
					return Token{}, errors.New(tr(lang, "auth.no_tokens"))
				}
				t.SavedAt = time.Now().Unix()
				return t, nil
			}
			if strings.Contains(e.Error(), "authorization_pending") || strings.Contains(e.Error(), "slow_down") {
				continue
			}
			return Token{}, e
		}
	}
}

type Twitch struct {
	mu                   sync.Mutex
	clientID             string
	lang                 string
	p                    paths
	token                Token
	broadcasterID, login string
}

func newTwitch(p paths, clientID, lang string) (*Twitch, error) {
	t := &Twitch{clientID: clientID, lang: normalizeLang(lang), p: p}
	b, e := os.ReadFile(p.secret)
	if errors.Is(e, os.ErrNotExist) {
		return t, nil
	}
	if e != nil {
		return nil, e
	}
	clear, e := unprotect(b)
	if e != nil {
		return nil, fmt.Errorf("Windows DPAPI: %w", e)
	}
	if e = json.Unmarshal(clear, &t.token); e != nil {
		return nil, e
	}
	return t, nil
}
func (t *Twitch) saveToken(token Token) error {
	b, e := json.Marshal(token)
	if e != nil {
		return e
	}
	sealed, e := protect(b)
	if e != nil {
		return e
	}
	if e = os.WriteFile(t.p.secret+".new", sealed, 0600); e != nil {
		return e
	}
	if e = os.Rename(t.p.secret+".new", t.p.secret); e != nil {
		return e
	}
	t.token = token
	return nil
}
func (t *Twitch) tokenFor(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.token.AccessToken == "" {
		return "", errors.New(tr(t.lang, "twitch.connect_first"))
	}
	if time.Since(time.Unix(t.token.SavedAt, 0)) < time.Duration(t.token.ExpiresIn-300)*time.Second {
		return t.token.AccessToken, nil
	}
	if t.token.RefreshToken == "" {
		return "", errors.New(tr(t.lang, "twitch.token_expired"))
	}
	form := url.Values{"client_id": {t.clientID}, "grant_type": {"refresh_token"}, "refresh_token": {t.token.RefreshToken}}
	var token Token
	if e := requestJSON(ctx, "POST", authURL+"/token", form, nil, "", "", &token); e != nil {
		return "", errors.New(tr(t.lang, "twitch.refresh_error", e.Error()))
	}
	if token.AccessToken == "" || token.RefreshToken == "" {
		return "", errors.New(tr(t.lang, "twitch.incomplete_tokens"))
	}
	token.SavedAt = time.Now().Unix()
	if e := t.saveToken(token); e != nil {
		return "", e
	}
	return token.AccessToken, nil
}
func (t *Twitch) identity(ctx context.Context) (TwitchIdentity, error) {
	token, e := t.tokenFor(ctx)
	if e != nil {
		return TwitchIdentity{}, e
	}
	var me TwitchIdentity
	e = requestJSON(ctx, "GET", authURL+"/validate", nil, nil, token, "", &me)
	if e != nil {
		return me, e
	}
	if me.ClientID != t.clientID {
		return me, errors.New(tr(t.lang, "twitch.wrong_client"))
	}
	valid := false
	for _, v := range me.Scopes {
		if v == scope {
			valid = true
		}
	}
	if !valid {
		return me, errors.New(tr(t.lang, "twitch.no_scope"))
	}
	t.mu.Lock()
	t.broadcasterID = me.UserID
	t.login = me.Login
	t.mu.Unlock()
	return me, nil
}
func (t *Twitch) req(ctx context.Context, method, route string, body any, dst any) error {
	token, e := t.tokenFor(ctx)
	if e != nil {
		return e
	}
	e = requestJSON(ctx, method, apiURL+route, nil, body, token, t.clientID, dst)
	if e != nil && strings.Contains(e.Error(), "HTTP 401") {
		t.mu.Lock()
		t.token.SavedAt = 0
		t.mu.Unlock()
		token, e = t.tokenFor(ctx)
		if e != nil {
			return e
		}
		return requestJSON(ctx, method, apiURL+route, nil, body, token, t.clientID, dst)
	}
	return e
}
func (t *Twitch) prediction(ctx context.Context, broadcaster, id string) (*apiResponse, error) {
	route := "/predictions?broadcaster_id=" + url.QueryEscape(broadcaster) + "&first=5"
	if id != "" {
		route = "/predictions?broadcaster_id=" + url.QueryEscape(broadcaster) + "&id=" + url.QueryEscape(id)
	}
	var d apiResponse
	e := t.req(ctx, "GET", route, nil, &d)
	return &d, e
}
func (t *Twitch) create(ctx context.Context, windowMinutes int, title, winTitle, lossTitle string) (*Prediction, error) {
	me, e := t.identity(ctx)
	if e != nil {
		return nil, e
	}
	previous, e := t.prediction(ctx, me.UserID, "")
	if e != nil {
		return nil, e
	}
	for _, v := range previous.Data {
		if v.Status == "ACTIVE" || v.Status == "LOCKED" {
			return nil, errors.New(tr(t.lang, "twitch.prediction_exists"))
		}
	}
	var d apiResponse
	e = t.req(ctx, "POST", "/predictions", map[string]any{"broadcaster_id": me.UserID, "title": title, "outcomes": []map[string]string{{"title": winTitle}, {"title": lossTitle}}, "prediction_window": windowMinutes * 60}, &d)
	if e != nil {
		return nil, e
	}
	if len(d.Data) != 1 || len(d.Data[0].Outcomes) != 2 {
		return nil, errors.New(tr(t.lang, "twitch.create_bad_response"))
	}
	p := &Prediction{ID: d.Data[0].ID, BroadcasterID: me.UserID, StartedAt: time.Now().UnixMilli()}
	for _, o := range d.Data[0].Outcomes {
		if o.Title == winTitle {
			p.WinID = o.ID
		}
		if o.Title == lossTitle {
			p.LossID = o.ID
		}
	}
	if p.ID == "" || p.WinID == "" || p.LossID == "" || p.WinID == p.LossID {
		return nil, errors.New(tr(t.lang, "twitch.outcomes_invalid"))
	}
	return p, nil
}
func (t *Twitch) end(ctx context.Context, p *Prediction, outcome string) error {
	status := "RESOLVED"
	if outcome == "cancel" {
		status = "CANCELED"
	}
	req := map[string]any{"broadcaster_id": p.BroadcasterID, "id": p.ID, "status": status}
	win := ""
	if outcome != "cancel" {
		win = p.LossID
		if outcome == "win" {
			win = p.WinID
		}
		req["winning_outcome_id"] = win
	}
	var d apiResponse
	e := t.req(ctx, "PATCH", "/predictions", req, &d)
	if e == nil {
		return nil
	}
	// A timed-out PATCH could have succeeded: verify before attempting any retry.
	actual, er := t.prediction(ctx, p.BroadcasterID, p.ID)
	if er == nil {
		for _, v := range actual.Data {
			if v.ID == p.ID && v.Status == status && (win == "" || v.WinningOutcomeID == win) {
				return nil
			}
		}
	}
	return e
}

type ReplayFile struct {
	Path     string
	Modified time.Time
	Size     int64
}

func replayRoots(cfg Config) []string {
	if cfg.ReplayRoot != "" {
		return []string{cfg.ReplayRoot}
	}
	home, _ := os.UserHomeDir()
	return []string{filepath.Join(home, "Documents", "Heroes of the Storm"), filepath.Join(home, "OneDrive", "Documents", "Heroes of the Storm")}
}
func replaysAfter(cfg Config, after time.Time, max int) []ReplayFile {
	var found []ReplayFile
	visited := 0
	for _, root := range replayRoots(cfg) {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return filepath.SkipDir
			}
			visited++
			if visited > 20000 {
				return filepath.SkipAll
			}
			if d.IsDir() {
				rel, er := filepath.Rel(root, path)
				if er == nil && strings.Count(rel, string(os.PathSeparator)) > 8 {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(strings.ToLower(d.Name()), ".stormreplay") {
				st, e := d.Info()
				if e == nil && st.ModTime().After(after) {
					found = append(found, ReplayFile{path, st.ModTime(), st.Size()})
				}
			}
			return nil
		})
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Modified.After(found[j].Modified) })
	if len(found) > max {
		return found[:max]
	}
	return found
}
func findLobby() (string, string, time.Time, error) {
	root := filepath.Join(os.TempDir(), "Heroes of the Storm")
	var newest string
	var st os.FileInfo
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return filepath.SkipDir
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(root, p)
			if strings.Count(rel, string(os.PathSeparator)) >= 3 {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(d.Name(), "replay.server.battlelobby") {
			s, err := d.Info()
			if err == nil && s.Size() > 0 && (st == nil || s.ModTime().After(st.ModTime())) {
				newest = p
				st = s
			}
		}
		return nil
	})
	if st == nil {
		return "", "", time.Time{}, errors.New(tr(langEn, "lobby.not_found"))
	}
	key := fmt.Sprintf("%s|%d|%d", newest, st.ModTime().UnixNano(), st.Size())
	return newest, key, st.ModTime(), nil
}

type Engine struct {
	mu      sync.Mutex
	actionMu sync.Mutex
	p       paths
	cfg     Config
	twitch  *Twitch
	state   State
	running bool
	stop    context.CancelFunc
	log      func(string)
	logEvent func(level, code, message string)
}

func newEngine(p paths, cfg Config, log func(string)) (*Engine, error) {
	s, e := loadState(p)
	if e != nil {
		return nil, e
	}
	t, e := newTwitch(p, cfg.ClientID, cfg.lang())
	if e != nil {
		return nil, e
	}
	return &Engine{p: p, cfg: cfg, state: s, twitch: t, log: log}, nil
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
	if cfg.lang() == e.cfg.lang() && cfg.WizardDone == e.cfg.WizardDone {
		return false
	}
	return cfg.ClientID == e.cfg.ClientID &&
		cfg.PlayerName == e.cfg.PlayerName &&
		cfg.PlayerToonHandle == e.cfg.PlayerToonHandle &&
		cfg.ReplayRoot == e.cfg.ReplayRoot &&
		cfg.RealMode == e.cfg.RealMode &&
		cfg.PredictionWindowMin == e.cfg.PredictionWindowMin &&
		cfg.PredictionTitle == e.cfg.PredictionTitle &&
		cfg.PredictionWin == e.cfg.PredictionWin &&
		cfg.PredictionLoss == e.cfg.PredictionLoss
}
func (e *Engine) updateConfig(cfg Config) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running && !e.safeLiveChange(cfg) {
		return errors.New(tr(e.cfg.lang(), "config.stop_watch_first"))
	}
	if e.state.Active != nil && e.state.Active.DryRun != !cfg.RealMode {
		return errors.New(tr(e.cfg.lang(), "config.mode_conflict_active"))
	}
	if er := writeConfig(e.p, cfg); er != nil {
		return er
	}
	t, er := newTwitch(e.p, cfg.ClientID, cfg.lang())
	if er != nil {
		return er
	}
	e.cfg = cfg
	e.twitch = t
	return nil
}
func (e *Engine) updateAndSave(s State) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state = s
	return saveState(e.p, s)
}
func (e *Engine) doCreate(ctx context.Context, s *State) error {
	if s.Active != nil {
		return nil
	}
	var p *Prediction
	var er error
	if !e.cfg.RealMode {
		p = &Prediction{ID: "SIMULATION", DryRun: true, StartedAt: time.Now().UnixMilli()}
		e.log(tr(e.cfg.lang(), "log.sim_created"))
	} else {
		title, winTitle, lossTitle := e.cfg.predictionTexts()
		p, er = e.twitch.create(ctx, e.cfg.predictionWindowMinutes(), title, winTitle, lossTitle)
		if er != nil {
			return er
		}
		e.log(tr(e.cfg.lang(), "log.prediction_created", p.ID))
	}
	s.Active = p
	return e.updateAndSave(*s)
}
func (e *Engine) doSettle(ctx context.Context, s *State, outcome string, source string) error {
	if s.Active == nil {
		return errors.New(tr(e.cfg.lang(), "settle.none_active"))
	}
	if s.Active.DryRun {
		e.log(tr(e.cfg.lang(), "log.sim_outcome", outcome))
	} else if er := e.twitch.end(ctx, s.Active, outcome); er != nil {
		return er
	}
	s.LastSettlementAt = time.Now().UnixMilli()
	s.LastResult = fmt.Sprintf("%s (%s)", outcome, filepath.Base(source))
	s.Active = nil
	return e.updateAndSave(*s)
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
  remote, err := e.twitch.prediction(ctx, s.Active.BroadcasterID, s.Active.ID)
  if err != nil { return err }
  for _, p := range remote.Data {
   if p.ID != s.Active.ID { continue }
   if p.Status != "RESOLVED" && p.Status != "CANCELED" {
    return errors.New(tr(e.cfg.lang(), "manual.still_open"))
   }
   result := "cancel"
   if p.Status == "RESOLVED" {
    if p.WinningOutcomeID == s.Active.WinID { result = "win" } else if p.WinningOutcomeID == s.Active.LossID { result = "loss" } else {
     result = "external"
    }
   }
   s.LastSettlementAt = time.Now().UnixMilli()
   s.LastResult = result + " (Twitch)"
   s.Active = nil
   if err := e.updateAndSave(s); err != nil { return err }
   e.report("info", "prediction_synced", tr(e.cfg.lang(), "manual.synced", p.Status))
   return nil
  }
  return errors.New(tr(e.cfg.lang(), "manual.remote_missing"))
 }
 if err := e.doSettle(ctx, &s, outcome, "manual"); err != nil {
  e.report("error", "prediction_settle", tr(e.cfg.lang(), "manual.failed", err.Error()))
  return err
 }
 e.report("info", "prediction_manual", tr(e.cfg.lang(), "manual.completed", outcome))
 return nil
}

func (e *Engine) start() error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return errors.New(tr(e.cfg.lang(), "automation.already_running"))
	}
	if e.cfg.PlayerName == "" && e.cfg.PlayerToonHandle == "" {
		e.mu.Unlock()
		return errors.New(tr(e.cfg.lang(), "automation.need_player"))
	}
	if e.cfg.ClientID == "" && e.cfg.RealMode {
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
		e.mu.Unlock()
		e.log(tr(e.cfg.lang(), "log.watch_stopped"))
	}()
	lang := e.cfg.lang()
	e.log(tr(lang, "log.watch_started", tr(lang, map[bool]string{true: "mode.real", false: "mode.simulation"}[e.cfg.RealMode])))
	if e.cfg.RealMode {
		if _, err := e.twitch.identity(ctx); err != nil {
			e.report("error", "twitch_auth", "Twitch: " + err.Error())
			return
		}
	}
	e.mu.Lock()
	s := e.state
	e.mu.Unlock()
	started := time.Now()
	lastError := ""
	lastLog := time.Time{}
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
	now := time.Now()
	path, key, modified, err := findLobby()
	if err == nil && key != s.LastLobbyKey && now.Sub(modified) > 2500*time.Millisecond && now.Sub(modified) < 120*time.Second {
		// Only react to a new/changed lobby *after* watch began.
		if !modified.Before(started.Add(-2 * time.Second)) {
			if s.Active == nil && now.Sub(time.UnixMilli(s.LastSettlementAt)) > 2*time.Minute {
				if er := e.doCreate(ctx, &s); er != nil {
					e.report("error", "prediction_create", tr(e.cfg.lang(), "log.create_error", er.Error()))
					continue
				}
				e.log(tr(e.cfg.lang(), "log.new_game", filepath.Base(filepath.Dir(path))))
			} else {
				e.log(tr(e.cfg.lang(), "log.skip_lobby"))
			}
		}
		s.LastLobbyKey = key
		_ = e.updateAndSave(s)
	}
	if s.Active == nil {
		return
	}
	age := now.Sub(time.UnixMilli(s.Active.StartedAt))
	if age > 100*time.Minute {
		e.report("warning", "prediction_timeout", tr(e.cfg.lang(), "log.timeout_cancel"))
		if er := e.doSettle(ctx, &s, "cancel", "timeout"); er != nil {
			e.report("error", "prediction_settle", tr(e.cfg.lang(), "log.cancel_error", er.Error()))
		}
		return
	}
	for _, f := range replaysAfter(e.cfg, time.UnixMilli(s.Active.StartedAt+1500), 8) {
		if now.Sub(f.Modified) < 8*time.Second || f.Size < 100 {
			continue
		}
		replay, er := parseReplay(f.Path)
		if er == nil {
			var p Player
			p, er = findOutcome(replay, e.cfg.PlayerName, e.cfg.PlayerToonHandle, e.cfg.lang())
			if er == nil {
				e.log(tr(e.cfg.lang(), "log.result", p.Name, p.Result, replay.Map))
				if er = e.doSettle(ctx, &s, p.Result, f.Path); er == nil {
					e.log(tr(e.cfg.lang(), "log.settled", p.Result))
					break
				}
			}
		}
		if er != nil && (er.Error() != *lastError || time.Since(*lastLog) > time.Minute) {
			e.report("warning", "replay_process", tr(e.cfg.lang(), "log.process_error", filepath.Base(f.Path), er.Error()))
			*lastError = er.Error()
			*lastLog = time.Now()
		}
	}
	}

