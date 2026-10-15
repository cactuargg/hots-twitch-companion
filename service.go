package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

//go:embed web/* web/assets/*
var interfaceFiles embed.FS

type event struct {
	Time    string `json:"time"`
	Message string `json:"message"`
	Level   string `json:"level"`
	Code    string `json:"code,omitempty"`
}
type replaySummary struct {
	File     string          `json:"file"`
	Map      string          `json:"map"`
	Result   string          `json:"result"`
	Modified string          `json:"modified"`
	Players  []playerSummary `json:"players,omitempty"`
	Selected string          `json:"selected,omitempty"`
	Error          string          `json:"error,omitempty"`
	SelectionError string          `json:"selectionError,omitempty"`
}
type playerSummary struct {
	Name       string `json:"name"`
	ToonHandle string `json:"toonHandle"`
	Result     string `json:"result"`
}

type dashboard struct {
	mu                                    sync.Mutex
	engine                                *Engine
	p                                     paths
	logs                                  []event
	authCode, authStatus, verificationURI string
	authPending                           bool
	replays                               []replaySummary
	replaysAt                             time.Time
	shutdown                              chan struct{}
	shutdownOnce                          sync.Once
	lastActivity                          time.Time
}

func newDashboard(p paths, c Config) (*dashboard, error) {
	d := &dashboard{p: p, shutdown: make(chan struct{})}
	e, err := newEngine(p, c, d.log)
	if err != nil {
		return nil, err
	}
	d.engine = e
	e.logEvent = d.logEvent
	// Verify restored tokens in the background; do not block the dashboard on network.
	if e.twitch.token.AccessToken != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			if who, err := e.twitch.identity(ctx); err == nil {
				d.log(tr(c.lang(), "log.twitch_connected", who.Login))
			} else {
				d.logEvent("warning", "twitch_auth", tr(c.lang(), "log.token_check", err.Error()))
			}
		}()
	}
	d.log(tr(c.lang(), "log.app_started"))
	d.log(tr(c.lang(), "log.read_only"))
	return d, nil
}
func (d *dashboard) tr(key string, args ...any) string {
	return tr(d.config().Lang, key, args...)
}
func (d *dashboard) log(s string) {
	d.logEvent("info", "", s)
}

func (d *dashboard) logEvent(level, code, message string) {
	if level != "warning" && level != "error" {
		level = "info"
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logs = append(d.logs, event{Time: time.Now().Format("15:04:05"), Level: level, Code: code, Message: message})
	if len(d.logs) > 150 {
		d.logs = d.logs[len(d.logs)-150:]
	}
}

// User-facing phases derived from persisted engine state. These describe what we
// know locally, not a live verification of Twitch's current Prediction status.
func connectionPhase(hasToken, validated, pending bool) string {
	if pending {
		return "authorizing"
	}
	if !hasToken {
		return "disconnected"
	}
	if validated {
		return "connected"
	}
	return "unverified"
}

func monitorPhase(running, tracking bool) string {
	if !running {
		return "off"
	}
	if tracking {
		return "tracking"
	}
	return "watching"
}

func predictionPhase(p *Prediction, windowMinutes int, now time.Time) (string, int) {
	if p == nil {
		return "none", 0
	}
	if p.DryRun {
		return "simulation", 0
	}
	// The betting window is an estimate: remote Twitch status is not polled here.
	deadline := time.UnixMilli(p.StartedAt).Add(time.Duration(windowMinutes) * time.Minute)
	remaining := deadline.Sub(now)
	if remaining <= 0 {
		return "awaiting_result", 0
	}
	return "window_estimate", int((remaining + time.Second - 1) / time.Second)
}

func (d *dashboard) status() map[string]any {
	e := d.engine
	e.mu.Lock()
	running, config, s := e.running, e.cfg, e.state
	tw := e.twitch
	e.mu.Unlock()
	tw.mu.Lock()
	// The encrypted DPAPI file is never sent to the UI.
	hasToken := tw.token.AccessToken != ""
	channel := tw.login
	validated := channel != ""
	connected := hasToken && validated
	tw.mu.Unlock()
	d.mu.Lock()
	logs := append([]event{}, d.logs...)
	authCode, authStatus, authPending := d.authCode, d.authStatus, d.authPending
	d.mu.Unlock()
	active := s.Active != nil
	activeDry := false
 activeID := ""
 activeStartedAt := int64(0)
 if active {
  activeDry = s.Active.DryRun
  activeID = s.Active.ID
  activeStartedAt = s.Active.StartedAt
 }
	phase, secondsLeft := predictionPhase(s.Active, config.predictionWindowMinutes(), time.Now())
	return map[string]any{"running": running, "realMode": config.RealMode, "connected": connected, "channel": channel,
		"twitchPhase": connectionPhase(hasToken, validated, authPending),
		"monitorPhase": monitorPhase(running, active),
		"predictionPhase": phase, "predictionWindowSecondsLeft": secondsLeft,
		"playerName": config.PlayerName, "playerToonHandle": config.PlayerToonHandle,
		"lastResult": s.LastResult, "active": active, "activeDryRun": activeDry,
  "activePredictionId": activeID, "activePredictionStartedAt": activeStartedAt,
		"authCode": authCode, "authStatus": authStatus, "authPending": authPending, "logs": logs}
}
func (d *dashboard) config() Config {
	d.engine.mu.Lock()
	defer d.engine.mu.Unlock()
	return d.engine.cfg
}
func (d *dashboard) currentReplays() []replaySummary {
	d.mu.Lock()
	if time.Since(d.replaysAt) < 25*time.Second && d.replays != nil {
		res := append([]replaySummary{}, d.replays...)
		d.mu.Unlock()
		return res
	}
	d.mu.Unlock()
	config := d.config()
	files := replaysAfter(config, time.Time{}, 12)
	res := make([]replaySummary, 0, len(files))
	for _, file := range files {
		s := replaySummary{File: filepath.Base(file.Path), Modified: file.Modified.Format(time.RFC3339)}
		r, err := parseReplay(file.Path)
		if err != nil {
			s.Error = err.Error()
			s.Result = "unknown"
		} else {
			s.Map = r.Map
			s.Players = playersFromReplay(r)
			p, er := findOutcome(r, config.PlayerName, config.PlayerToonHandle, config.lang())
			if er == nil {
				s.Result = p.Result
				s.Selected = p.Name
			} else {
				s.Result = "unknown"
				s.SelectionError = er.Error()
			}
		}
		res = append(res, s)
	}
	d.mu.Lock()
	d.replays = res
	d.replaysAt = time.Now()
	d.mu.Unlock()
	return res
}
func (d *dashboard) invalidateReplays() {
	d.mu.Lock()
	d.replays = nil
	d.replaysAt = time.Time{}
	d.mu.Unlock()
}
func playersFromReplay(r Replay) []playerSummary {
	out := make([]playerSummary, 0, len(r.Players))
	for _, p := range r.Players {
		out = append(out, playerSummary{Name: p.Name, ToonHandle: p.ToonHandle, Result: p.Result})
	}
	return out
}

type replayRootEntry struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Replays int    `json:"replays"`
	Custom  bool   `json:"custom"`
}

func countStormReplays(root string) int {
	count := 0
	visited := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		visited++
		if visited > 20000 {
			return filepath.SkipAll
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".stormreplay") {
			count++
		}
		return nil
	})
	return count
}

// checkReplayRoot verifies a candidate folder (or the defaults when empty)
// so the UI can confirm the path before saving.
func (d *dashboard) checkReplayRoot(custom string) (map[string]any, error) {
	custom = strings.TrimSpace(custom)
	if len(custom) > 1000 {
		return nil, errors.New(d.tr("config.too_long"))
	}
	var roots []string
	isCustom := custom != ""
	if isCustom {
		roots = []string{custom}
	} else {
		roots = replayRoots(d.config())
	}
	entries := make([]replayRootEntry, 0, len(roots))
	for _, root := range roots {
		entry := replayRootEntry{Path: root, Custom: isCustom}
		if st, err := os.Stat(root); err == nil && st.IsDir() {
			entry.Exists = true
			entry.Replays = countStormReplays(root)
		}
		entries = append(entries, entry)
	}
	return map[string]any{"custom": isCustom, "entries": entries}, nil
}
func (d *dashboard) doAuth() (map[string]any, error) {
	d.mu.Lock()
	if d.authPending {
		code := d.authCode
		u := d.verificationURI
		d.mu.Unlock()
		return map[string]any{"code": code, "verificationUri": u}, nil
	}
	d.authPending = true
	d.authCode = ""
	d.authStatus = d.tr("auth.requesting")
	d.mu.Unlock()
	config := d.config()
	if config.ClientID == "" {
		d.mu.Lock()
		d.authPending = false
		d.mu.Unlock()
		return nil, errors.New(d.tr("auth.need_client_id"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 31*time.Minute)
	code, err := startDevice(ctx, config.ClientID)
	if err != nil {
		cancel()
		d.mu.Lock()
		d.authPending = false
		d.authStatus = d.tr("auth.request_failed")
		d.mu.Unlock()
		return nil, err
	}
	uri := "https://www.twitch.tv/activate"
	if strings.HasPrefix(code.VerificationURI, "https://") {
		parsed, e := url.Parse(code.VerificationURI)
		if e == nil && (strings.EqualFold(parsed.Hostname(), "twitch.tv") || strings.EqualFold(parsed.Hostname(), "www.twitch.tv")) {
			uri = code.VerificationURI
		}
	}
	d.mu.Lock()
	d.authCode = code.UserCode
	d.authStatus = d.tr("auth.enter_code")
	d.verificationURI = uri
	d.mu.Unlock()
	d.log(d.tr("log.auth_code_received"))
	go func() {
		defer cancel()
		tok, err := pollDevice(ctx, config.ClientID, config.lang(), code, d.log)
		if err != nil {
			d.logEvent("error", "twitch_auth", d.tr("log.auth_error", err.Error()))
			d.mu.Lock()
			d.authPending = false
			d.authCode = ""
			d.authStatus = ""
			d.mu.Unlock()
			return
		}
		d.engine.mu.Lock()
		t := d.engine.twitch
		d.engine.mu.Unlock()
		t.mu.Lock()
		err = t.saveToken(tok)
		t.mu.Unlock()
		if err != nil {
			d.logEvent("error", "token_store", d.tr("log.token_protect_error", err.Error()))
			d.mu.Lock()
			d.authPending = false
			d.authCode = ""
			d.mu.Unlock()
			return
		}
		me, err := t.identity(ctx)
		if err != nil {
			d.logEvent("warning", "twitch_auth", d.tr("log.identity_failed", err.Error()))
		} else {
			d.log(d.tr("log.twitch_connected", me.Login))
		}
		d.mu.Lock()
		d.authPending = false
		d.authCode = ""
		d.authStatus = ""
		d.mu.Unlock()
	}()
	return map[string]any{"code": code.UserCode, "verificationUri": uri}, nil
}
func (d *dashboard) checkReplay() (map[string]any, error) {
	cfg := d.config()
	list := replaysAfter(cfg, time.Time{}, 1)
	if len(list) == 0 {
		return nil, errors.New(d.tr("replay.not_found"))
	}
	r, err := parseReplay(list[0].Path)
	if err != nil {
		return nil, fmt.Errorf("%s", d.tr("replay.read_error", filepath.Base(list[0].Path), err.Error()))
	}
	players := playersFromReplay(r)
	res := map[string]any{"file": filepath.Base(list[0].Path), "map": r.Map, "players": players}
	selected, err := findOutcome(r, cfg.PlayerName, cfg.PlayerToonHandle, cfg.lang())
	if err != nil {
		res["playerError"] = err.Error()
		d.logEvent("warning", "player_not_found", d.tr("log.replay_no_player", err.Error()))
	} else {
		res["selected"] = playerSummary{Name: selected.Name, ToonHandle: selected.ToonHandle, Result: selected.Result}
		d.log(d.tr("log.replay_summary", r.Map, selected.Name, selected.Result))
	}
	d.invalidateReplays()
	return res, nil
}
func readJSONRequest(r *http.Request, v any) error {
	if r.ContentLength > 16384 {
		return errors.New(tr(langEn, "http.request_too_large"))
	}
	r.Body = http.MaxBytesReader(nil, r.Body, 16384)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}
func jsonOutput(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func requestError(w http.ResponseWriter, err error) {
	jsonOutput(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}
func randomSegment() (string, error) {
	b := make([]byte, 24)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func (d *dashboard) routes(addr, secret string) http.Handler {
	prefix := "/" + secret + "/"
	web, err := fs.Sub(interfaceFiles, "web")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(web))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// URL capability and strict loopback binding prevent ordinary cross-site requests.
		if r.Host != addr {
			http.Error(w, "Unknown Host", http.StatusForbidden)
			return
		}
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; script-src 'self'; style-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		// Used to exit the helper process after the dashboard window has been closed.
		if strings.HasPrefix(r.URL.Path, prefix+"api/") {
			d.mu.Lock()
			d.lastActivity = time.Now()
			d.mu.Unlock()
		}
		sub := strings.TrimPrefix(r.URL.Path, prefix)
		if !strings.HasPrefix(sub, "api/") {
			if sub == "" {
				page, err := interfaceFiles.ReadFile("web/index.html")
				if err != nil {
					http.Error(w, "UI unavailable", 500)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-store")
				_, _ = w.Write(page)
				return
			}
			r.URL.Path = "/" + sub
			files.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("X-Companion") != "1" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if r.Method == "POST" {
			origin := r.Header.Get("Origin")
			if origin != "" && origin != "http://"+addr {
				http.Error(w, "Origin refused", http.StatusForbidden)
				return
			}
			if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
				http.Error(w, "Cross-site requests denied", http.StatusForbidden)
				return
			}
		}
		route := strings.TrimPrefix(sub, "api/")
		if r.Method == "GET" {
			switch route {
			case "bootstrap":
				jsonOutput(w, 200, map[string]any{"config": d.config(), "status": d.status(), "replays": d.currentReplays()})
				return
			case "status":
				jsonOutput(w, 200, d.status())
				return
			case "replays":
				jsonOutput(w, 200, map[string]any{"replays": d.currentReplays()})
				return
			}
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		switch route {
		case "config":
			d.mu.Lock()
			pending := d.authPending
			d.mu.Unlock()
			if pending {
				requestError(w, errors.New(d.tr("config.auth_pending")))
				return
			}
			var body struct {
				ClientID            string `json:"clientId"`
				PlayerName          string `json:"playerName"`
				PlayerToonHandle    string `json:"playerToonHandle"`
				ReplayRoot          string `json:"replayRoot"`
				RealMode            bool   `json:"realMode"`
				ConfirmedRealMode   bool   `json:"confirmedRealMode"`
				Lang                string `json:"lang"`
				PredictionWindowMin int    `json:"predictionWindowMin"`
				WizardDone          bool   `json:"wizardDone"`
				PredictionTitle     string `json:"predictionTitle"`
				PredictionWin       string `json:"predictionWin"`
				PredictionLoss      string `json:"predictionLoss"`
			}
			if err := readJSONRequest(r, &body); err != nil {
				requestError(w, err)
				return
			}
			if body.RealMode && !body.ConfirmedRealMode {
				requestError(w, errors.New(d.tr("config.confirm_real")))
				return
			}
			if body.Lang != "" && normalizeLang(body.Lang) != body.Lang {
				requestError(w, errors.New(d.tr("config.lang_invalid")))
				return
			}
			if body.PredictionWindowMin < 0 || body.PredictionWindowMin > maxPredictionWindow {
				requestError(w, errors.New(d.tr("config.window_invalid")))
				return
			}
			// Twitch API limits: prediction title 45 chars, outcome titles 25 chars.
			resolved := Config{Lang: normalizeLang(body.Lang), PredictionTitle: body.PredictionTitle, PredictionWin: body.PredictionWin, PredictionLoss: body.PredictionLoss}
			_, winTitle, lossTitle := resolved.predictionTexts()
			if utf8.RuneCountInString(strings.TrimSpace(body.PredictionTitle)) > 45 ||
				utf8.RuneCountInString(strings.TrimSpace(body.PredictionWin)) > 25 ||
				utf8.RuneCountInString(strings.TrimSpace(body.PredictionLoss)) > 25 {
				requestError(w, errors.New(d.tr("config.prediction_len")))
				return
			}
			if strings.EqualFold(winTitle, lossTitle) {
				requestError(w, errors.New(d.tr("config.prediction_same")))
				return
			}
			if len(body.ClientID) > 160 || len(body.PlayerName) > 160 || len(body.PlayerToonHandle) > 240 || len(body.ReplayRoot) > 1000 {
				requestError(w, errors.New(d.tr("config.too_long")))
				return
			}
			cfg := Config{ClientID: strings.TrimSpace(body.ClientID), PlayerName: strings.TrimSpace(body.PlayerName), PlayerToonHandle: strings.TrimSpace(body.PlayerToonHandle), ReplayRoot: strings.TrimSpace(body.ReplayRoot), RealMode: body.RealMode, Lang: normalizeLang(body.Lang), PredictionWindowMin: body.PredictionWindowMin, WizardDone: body.WizardDone, PredictionTitle: strings.TrimSpace(body.PredictionTitle), PredictionWin: strings.TrimSpace(body.PredictionWin), PredictionLoss: strings.TrimSpace(body.PredictionLoss)}
			if err := d.engine.updateConfig(cfg); err != nil {
				requestError(w, err)
				return
			}
			d.invalidateReplays()
			d.log(d.tr("log.config_saved"))
			jsonOutput(w, 200, map[string]any{"config": cfg})
			return
		case "auth":
			v, err := d.doAuth()
			if err != nil {
				requestError(w, err)
				return
			}
			jsonOutput(w, 200, v)
			return
		case "replay/check":
			v, err := d.checkReplay()
			if err != nil {
				d.logEvent("error", "replay_check", d.tr("log.replay_check_error", err.Error()))
				requestError(w, err)
				return
			}
			jsonOutput(w, 200, v)
			return
		case "replayroot/check":
			var body struct {
				Path string `json:"path"`
			}
			if err := readJSONRequest(r, &body); err != nil {
				requestError(w, err)
				return
			}
			v, err := d.checkReplayRoot(body.Path)
			if err != nil {
				requestError(w, err)
				return
			}
			jsonOutput(w, 200, v)
			return
		case "prediction/manual":
   var body struct {
    Outcome string `json:"outcome"`
    ID string `json:"id"`
    StartedAt int64 `json:"startedAt"`
   }
   if err := readJSONRequest(r, &body); err != nil {
    requestError(w, err)
    return
   }
   ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
   defer cancel()
   if err := d.engine.manualOutcome(ctx, body.Outcome, body.ID, body.StartedAt); err != nil {
    requestError(w, err)
    return
   }
   jsonOutput(w, 200, map[string]any{"ok": true})
   return
		case "automation/start":
			if err := d.engine.start(); err != nil {
				requestError(w, err)
				return
			}
			d.log(d.tr("log.automation_started"))
			jsonOutput(w, 200, map[string]bool{"ok": true})
			return
		case "automation/stop":
			d.engine.stopWatch()
			d.log(d.tr("log.automation_stopping"))
			jsonOutput(w, 200, map[string]bool{"ok": true})
			return
		case "quit":
			d.engine.stopWatch()
			d.shutdownOnce.Do(func() { close(d.shutdown) })
			jsonOutput(w, 200, map[string]bool{"ok": true})
			return
		default:
			http.NotFound(w, r)
			return
		}
	})
}
func (d *dashboard) launch() (string, *http.Server, error) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	addr := ln.Addr().String()
	key, err := randomSegment()
	if err != nil {
		ln.Close()
		return "", nil, err
	}
	srv := &http.Server{Handler: d.routes(addr, key), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() {
		e := srv.Serve(ln)
		if e != nil && !errors.Is(e, http.ErrServerClosed) {
			d.logEvent("error", "local_server", tr(d.config().Lang, "log.http_server", e.Error()))
		}
	}()
	return "http://" + addr + "/" + key + "/", srv, nil
}
func (d *dashboard) stop(srv *http.Server) {
	d.engine.stopWatch()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
func (d *dashboard) runDesktop(open func(string) error) error {
	url, srv, err := d.launch()
	if err != nil {
		return err
	}
	defer d.stop(srv)
	if err = open(url); err != nil {
		return err
	}
	// A minimized window continues polling; a closed window stops for several minutes.
	// Explicit Exit terminates immediately.
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-d.shutdown:
				return
			case <-ticker.C:
				d.mu.Lock()
				last := d.lastActivity
				d.mu.Unlock()
				if !last.IsZero() && time.Since(last) > 4*time.Minute {
					d.log(d.tr("log.shutdown_idle"))
					d.shutdownOnce.Do(func() { close(d.shutdown) })
					return
				}
			}
		}
	}()
	<-d.shutdown
	return nil
}
