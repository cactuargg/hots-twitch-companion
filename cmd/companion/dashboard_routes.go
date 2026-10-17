package main

import (
 dashview "hots-twitch-companion/internal/dashboard"
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// Local dashboard HTTP transport, request validation and route dispatch.
// The dashboard's runtime state, Twitch auth orchestration and replay display
// stay in service.go. No request handling behavior changes in this extraction.

func readJSONRequest(r *http.Request, v any) error {
 return dashview.ReadJSONRequest(r, v, tr(langEn, "http.request_too_large"))
}
func jsonOutput(w http.ResponseWriter, code int, v any) { dashview.JSONOutput(w, code, v) }
func requestError(w http.ResponseWriter, err error) { dashview.RequestError(w, err) }
func randomSegment() (string, error) { return dashview.RandomSegment() }
func (d *dashboard) routes(addr, secret string) http.Handler {
 return dashview.NewHTTPHandler(addr, secret, func(w http.ResponseWriter, r *http.Request, route string) {
  d.handleAPIRoute(w, r, route)
 }, func() {
  d.mu.Lock()
  d.lastActivity = time.Now()
  d.mu.Unlock()
 })
}

func (d *dashboard) handleAPIRoute(w http.ResponseWriter, r *http.Request, route string) {
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
		d.postMu.Lock()
		defer d.postMu.Unlock()
		select {
		case <-d.shutdown:
			jsonOutput(w, 503, map[string]string{"error": "Application is closing"})
			return
		default:
		}
		switch route {
		case "profile/reset":
			var body struct { Confirmed bool `json:"confirmed"` }
			if err:=readJSONRequest(r,&body);err!=nil {requestError(w,err);return}
			if !body.Confirmed {requestError(w,errors.New(d.tr("reset.confirm_required")));return}
			if err:=d.requestProfileReset();err!=nil {requestError(w,err);return}
			jsonOutput(w,200,map[string]bool{"ok":true})
			return
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
			cfg := Config{ClientID: strings.TrimSpace(body.ClientID), PlayerName: strings.TrimSpace(body.PlayerName), PlayerToonHandle: strings.TrimSpace(body.PlayerToonHandle), ReplayRoot: strings.TrimSpace(body.ReplayRoot), RealMode: true, Lang: normalizeLang(body.Lang), PredictionWindowMin: body.PredictionWindowMin, WizardDone: body.WizardDone, PredictionTitle: strings.TrimSpace(body.PredictionTitle), PredictionWin: strings.TrimSpace(body.PredictionWin), PredictionLoss: strings.TrimSpace(body.PredictionLoss)}
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
		case "prediction/acknowledge-unconfirmed":
			var body struct { Confirmed bool `json:"confirmed"` }
			if err := readJSONRequest(r, &body); err != nil { requestError(w, err); return }
			if !body.Confirmed { requestError(w, errors.New("Explicit confirmation required")); return }
			ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
			defer cancel()
			if err := d.engine.acknowledgeUnconfirmedCreation(ctx); err != nil { requestError(w, err); return }
			jsonOutput(w, 200, map[string]bool{"ok": true})
			return
		case "prediction/recheck":
   var body struct {
    ID string `json:"id"`
    StartedAt int64 `json:"startedAt"`
   }
   if err := readJSONRequest(r, &body); err != nil {
    requestError(w, err)
    return
   }
   ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
   defer cancel()
   scan, err := d.engine.recheckReplay(ctx, body.ID, body.StartedAt)
   if err != nil { requestError(w, err); return }
   jsonOutput(w, 200, map[string]any{"ok":true, "scan":scan})
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
			var body struct {
				ConfirmedRealMode bool `json:"confirmedRealMode"`
			}
			if err := readJSONRequest(r, &body); err != nil {
				requestError(w, err)
				return
			}
			if !body.ConfirmedRealMode {
				requestError(w, errors.New(d.tr("config.confirm_real")))
				return
			}
			d.engine.mu.Lock()
			tw := d.engine.twitch
			live := d.engine.cfg.RealMode
			d.engine.mu.Unlock()
			tw.mu.Lock()
			connected := tw.token.AccessToken != "" && tw.login != ""
			tw.mu.Unlock()
			if !live || !connected {
				requestError(w, errors.New(d.tr("automation.need_twitch")))
				return
			}
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
		case "window/hide":
			if d.hideToTray == nil {
				jsonOutput(w, 400, map[string]string{"error": "Desktop shell unavailable"})
				return
			}
			if err := d.hideToTray(); err != nil {
				jsonOutput(w, 503, map[string]string{"error": err.Error()})
				return
			}
			jsonOutput(w, 200, map[string]bool{"ok": true})
			return
		default:
			http.NotFound(w, r)
			return
		}
}
