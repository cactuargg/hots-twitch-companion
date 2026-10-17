package main

import (
	"context"
	"errors"
	"fmt"
	dashview "hots-twitch-companion/internal/dashboard"
	"hots-twitch-companion/internal/hots/discovery"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Embedded resources now live in the reusable dashboard package.
var interfaceFiles = dashview.WebFiles

type event struct {
	Time    string `json:"time"`
	Message string `json:"message"`
	Level   string `json:"level"`
	Code    string `json:"code,omitempty"`
}
type replaySummary struct {
	File               string          `json:"file"`
	Map                string          `json:"map"`
	Result             string          `json:"result"`
	Modified           string          `json:"modified"`
	Players            []playerSummary `json:"players,omitempty"`
	Selected           string          `json:"selected,omitempty"`
	SelectedToonHandle string          `json:"selectedToonHandle,omitempty"`
	Error              string          `json:"error,omitempty"`
	SelectionError     string          `json:"selectionError,omitempty"`
}
type playerSummary struct {
	Name       string `json:"name"`
	ToonHandle string `json:"toonHandle"`
	Result     string `json:"result"`
}

type dashboard struct {
	postMu                                sync.Mutex
	hideToTray                            func() error
	mu                                    sync.Mutex
	engine                                *Engine
	p                                     paths
	logs                                  []event
	authCode, authStatus, verificationURI string
	authPending                           bool
	replays                               []replaySummary
	replaysAt                             time.Time
	updateVersion, updateURL              string
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
	// Background update check against public GitHub release metadata.
	go d.watchUpdates()
	// Verify restored tokens in the background; do not block the dashboard on network.
	twitch := e.twitch
	if twitch.Connection().HasToken {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			if who, err := twitch.Identity(ctx); err == nil {
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
	return dashview.ConnectionPhase(hasToken, validated, pending)
}
func monitorPhase(running, tracking bool) string {
	return dashview.MonitorPhase(running, tracking)
}
func predictionPhase(p *Prediction, windowMinutes int, now time.Time) (string, int) {
	return dashview.PredictionPhase(p, windowMinutes, now)
}

func (d *dashboard) status() map[string]any {
	e := d.engine
	e.mu.Lock()
	running, config, s, scan := e.running, e.cfg, e.state, e.scan
	tw := e.twitch
	e.mu.Unlock()
	connection := tw.Connection()
	hasToken, channel := connection.HasToken, connection.Login
	validated := channel != ""
	connected := hasToken && validated
	d.mu.Lock()
	logs := append([]event{}, d.logs...)
	authCode, authStatus, authPending := d.authCode, d.authStatus, d.authPending
	updateVersion, updateURL := d.updateVersion, d.updateURL
	d.mu.Unlock()
	active := s.Active != nil
	activeDry := false
	activeID := ""
	activeStartedAt := int64(0)
	pendingOutcome := ""
	if active {
		activeDry = s.Active.DryRun
		activeID = s.Active.ID
		activeStartedAt = s.Active.StartedAt
		if s.Active.Settlement != nil {
			pendingOutcome = s.Active.Settlement.Outcome
		}
	}
	phase, secondsLeft := predictionPhase(s.Active, config.predictionWindowMinutes(), time.Now())
	return map[string]any{"version": appVersion, "running": running, "realMode": config.RealMode, "connected": connected, "channel": channel,
		"twitchPhase":     connectionPhase(hasToken, validated, authPending),
		"monitorPhase":    monitorPhase(running, active),
		"predictionPhase": phase, "predictionWindowSecondsLeft": secondsLeft,
		"playerName": config.PlayerName, "playerToonHandle": config.PlayerToonHandle,
		"lastResult": s.LastResult, "active": active, "activeDryRun": activeDry,
		"activePredictionId": activeID, "activePredictionStartedAt": activeStartedAt,
		"pendingOutcome":      pendingOutcome,
		"creationUnconfirmed": s.UnconfirmedCreation != nil && s.Active == nil,
		"replayScan":          scan, "hasTwitchApp": config.twitchClientID() != "",
		"authCode": authCode, "authStatus": authStatus, "authPending": authPending, "logs": logs,
		"updateVersion": updateVersion, "updateURL": updateURL}
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
				s.SelectedToonHandle = p.ToonHandle
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

func countStormReplays(root string) int { return discovery.CountStormReplays(root) }

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
		roots = defaultReplayRoots()
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
	if config.twitchClientID() == "" {
		d.mu.Lock()
		d.authPending = false
		d.mu.Unlock()
		return nil, errors.New(d.tr("auth.need_client_id"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 31*time.Minute)
	code, err := startDevice(ctx, config.twitchClientID())
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
		tok, err := pollDevice(ctx, config.twitchClientID(), config.lang(), code, d.log)
		if err != nil {
			d.logEvent("error", "twitch_auth", d.tr("log.auth_error", err.Error()))
			d.mu.Lock()
			d.authPending = false
			d.authCode = ""
			d.authStatus = ""
			d.mu.Unlock()
			return
		}
		me, err := d.engine.installTwitchToken(ctx, config.twitchClientID(), tok)
		if err != nil {
			d.logEvent("error", "twitch_auth", d.tr("log.auth_error", err.Error()))
			d.mu.Lock()
			d.authPending = false
			d.authCode = ""
			d.authStatus = ""
			d.mu.Unlock()
			return
		}
		d.log(d.tr("log.twitch_connected", me.Login))
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
	// Start the idle clock even if the browser never reaches the API.
	// Otherwise a failed browser launch leaves the companion process alive
	// indefinitely because lastActivity stays zero.
	d.mu.Lock()
	if d.lastActivity.IsZero() {
		d.lastActivity = time.Now()
	}
	d.mu.Unlock()
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
