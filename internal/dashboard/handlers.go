package dashboard

import (
	"context"
	"errors"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/i18n"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

func (d *Dashboard) handleAPIRoute(w http.ResponseWriter, r *http.Request, route string) {
	if route == "sessions" && r.Method == "GET" {
		d.overlayAPI(w, r, route)
		return
	}
	if r.Method == "GET" {
		switch route {
		case "statistics":
			d.statisticsAPI(w, r)
			return
		case "bootstrap":
			JSONOutput(w, 200, map[string]any{"config": d.Config(), "status": d.Status(), "replays": d.CurrentReplays()})
			return
		case "status":
			JSONOutput(w, 200, d.Status())
			return
		case "replays":
			JSONOutput(w, 200, map[string]any{"replays": d.CurrentReplays()})
			return
		}
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	d.postMu.Lock()
	defer d.postMu.Unlock()
	select {
	case <-d.shutdown:
		JSONOutput(w, 503, map[string]string{"error": "Application is closing"})
		return
	default:
	}
	if strings.HasPrefix(route, "sessions/") {
		d.overlayAPI(w, r, route)
		return
	}
	switch route {
	case "updates/startup-ready":
		var body struct{}
		if err := ReadJSONRequest(r, &body, d.Text("http.request_too_large")); err != nil {
			RequestError(w, err)
			return
		}
		if err := d.NotifyUIReady(); err != nil {
			RequestError(w, err)
			return
		}
		JSONOutput(w, 200, map[string]bool{"ok": true})
		return
	case "updates/preferences":
		var body struct {
			Enabled *bool `json:"enabled"`
		}
		if err := ReadJSONRequest(r, &body, d.Text("http.request_too_large")); err != nil {
			RequestError(w, err)
			return
		}
		if body.Enabled == nil {
			RequestError(w, errors.New(d.Text("update.preference_required")))
			return
		}
		cfg := d.Config()
		cfg.DisableUpdateChecks = !*body.Enabled
		if err := d.engine.UpdateConfig(cfg); err != nil {
			RequestError(w, err)
			return
		}
		d.updates.SetEnabled(*body.Enabled)
		JSONOutput(w, 200, map[string]any{"config": cfg, "updates": d.updates.Snapshot()})
		return
	case "updates/check":
		if err := d.updates.RequestCheck(); err != nil {
			RequestError(w, errors.New(d.Text("update.wait")))
			return
		}
		JSONOutput(w, 200, d.updates.Snapshot())
		return
	case "updates/download":
		if err := d.updates.RequestDownload(); err != nil {
			RequestError(w, errors.New(d.Text("update.wait")))
			return
		}
		JSONOutput(w, 200, d.updates.Snapshot())
		return
	case "updates/install":
		var body struct {
			Confirmed bool `json:"confirmed"`
		}
		if err := ReadJSONRequest(r, &body, d.Text("http.request_too_large")); err != nil {
			RequestError(w, err)
			return
		}
		if !body.Confirmed {
			RequestError(w, errors.New(d.Text("update.confirm_required")))
			return
		}
		if err := d.installPreparedUpdate(); err != nil {
			RequestError(w, err)
			return
		}
		JSONOutput(w, 200, map[string]bool{"ok": true})
		return
	case "profile/reset":
		var body struct {
			Confirmed bool `json:"confirmed"`
		}
		if err := ReadJSONRequest(r, &body, i18n.Text(i18n.En, "http.request_too_large")); err != nil {
			RequestError(w, err)
			return
		}
		if !body.Confirmed {
			RequestError(w, errors.New(d.Text("reset.confirm_required")))
			return
		}
		if err := d.RequestProfileReset(); err != nil {
			RequestError(w, err)
			return
		}
		JSONOutput(w, 200, map[string]bool{"ok": true})
		return
	case "config":
		d.mu.Lock()
		pending := d.authPending
		d.mu.Unlock()
		if pending {
			RequestError(w, errors.New(d.Text("config.auth_pending")))
			return
		}
		var body struct {
			GameProfiles        []config.GameProfile `json:"gameProfiles"`
			ActiveGameProfile   string               `json:"activeGameProfile"`
			ClientID            string               `json:"clientId"`
			PlayerName          string               `json:"playerName"`
			PlayerToonHandle    string               `json:"playerToonHandle"`
			ReplayRoot          string               `json:"replayRoot"`
			RealMode            bool                 `json:"realMode"`
			ConfirmedRealMode   bool                 `json:"confirmedRealMode"`
			Lang                string               `json:"lang"`
			PredictionWindowMin int                  `json:"predictionWindowMin"`
			WizardDone          bool                 `json:"wizardDone"`
			PredictionTitle     string               `json:"predictionTitle"`
			PredictionWin       string               `json:"predictionWin"`
			PredictionLoss      string               `json:"predictionLoss"`
			DisableUpdateChecks bool                 `json:"disableUpdateChecks"`
		}
		if err := ReadJSONRequest(r, &body, i18n.Text(i18n.En, "http.request_too_large")); err != nil {
			RequestError(w, err)
			return
		}
		if body.Lang != "" && i18n.Normalize(body.Lang) != body.Lang {
			RequestError(w, errors.New(d.Text("config.lang_invalid")))
			return
		}
		if body.PredictionWindowMin < 0 || body.PredictionWindowMin > config.MaxPredictionWindow {
			RequestError(w, errors.New(d.Text("config.window_invalid")))
			return
		}
		// Twitch API limits: prediction title 45 chars, outcome titles 25 chars.
		resolved := config.Data{Lang: i18n.Normalize(body.Lang), PredictionTitle: body.PredictionTitle, PredictionWin: body.PredictionWin, PredictionLoss: body.PredictionLoss}
		_, winTitle, lossTitle := resolved.PredictionTexts()
		if utf8.RuneCountInString(strings.TrimSpace(body.PredictionTitle)) > 45 ||
			utf8.RuneCountInString(strings.TrimSpace(body.PredictionWin)) > 25 ||
			utf8.RuneCountInString(strings.TrimSpace(body.PredictionLoss)) > 25 {
			RequestError(w, errors.New(d.Text("config.prediction_len")))
			return
		}
		if strings.EqualFold(winTitle, lossTitle) {
			RequestError(w, errors.New(d.Text("config.prediction_same")))
			return
		}
		if len(body.ClientID) > 160 || len(body.PlayerName) > 160 || len(body.PlayerToonHandle) > 240 || len(body.ReplayRoot) > 1000 {
			RequestError(w, errors.New(d.Text("config.too_long")))
			return
		}
		cfg := config.Data{ClientID: strings.TrimSpace(body.ClientID), PlayerName: strings.TrimSpace(body.PlayerName), PlayerToonHandle: strings.TrimSpace(body.PlayerToonHandle), ReplayRoot: strings.TrimSpace(body.ReplayRoot), RealMode: true, Lang: i18n.Normalize(body.Lang), PredictionWindowMin: body.PredictionWindowMin, WizardDone: body.WizardDone, PredictionTitle: strings.TrimSpace(body.PredictionTitle), PredictionWin: strings.TrimSpace(body.PredictionWin), PredictionLoss: strings.TrimSpace(body.PredictionLoss)}
		// The independent preference endpoint is authoritative. Old tabs and
		// wizard/language snapshots must not silently re-enable network checks.
		cfg.DisableUpdateChecks = d.Config().DisableUpdateChecks
		cfg.GameProfiles = body.GameProfiles
		cfg.ActiveGameProfile = body.ActiveGameProfile
		if err := d.engine.UpdateConfig(cfg); err != nil {
			RequestError(w, err)
			return
		}
		d.invalidateReplays()
		d.log(d.Text("log.config_saved"))
		JSONOutput(w, 200, map[string]any{"config": d.Config()})
		return
	case "auth":
		v, err := d.doAuth()
		if err != nil {
			RequestError(w, err)
			return
		}
		JSONOutput(w, 200, v)
		return
	case "replay/check":
		v, err := d.checkReplay()
		if err != nil {
			d.LogEvent("error", "replay_check", d.Text("log.replay_check_error", err.Error()))
			RequestError(w, err)
			return
		}
		JSONOutput(w, 200, v)
		return
	case "replayroot/check":
		var body struct {
			Path string `json:"path"`
		}
		if err := ReadJSONRequest(r, &body, i18n.Text(i18n.En, "http.request_too_large")); err != nil {
			RequestError(w, err)
			return
		}
		v, err := d.CheckReplayRoot(body.Path)
		if err != nil {
			RequestError(w, err)
			return
		}
		JSONOutput(w, 200, v)
		return
	case "prediction/acknowledge-unconfirmed":
		var body struct {
			Confirmed bool `json:"confirmed"`
		}
		if err := ReadJSONRequest(r, &body, i18n.Text(i18n.En, "http.request_too_large")); err != nil {
			RequestError(w, err)
			return
		}
		if !body.Confirmed {
			RequestError(w, errors.New("Explicit confirmation required"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
		defer cancel()
		if err := d.engine.AcknowledgeUnconfirmedCreation(ctx); err != nil {
			RequestError(w, err)
			return
		}
		JSONOutput(w, 200, map[string]bool{"ok": true})
		return
	case "prediction/recheck":
		var body struct {
			ID        string `json:"id"`
			StartedAt int64  `json:"startedAt"`
		}
		if err := ReadJSONRequest(r, &body, i18n.Text(i18n.En, "http.request_too_large")); err != nil {
			RequestError(w, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
		defer cancel()
		scan, err := d.engine.RecheckReplay(ctx, body.ID, body.StartedAt)
		if err != nil {
			RequestError(w, err)
			return
		}
		JSONOutput(w, 200, map[string]any{"ok": true, "scan": scan})
		return
	case "prediction/manual":
		var body struct {
			Outcome   string `json:"outcome"`
			ID        string `json:"id"`
			StartedAt int64  `json:"startedAt"`
		}
		if err := ReadJSONRequest(r, &body, i18n.Text(i18n.En, "http.request_too_large")); err != nil {
			RequestError(w, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
		defer cancel()
		if err := d.engine.ManualOutcome(ctx, body.Outcome, body.ID, body.StartedAt); err != nil {
			RequestError(w, err)
			return
		}
		JSONOutput(w, 200, map[string]any{"ok": true})
		return
	case "automation/start":
		if d.updateStartupPending() {
			RequestError(w, errors.New(d.Text("update.startup_wait")))
			return
		}
		var body struct {
			ConfirmedRealMode bool `json:"confirmedRealMode"`
		}
		if err := ReadJSONRequest(r, &body, i18n.Text(i18n.En, "http.request_too_large")); err != nil {
			RequestError(w, err)
			return
		}
		if !body.ConfirmedRealMode {
			RequestError(w, errors.New(d.Text("config.confirm_real")))
			return
		}
		live := d.engine.Config().RealMode
		connection := d.engine.Connection()
		connected := connection.HasToken && connection.Login != ""
		if !live || !connected {
			RequestError(w, errors.New(d.Text("automation.need_twitch")))
			return
		}
		if err := d.engine.Start(); err != nil {
			RequestError(w, err)
			return
		}
		d.log(d.Text("log.automation_started"))
		JSONOutput(w, 200, map[string]bool{"ok": true})
		return
	case "automation/stop":
		d.engine.StopWatch()
		d.log(d.Text("log.automation_stopping"))
		JSONOutput(w, 200, map[string]bool{"ok": true})
		return
	case "quit":
		d.Close()
		JSONOutput(w, 200, map[string]bool{"ok": true})
		return
	case "window/hide":
		if d.hideToTray == nil {
			JSONOutput(w, 400, map[string]string{"error": "Desktop shell unavailable"})
			return
		}
		if err := d.hideToTray(); err != nil {
			JSONOutput(w, 503, map[string]string{"error": err.Error()})
			return
		}
		JSONOutput(w, 200, map[string]bool{"ok": true})
		return
	default:
		http.NotFound(w, r)
		return
	}
}
