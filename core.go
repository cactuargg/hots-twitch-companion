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
const appVersion = "0.9.6"
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

// Public OAuth Client ID may be embedded by release builds using -ldflags.
// Do not embed Client Secret, access tokens, or refresh tokens.
var defaultTwitchClientID string

func (c Config) twitchClientID() string {
 if id := strings.TrimSpace(c.ClientID); id != "" { return id }
 return strings.TrimSpace(defaultTwitchClientID)
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
	Match         *MatchBinding `json:"match,omitempty"`
	Settlement    *SettlementAttempt `json:"settlement,omitempty"`
}
// Persisted proof that a real Twitch creation POST may have been sent
// without a confirmed remote ID. Never automatically retry this POST.
type CreationAttempt struct {
	LobbySHA string `json:"lobbySHA"`
	BroadcasterID string `json:"broadcasterId"`
	AttemptedAt int64 `json:"attemptedAt"`
}

type State struct {
	UnconfirmedCreation *CreationAttempt `json:"unconfirmedCreation,omitempty"`
	Active           *Prediction `json:"active"`
	LastLobby        *LobbySnapshot `json:"lastLobby,omitempty"`
	LastLobbyKey     string      `json:"lastLobbyKey"`
	LastSettlementAt int64       `json:"lastSettlementAt"`
	LastResult       string      `json:"lastResult"`
	PendingLobby     *LobbyEvent `json:"pendingLobby,omitempty"`
	PendingLobbies  []*LobbyEvent `json:"pendingLobbies,omitempty"`
	LastCreatedLobbySHA string `json:"lastCreatedLobbySHA,omitempty"`
	RecentLobbySHAs []string `json:"recentLobbySHAs,omitempty"`
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
type apiPrediction struct {
		ID               string `json:"id"`
		Status           string `json:"status"`
		Title            string `json:"title"`
		WinningOutcomeID string `json:"winning_outcome_id"`
		Outcomes         []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"outcomes"`
}
type apiResponse struct { Data []apiPrediction `json:"data"` }
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
	f, e := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if e != nil { return e }
	if _, e = f.Write(b); e == nil { e = f.Sync() }
	closeErr := f.Close()
	if e != nil { return e }
	if closeErr != nil { return closeErr }
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
func readConfig(p paths) (Config, error) {
	var c Config
	err := loadJSON(p.config, &c)
	// Desktop profiles always use Twitch; dry runs are an internal test facility.
	c.RealMode = true
	return c, err
}
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
		return responseHTTPError(res, msg)
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
// Twitch asks Device Flow clients to increase their polling delay by five
// seconds each time the OAuth endpoint returns slow_down.
func devicePollRetry(interval time.Duration, err error) (time.Duration, bool) {
	if err == nil {
		return interval, false
	}
	if strings.Contains(err.Error(), "slow_down") {
		return interval + 5*time.Second, true
	}
	if strings.Contains(err.Error(), "authorization_pending") {
		return interval, true
	}
	return interval, false
}

func pollDevice(ctx context.Context, clientID, lang string, d DeviceResponse, log func(string)) (Token, error) {
	if d.Interval < 5 {
		d.Interval = 5
	}
	interval := time.Duration(d.Interval) * time.Second
	deadline := time.NewTimer(time.Duration(d.ExpiresIn) * time.Second)
	defer deadline.Stop()
	wait := time.NewTimer(interval)
	defer wait.Stop()
	for {
		select {
		case <-ctx.Done():
			return Token{}, ctx.Err()
		case <-deadline.C:
			return Token{}, errors.New(tr(lang, "auth.code_expired"))
		case <-wait.C:
			var t Token
			form := url.Values{"client_id": {clientID}, "scopes": {scope}, "device_code": {d.DeviceCode}, "grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}}
			err := requestJSON(ctx, "POST", authURL+"/token", form, nil, "", "", &t)
			if err == nil {
				if t.AccessToken == "" || t.RefreshToken == "" {
					return Token{}, errors.New(tr(lang, "auth.no_tokens"))
				}
				t.SavedAt = time.Now().Unix()
				return t, nil
			}
			next, retry := devicePollRetry(interval, err)
			if !retry {
				return Token{}, err
			}
			interval = next
			wait.Reset(interval)
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
// validateTwitchIdentity verifies candidate credentials without changing
// the persisted token. A device authorization must not replace the currently
// connected broadcaster before this check completes.
func validateTwitchIdentity(ctx context.Context, accessToken, clientID, lang string) (TwitchIdentity, error) {
	var me TwitchIdentity
	if err := requestJSON(ctx, "GET", authURL+"/validate", nil, nil, accessToken, "", &me); err != nil {
		return me, err
	}
	if me.ClientID != clientID {
		return me, errors.New(tr(lang, "twitch.wrong_client"))
	}
	valid := false
	for _, v := range me.Scopes {
		if v == scope {
			valid = true
		}
	}
	if !valid {
		return me, errors.New(tr(lang, "twitch.no_scope"))
	}
	if me.UserID == "" {
		return me, errors.New("Twitch validation returned no broadcaster ID")
	}
	return me, nil
}

func (t *Twitch) identity(ctx context.Context) (TwitchIdentity, error) {
	token, err := t.tokenFor(ctx)
	if err != nil {
		return TwitchIdentity{}, err
	}
	me, err := validateTwitchIdentity(ctx, token, t.clientID, t.lang)
	if err != nil {
		// Failed validation must not leave a stale "connected" identity
		// visible after revoked credentials or a different OAuth client.
		t.mu.Lock()
		if t.token.AccessToken == token {
			t.login, t.broadcasterID = "", ""
		}
		t.mu.Unlock()
		return me, err
	}
	t.mu.Lock()
	if t.token.AccessToken == token {
		t.broadcasterID = me.UserID
		t.login = me.Login
	}
	t.mu.Unlock()
	return me, nil
}

func (t *Twitch) req(ctx context.Context, method, route string, body any, dst any) error {
	token, e := t.tokenFor(ctx)
	if e != nil {
		return e
	}
	e = requestJSON(ctx, method, apiURL+route, nil, body, token, t.clientID, dst)
	var httpError *twitchHTTPError
	if errors.As(e, &httpError) && httpError.Status == 401 {
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
// preflightCreate performs all fallible read-only checks before a lobby is
// consumed. A failure here is safe to retry: no creation POST has been sent.
func (t *Twitch) preflightCreate(ctx context.Context) (TwitchIdentity, error) {
	me, err := t.identity(ctx)
	if err != nil {
		return TwitchIdentity{}, err
	}
	previous, err := t.prediction(ctx, me.UserID, "")
	if err != nil {
		return TwitchIdentity{}, err
	}
	for _, v := range previous.Data {
		if v.Status == "ACTIVE" || v.Status == "LOCKED" {
			return TwitchIdentity{}, errors.New(tr(t.lang, "twitch.prediction_exists"))
		}
	}
	return me, nil
}

// createPrepared must only run after the caller durably records the creation
// attempt. A lost POST response must never trigger a second POST.
func (t *Twitch) createPrepared(ctx context.Context, broadcasterID string, windowMinutes int, title, winTitle, lossTitle string) (*Prediction, error) {
	if broadcasterID == "" || winTitle == lossTitle {
		return nil, errors.New(tr(t.lang, "twitch.outcomes_invalid"))
	}
	var d apiResponse
	err := t.req(ctx, "POST", "/predictions", map[string]any{
		"broadcaster_id": broadcasterID,
		"title": title,
		"outcomes": []map[string]string{{"title": winTitle}, {"title": lossTitle}},
		"prediction_window": windowMinutes * 60,
	}, &d)
	if err != nil {
		return nil, err
	}
	if len(d.Data) != 1 || len(d.Data[0].Outcomes) != 2 || d.Data[0].Status != "ACTIVE" {
		return nil, errors.New(tr(t.lang, "twitch.create_bad_response"))
	}
	p := &Prediction{ID: d.Data[0].ID, BroadcasterID: broadcasterID, StartedAt: time.Now().UnixMilli()}
	for _, outcome := range d.Data[0].Outcomes {
		if outcome.Title == winTitle {
			p.WinID = outcome.ID
		}
		if outcome.Title == lossTitle {
			p.LossID = outcome.ID
		}
	}
	if p.ID == "" || p.WinID == "" || p.LossID == "" || p.WinID == p.LossID {
		return nil, errors.New(tr(t.lang, "twitch.outcomes_invalid"))
	}
	return p, nil
}

func (t *Twitch) create(ctx context.Context, windowMinutes int, title, winTitle, lossTitle string) (*Prediction, error) {
	if winTitle == lossTitle {
		return nil, errors.New(tr(t.lang, "twitch.outcomes_invalid"))
	}
	me, err := t.preflightCreate(ctx)
	if err != nil {
		return nil, err
	}
	return t.createPrepared(ctx, me.UserID, windowMinutes, title, winTitle, lossTitle)
}
func (t *Twitch) end(ctx context.Context, p *Prediction, outcome string) error {
 return t.finishPrediction(ctx, p, outcome)
}

type ReplayFile struct {
	Path     string
	Modified time.Time
	Size     int64
}

func defaultReplayRoots() []string {
	home, _ := os.UserHomeDir()
	return []string{filepath.Join(home, "Documents", "Heroes of the Storm"), filepath.Join(home, "OneDrive", "Documents", "Heroes of the Storm")}
}
func replayRoots(cfg Config) []string {
	if cfg.ReplayRoot != "" {
		return []string{cfg.ReplayRoot}
	}
	return defaultReplayRoots()
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
	s, e := loadState(p)
	if e != nil {
		return nil, e
	}
	if s.LastCreatedLobbySHA == "" {
		if s.Active != nil && s.Active.Match != nil && s.Active.Match.Lobby != nil {
			s.LastCreatedLobbySHA = s.Active.Match.Lobby.SHA256
		} else if s.LastLobby != nil { s.LastCreatedLobbySHA = s.LastLobby.SHA256 }
	}
	if len(s.RecentLobbySHAs)==0 && s.LastCreatedLobbySHA!="" {s.RecentLobbySHAs=[]string{s.LastCreatedLobbySHA}}
	t, e := newTwitch(p, cfg.twitchClientID(), cfg.lang())
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
	if e.state.Active != nil && !e.state.Active.DryRun && cfg.twitchClientID() != e.cfg.twitchClientID() {
		return errors.New("Finish or cancel the active Twitch Prediction before changing the OAuth Client ID")
	}
	if er := writeConfig(e.p, cfg); er != nil {
		return er
	}
	t, er := newTwitch(e.p, cfg.twitchClientID(), cfg.lang())
	if er != nil {
		return er
	}
	// newTwitch restores the persisted token but not the verified identity,
	// which only identity() sets (app start or monitoring start). Saving
	// settings must not drop the connected login while the OAuth application
	// stays the same, or the UI bounces the user back to Twitch setup.
	if e.cfg.twitchClientID() == cfg.twitchClientID() {
		e.twitch.mu.Lock()
		login, broadcaster := e.twitch.login, e.twitch.broadcasterID
		e.twitch.mu.Unlock()
		t.mu.Lock()
		t.login, t.broadcasterID = login, broadcaster
		t.mu.Unlock()
	}
	e.cfg = cfg
	e.twitch = t
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
	t := e.twitch
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.saveToken(token); err != nil {
		return TwitchIdentity{}, err
	}
	t.broadcasterID, t.login = me.UserID, me.Login
	return me, nil
}

func (e *Engine) updateAndSave(s State) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := saveState(e.p, s); err != nil { return err }
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
		p = &Prediction{ID: "SIMULATION", DryRun: true, StartedAt: max(time.Now().UnixMilli(), s.LastSettlementAt+1)}
		e.log(tr(e.cfg.lang(), "log.sim_created"))
	} else {
		title, winTitle, lossTitle := e.cfg.predictionTexts()
		if broadcasterID != "" {
			p, er = e.twitch.createPrepared(ctx, broadcasterID, e.cfg.predictionWindowMinutes(), title, winTitle, lossTitle)
		} else {
			p, er = e.twitch.create(ctx, e.cfg.predictionWindowMinutes(), title, winTitle, lossTitle)
		}
		if er != nil {
			return er
		}
		e.log(tr(e.cfg.lang(), "log.prediction_created", p.ID))
	}
	p.Match = bindPredictionMatch(s.LastLobby, lobbyKey, e.cfg.PlayerToonHandle)
	s.Active = p
	// Commit the known ID and clear the uncertainty in one state write.
	s.UnconfirmedCreation = nil
 e.setReplayScan("waiting_replay", "", "")
	if err := e.updateAndSave(*s); err != nil {
		// Keep the known remote ID available in this process. The lobby event
		// was consumed durably before POST, so restart cannot repeat creation.
		e.mu.Lock(); e.state = *s; e.mu.Unlock()
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
 e.report("info", "prediction_manual", tr(e.cfg.lang(), "manual.completed", s.LastResult))
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
	started := time.Now()
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
 e.scanLifecycle(ctx, &s, started, time.Now(), lastError, lastLog)
}

func (e *Engine) setReplayScan(code, file, detail string) {
 e.mu.Lock()
 e.scan = ReplayScan{Code: code, File: file, Detail: detail, CheckedAt: time.Now().UnixMilli()}
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
 e.scanReplay(ctx, &s, time.Now(), &lastError, &lastLog)
 e.mu.Lock()
 status := e.scan
 e.mu.Unlock()
 return status, nil
}
