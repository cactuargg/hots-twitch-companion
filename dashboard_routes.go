package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// Local dashboard HTTP transport, request validation and route dispatch.
// The dashboard's runtime state, Twitch auth orchestration and replay display
// stay in service.go. No request handling behavior changes in this extraction.

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
	// A complete HTTP body must contain exactly one JSON value. Without
	// the second decode, a valid config followed by another object is
	// silently accepted even though the request is malformed.
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values in request body")
		}
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
	})
}
