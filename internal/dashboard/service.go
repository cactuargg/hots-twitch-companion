package dashboard

import (
	"context"
	"errors"
	"fmt"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/hots/discovery"
	"hots-twitch-companion/internal/hots/replay"
	"hots-twitch-companion/internal/i18n"
	"hots-twitch-companion/internal/update"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Event struct {
	Time    string `json:"time"`
	Message string `json:"message"`
	Level   string `json:"level"`
	Code    string `json:"code,omitempty"`
}
type ReplaySummary struct {
	File               string          `json:"file"`
	Map                string          `json:"map"`
	Result             string          `json:"result"`
	Modified           string          `json:"modified"`
	Players            []PlayerSummary `json:"players,omitempty"`
	Selected           string          `json:"selected,omitempty"`
	SelectedToonHandle string          `json:"selectedToonHandle,omitempty"`
	Error              string          `json:"error,omitempty"`
	SelectionError     string          `json:"selectionError,omitempty"`
}
type PlayerSummary struct {
	Name       string `json:"name"`
	ToonHandle string `json:"toonHandle"`
	Result     string `json:"result"`
}

type Dashboard struct {
	version                               string
	postMu                                sync.Mutex
	hideToTray                            func() error
	mu                                    sync.Mutex
	engine                                Application
	p                                     config.Paths
	logs                                  []Event
	authCode, authStatus, verificationURI string
	authPending                           bool
	replays                               []ReplaySummary
	replaysAt                             time.Time
	shutdown                              chan struct{}
	shutdownOnce                          sync.Once
	lastActivity                          time.Time
	backgroundOnce                        sync.Once
	backgroundCancel                      context.CancelFunc
	updates                               *update.Manager
	installUpdate                         func(update.Prepared) error
	uiReady                               func() error
	uiReadyConfirmed                      bool
}

func New(engine Application, p config.Paths, version string) *Dashboard {
	return &Dashboard{engine: engine, p: p, version: version, shutdown: make(chan struct{}), updates: update.NewManager(update.NewClient(), version, "", "", !engine.Config().DisableUpdateChecks)}
}

// StartBackground is explicit so tests never contact public services.
func (d *Dashboard) StartBackground() {
	d.backgroundOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		d.mu.Lock()
		select {
		case <-d.shutdown:
			d.mu.Unlock()
			cancel()
			return
		default:
		}
		d.backgroundCancel = cancel
		d.mu.Unlock()
		go d.watchUpdates()
		go d.watchConnection(ctx, time.Minute)
		c := d.engine.Config()
		d.log(i18n.Text(c.Language(), "log.app_started"))
		d.log(i18n.Text(c.Language(), "log.read_only"))
	})
}

// A failed startup check must recover when connectivity returns, without OAuth
// reauthorization. This loop only validates credentials; it never starts wagers.
func (d *Dashboard) watchConnection(ctx context.Context, interval time.Duration) {
	lastError := ""
	check := func() {
		connection := d.engine.Connection()
		if !connection.HasToken {
			return
		}
		checkCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		defer cancel()
		who, err := d.engine.Identity(checkCtx)
		if ctx.Err() != nil {
			return
		}
		if err == nil && (connection.Login == "" || connection.Login != who.Login) {
			d.log(d.Text("log.twitch_connected", who.Login))
		} else if err != nil && err.Error() != lastError {
			d.LogEvent("warning", "twitch_auth", d.Text("log.token_check", err.Error()))
		}
		if err != nil {
			lastError = err.Error()
		} else {
			lastError = ""
		}
	}
	check()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}
func (d *Dashboard) Text(key string, args ...any) string {
	return i18n.Text(d.Config().Lang, key, args...)
}
func (d *Dashboard) log(s string) {
	d.LogEvent("info", "", s)
}

func (d *Dashboard) LogEvent(level, code, message string) {
	if level != "warning" && level != "error" {
		level = "info"
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logs = append(d.logs, Event{Time: time.Now().Format("15:04:05"), Level: level, Code: code, Message: message})
	if len(d.logs) > 150 {
		d.logs = d.logs[len(d.logs)-150:]
	}
}

// User-facing phases derived from persisted engine state. These describe what we
// know locally, not a live verification of Twitch's current Prediction status.
func (d *Dashboard) Status() map[string]any {
	running, config, s, scan := d.engine.Snapshot()
	connection := d.engine.Connection()
	hasToken, channel := connection.HasToken, connection.Login
	validated := channel != ""
	connected := hasToken && validated
	d.mu.Lock()
	logs := append([]Event{}, d.logs...)
	authCode, authStatus, authPending := d.authCode, d.authStatus, d.authPending
	updateStartupPending := d.uiReady != nil && !d.uiReadyConfirmed
	d.mu.Unlock()
	updateState := d.updates.Snapshot()
	updateVersion, updateURL := "", ""
	if updateState.Enabled && updateState.Version != "" {
		updateVersion, updateURL = updateState.Version, updateState.URL
	}
	if !updateState.Enabled {
		updateVersion, updateURL = "", ""
	}
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
	phase, secondsLeft := PredictionPhase(s.Active, config.PredictionWindowMinutes(), time.Now())
	return map[string]any{"version": d.version, "running": running, "realMode": config.RealMode, "connected": connected, "channel": channel,
		"twitchPhase":     ConnectionPhase(hasToken, validated, authPending),
		"monitorPhase":    MonitorPhase(running, active),
		"predictionPhase": phase, "predictionWindowSecondsLeft": secondsLeft,
		"playerName": config.PlayerName, "playerToonHandle": config.PlayerToonHandle,
		"lastResult": s.LastResult, "active": active, "activeDryRun": activeDry,
		"activePredictionId": activeID, "activePredictionStartedAt": activeStartedAt,
		"pendingOutcome":      pendingOutcome,
		"creationUnconfirmed": s.UnconfirmedCreation != nil && s.Active == nil,
		"replayScan":          scan, "hasTwitchApp": d.engine.ClientID() != "",
		"authCode": authCode, "authStatus": authStatus, "authPending": authPending, "logs": logs,
		"updateVersion": updateVersion, "updateURL": updateURL, "updates": updateState, "updateStartupPending": updateStartupPending}
}
func (d *Dashboard) Config() config.Data { return d.engine.Config() }
func (d *Dashboard) CurrentReplays() []ReplaySummary {
	d.mu.Lock()
	if time.Since(d.replaysAt) < 25*time.Second && d.replays != nil {
		res := append([]ReplaySummary{}, d.replays...)
		d.mu.Unlock()
		return res
	}
	d.mu.Unlock()
	config := d.Config()
	files := discovery.ReplaysAfter(config.ReplayRoot, time.Time{}, 12)
	res := make([]ReplaySummary, 0, len(files))
	for _, file := range files {
		s := ReplaySummary{File: filepath.Base(file.Path), Modified: file.Modified.Format(time.RFC3339)}
		r, err := replay.Parse(file.Path)
		if err != nil {
			s.Error = err.Error()
			s.Result = "unknown"
		} else {
			s.Map = r.Map
			s.Players = playersFromReplay(r)
			p, er := replay.FindOutcome(r, config.PlayerName, config.PlayerToonHandle, config.Language())
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
func (d *Dashboard) invalidateReplays() {
	d.mu.Lock()
	d.replays = nil
	d.replaysAt = time.Time{}
	d.mu.Unlock()
}
func playersFromReplay(r replay.Replay) []PlayerSummary {
	out := make([]PlayerSummary, 0, len(r.Players))
	for _, p := range r.Players {
		out = append(out, PlayerSummary{Name: p.Name, ToonHandle: p.ToonHandle, Result: p.Result})
	}
	return out
}

type ReplayRootEntry struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Replays int    `json:"replays"`
	Custom  bool   `json:"custom"`
}

// checkReplayRoot verifies a candidate folder (or the defaults when empty)
// so the UI can confirm the path before saving.
func (d *Dashboard) CheckReplayRoot(custom string) (map[string]any, error) {
	custom = strings.TrimSpace(custom)
	if len(custom) > 1000 {
		return nil, errors.New(d.Text("config.too_long"))
	}
	var roots []string
	isCustom := custom != ""
	if isCustom {
		roots = []string{custom}
	} else {
		roots = discovery.DefaultReplayRoots()
	}
	entries := make([]ReplayRootEntry, 0, len(roots))
	for _, root := range roots {
		entry := ReplayRootEntry{Path: root, Custom: isCustom}
		if st, err := os.Stat(root); err == nil && st.IsDir() {
			entry.Exists = true
			entry.Replays = discovery.CountStormReplays(root)
		}
		entries = append(entries, entry)
	}
	return map[string]any{"custom": isCustom, "entries": entries}, nil
}
func (d *Dashboard) doAuth() (map[string]any, error) {
	d.mu.Lock()
	if d.authPending {
		code := d.authCode
		u := d.verificationURI
		d.mu.Unlock()
		return map[string]any{"code": code, "verificationUri": u}, nil
	}
	d.authPending = true
	d.authCode = ""
	d.authStatus = d.Text("auth.requesting")
	d.mu.Unlock()
	clientID := d.engine.ClientID()
	if clientID == "" {
		d.mu.Lock()
		d.authPending = false
		d.mu.Unlock()
		return nil, errors.New(d.Text("auth.need_client_id"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 31*time.Minute)
	code, err := d.engine.StartDevice(ctx)
	if err != nil {
		cancel()
		d.mu.Lock()
		d.authPending = false
		d.authStatus = d.Text("auth.request_failed")
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
	d.authStatus = d.Text("auth.enter_code")
	d.verificationURI = uri
	d.mu.Unlock()
	d.log(d.Text("log.auth_code_received"))
	go func() {
		defer cancel()
		tok, err := d.engine.PollDevice(ctx, code, d.log)
		if err != nil {
			d.LogEvent("error", "twitch_auth", d.Text("log.auth_error", err.Error()))
			d.mu.Lock()
			d.authPending = false
			d.authCode = ""
			d.authStatus = ""
			d.mu.Unlock()
			return
		}
		me, err := d.engine.InstallTwitchToken(ctx, clientID, tok)
		if err != nil {
			d.LogEvent("error", "twitch_auth", d.Text("log.auth_error", err.Error()))
			d.mu.Lock()
			d.authPending = false
			d.authCode = ""
			d.authStatus = ""
			d.mu.Unlock()
			return
		}
		d.log(d.Text("log.twitch_connected", me.Login))
		d.mu.Lock()
		d.authPending = false
		d.authCode = ""
		d.authStatus = ""
		d.mu.Unlock()
	}()
	return map[string]any{"code": code.UserCode, "verificationUri": uri}, nil
}
func (d *Dashboard) checkReplay() (map[string]any, error) {
	cfg := d.Config()
	list := discovery.ReplaysAfter(cfg.ReplayRoot, time.Time{}, 1)
	if len(list) == 0 {
		return nil, errors.New(d.Text("replay.not_found"))
	}
	r, err := replay.Parse(list[0].Path)
	if err != nil {
		return nil, fmt.Errorf("%s", d.Text("replay.read_error", filepath.Base(list[0].Path), err.Error()))
	}
	players := playersFromReplay(r)
	res := map[string]any{"file": filepath.Base(list[0].Path), "map": r.Map, "players": players}
	selected, err := replay.FindOutcome(r, cfg.PlayerName, cfg.PlayerToonHandle, cfg.Language())
	if err != nil {
		res["playerError"] = err.Error()
		d.LogEvent("warning", "player_not_found", d.Text("log.replay_no_player", err.Error()))
	} else {
		res["selected"] = PlayerSummary{Name: selected.Name, ToonHandle: selected.ToonHandle, Result: selected.Result}
		d.log(d.Text("log.replay_summary", r.Map, selected.Name, selected.Result))
	}
	d.invalidateReplays()
	return res, nil
}
func (d *Dashboard) launch() (string, *http.Server, error) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	addr := ln.Addr().String()
	key, err := RandomSegment()
	if err != nil {
		ln.Close()
		return "", nil, err
	}
	srv := &http.Server{Handler: d.Routes(addr, key), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() {
		e := srv.Serve(ln)
		if e != nil && !errors.Is(e, http.ErrServerClosed) {
			d.LogEvent("error", "local_server", i18n.Text(d.Config().Lang, "log.http_server", e.Error()))
		}
	}()
	return "http://" + addr + "/" + key + "/", srv, nil
}
func (d *Dashboard) stop(srv *http.Server) {
	if d.engine != nil {
		d.engine.StopWatch()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
func (d *Dashboard) RunDesktop(open func(string) error) error {
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
					d.log(d.Text("log.shutdown_idle"))
					d.Close()
					return
				}
			}
		}
	}()
	<-d.shutdown
	return nil
}
