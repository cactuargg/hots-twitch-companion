package main

import (
	"context"
	"fmt"
	"bytes"
	"compress/zlib"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoopbackDashboardAuthorization(t *testing.T) {
	dir := t.TempDir()
	p := paths{dir: dir, config: filepath.Join(dir, "config.json"), secret: filepath.Join(dir, "tokens.dpapi"), state: filepath.Join(dir, "state.json")}
	d, err := newDashboard(p, Config{})
	if err != nil {
		t.Fatal(err)
	}
	h := d.routes("127.0.0.1:12345", "secret")
	test := func(method, path, host, origin string, payload []byte, header bool) int {
		r := httptest.NewRequest(method, "http://127.0.0.1:12345"+path, bytes.NewReader(payload))
		r.Host = host
		if header {
			r.Header.Set("X-Companion", "1")
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	if got := test("GET", "/incorrect/api/status", "127.0.0.1:12345", "", nil, true); got != 404 {
		t.Fatalf("secret mismatch returned %d", got)
	}
	if got := test("GET", "/secret/api/status", "evil.local", "", nil, true); got != 403 {
		t.Fatalf("host check returned %d", got)
	}
	if got := test("GET", "/secret/api/status", "127.0.0.1:12345", "", nil, false); got != 403 {
		t.Fatalf("missing header returned %d", got)
	}
	payload, _ := json.Marshal(map[string]any{"clientId": "123", "playerName": "Demo", "realMode": false})
	if got := test("POST", "/secret/api/config", "127.0.0.1:12345", "https://evil.example", payload, true); got != 403 {
		t.Fatalf("cross-origin returned %d", got)
	}
	if got := test("POST", "/secret/api/config", "127.0.0.1:12345", "http://127.0.0.1:12345", payload, true); got != 200 {
		t.Fatalf("save config returned %d", got)
	}
	if c := d.config(); c.PlayerName != "Demo" {
		t.Fatalf("config wasn't saved: %+v", c)
	}
}

func TestConfigLanguageAndPredictionWindow(t *testing.T) {
	dir := t.TempDir()
	p := paths{dir: dir, config: filepath.Join(dir, "config.json"), secret: filepath.Join(dir, "tokens.dpapi"), state: filepath.Join(dir, "state.json")}
	d, err := newDashboard(p, Config{})
	if err != nil {
		t.Fatal(err)
	}
	h := d.routes("127.0.0.1:12345", "secret")
	post := func(body map[string]any) int {
		payload, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "http://127.0.0.1:12345/secret/api/config", bytes.NewReader(payload))
		r.Host = "127.0.0.1:12345"
		r.Header.Set("X-Companion", "1")
		r.Header.Set("Origin", "http://127.0.0.1:12345")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	base := map[string]any{"clientId": "123", "playerName": "Demo", "realMode": false}
	if got := post(map[string]any{"clientId": "123", "playerName": "Demo", "realMode": false, "lang": "ru", "predictionWindowMin": 5}); got != 200 {
		t.Fatalf("save with lang/window returned %d", got)
	}
	c := d.config()
	if c.Lang != "ru" || c.PredictionWindowMin != 5 || c.predictionWindowMinutes() != 5 {
		t.Fatalf("lang/window not saved: %+v", c)
	}
	if got := post(map[string]any{"clientId": "123", "playerName": "Demo", "realMode": false, "lang": "fr", "predictionWindowMin": 5}); got != 400 {
		t.Fatalf("invalid lang returned %d", got)
	}
	if got := post(map[string]any{"clientId": "123", "playerName": "Demo", "realMode": false, "lang": "ru", "predictionWindowMin": 45}); got != 400 {
		t.Fatalf("oversize window returned %d", got)
	}
	// Default window and normalized empty language.
	if got := post(base); got != 200 {
		t.Fatalf("minimal save returned %d", got)
	}
	if c = d.config(); c.lang() != "en" || c.predictionWindowMinutes() != 2 {
		t.Fatalf("defaults wrong: %+v", c)
	}
	// Start monitoring, then verify only the language may change while running.
	if got := post(map[string]any{"clientId": "123", "playerName": "Demo", "realMode": false, "confirmedRealMode": false}); got != 200 {
		t.Fatalf("prepare config returned %d", got)
	}
	// Test configuration locking without starting a Twitch network worker.
	d.engine.mu.Lock()
	d.engine.running = true
	d.engine.mu.Unlock()
	if got := post(map[string]any{"clientId": "123", "playerName": "Other", "realMode": false, "lang": "ru"}); got != 400 {
		t.Fatalf("field change while running returned %d", got)
	}
	if got := post(map[string]any{"clientId": "123", "playerName": "Demo", "realMode": false, "lang": "ru"}); got != 200 {
		t.Fatalf("lang-only change while running returned %d", got)
	}
	if c = d.config(); c.lang() != "ru" || c.PlayerName != "Demo" {
		t.Fatalf("lang-only change not applied: %+v", c)
	}
	// Finishing the setup wizard is a live-safe change too.
	if got := post(map[string]any{"clientId": "123", "playerName": "Demo", "realMode": false, "lang": "ru", "wizardDone": true}); got != 200 {
		t.Fatalf("wizardDone change while running returned %d", got)
	}
	if c = d.config(); !c.WizardDone {
		t.Fatal("wizardDone not saved")
	}
}

func TestPredictionTextsAndReplayRootCheck(t *testing.T) {
	dir := t.TempDir()
	p := paths{dir: dir, config: filepath.Join(dir, "config.json"), secret: filepath.Join(dir, "tokens.dpapi"), state: filepath.Join(dir, "state.json")}
	rp := filepath.Join(dir, "replays")
	if err := os.MkdirAll(rp, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rp, "a.StormReplay"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	d, err := newDashboard(p, Config{PlayerName: "Demo", ReplayRoot: rp})
	if err != nil {
		t.Fatal(err)
	}
	// Resetting the custom folder must inspect defaults, not the saved custom
	// path. Check before changing the stored config via the API.
	defaults, err := d.checkReplayRoot("")
	if err != nil { t.Fatal(err) }
	defaultEntries := defaults["entries"].([]replayRootEntry)
	if defaults["custom"] != false || len(defaultEntries) != len(defaultReplayRoots()) {
		t.Fatalf("reset should inspect default roots: %+v", defaults)
	}
	for i, entry := range defaultEntries {
		if entry.Path != defaultReplayRoots()[i] || entry.Path == rp || entry.Custom {
			t.Fatalf("custom replay root leaked into default check: %+v", entry)
		}
	}
	h := d.routes("127.0.0.1:12345", "secret")
	post := func(route string, body map[string]any) (int, map[string]any) {
		payload, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "http://127.0.0.1:12345/secret/api/"+route, bytes.NewReader(payload))
		r.Host = "127.0.0.1:12345"
		r.Header.Set("X-Companion", "1")
		r.Header.Set("Origin", "http://127.0.0.1:12345")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	// Custom prediction texts round-trip.
	if got, _ := post("config", map[string]any{"playerName": "Demo", "realMode": false, "predictionTitle": "Win?", "predictionWin": "Yes", "predictionLoss": "No"}); got != 200 {
		t.Fatalf("custom prediction texts rejected: %d", got)
	}
	c := d.config()
	if c.PredictionTitle != "Win?" || c.PredictionWin != "Yes" || c.PredictionLoss != "No" {
		t.Fatalf("prediction texts not saved: %+v", c)
	}
	if title, win, loss := c.predictionTexts(); title != "Win?" || win != "Yes" || loss != "No" {
		t.Fatalf("resolved texts wrong: %q %q %q", title, win, loss)
	}
	// Length limits (Twitch: 45 title, 25 outcomes).
	long := strings.Repeat("й", 46)
	if got, out := post("config", map[string]any{"playerName": "Demo", "realMode": false, "predictionTitle": long}); got != 400 {
		t.Fatalf("oversize title accepted: %d %v", got, out)
	}
	if got, _ := post("config", map[string]any{"playerName": "Demo", "realMode": false, "predictionWin": "Same", "predictionLoss": "same"}); got != 400 {
		t.Fatal("equal outcomes accepted")
	}
	// Defaults resolve via the message catalog when empty.
	if title, win, loss := (Config{Lang: "en"}).predictionTexts(); title != "HotS win?" || win != "Win" || loss != "Loss" {
		t.Fatalf("english defaults wrong: %q %q %q", title, win, loss)
	}
	// Replay root check: explicit path from the input field.
	got, out := post("replayroot/check", map[string]any{"path": rp})
	if got != 200 {
		t.Fatalf("replayroot check returned %d", got)
	}
	entries, _ := out["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %v", out)
	}
	entry, _ := entries[0].(map[string]any)
	if entry["path"] != rp || entry["exists"] != true || entry["replays"] != float64(1) {
		t.Fatalf("root entry wrong: %v", entry)
	}
	// Empty path checks the default folders (Documents / OneDrive).
	got, out = post("replayroot/check", map[string]any{"path": ""})
	if got != 200 || out["custom"] != false {
		t.Fatalf("default check wrong: %d %v", got, out)
	}
	// Non-existing path is reported as missing.
	got, out = post("replayroot/check", map[string]any{"path": filepath.Join(dir, "missing")})
	if got != 200 {
		t.Fatalf("missing root check returned %d", got)
	}
	entries, _ = out["entries"].([]any)
	entry, _ = entries[0].(map[string]any)
	if entry["exists"] != false {
		t.Fatalf("missing root reported as existing: %v", entry)
	}
}

func TestDashboardReplayAndAssetDelivery(t *testing.T) {
	dir := t.TempDir()
	p := paths{dir: dir, config: filepath.Join(dir, "config.json"), secret: filepath.Join(dir, "tokens.dpapi"), state: filepath.Join(dir, "state.json")}
	rp := filepath.Join(dir, "replays")
	if err := os.MkdirAll(rp, 0700); err != nil {
		t.Fatal(err)
	}
	details := detailsTest()
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	if _, err := zw.Write(details); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	fixture := buildTestMPQ(append([]byte{0x02}, z.Bytes()...), details)
	name := filepath.Join(rp, "Garden.StormReplay")
	if err := os.WriteFile(name, fixture, 0600); err != nil {
		t.Fatal(err)
	}
	d, err := newDashboard(p, Config{PlayerName: "Проверка", ReplayRoot: rp})
	if err != nil {
		t.Fatal(err)
	}
	h := d.routes("127.0.0.1:12345", "secret")
	get := func(path string, withHeader bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "http://127.0.0.1:12345"+path, nil)
		r.Host = "127.0.0.1:12345"
		if withHeader {
			r.Header.Set("X-Companion", "1")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	page := get("/secret/", false)
	if page.Code != 200 || !bytes.Contains(page.Body.Bytes(), []byte("HotS Twitch Companion")) {
		t.Fatalf("UI not served: %d", page.Code)
	}
	script := get("/secret/assets/app.js", false)
	if script.Code != 200 || !bytes.Contains(script.Body.Bytes(), []byte("automation/start")) {
		t.Fatalf("JS missing: %d", script.Code)
	}
	// All standalone JS/CSS modules must be embedded and served by the secure local server.
	for _, asset := range []string{
		"core.js", "i18n.js", "replays.js", "diagnostics.js", "onboarding.js", "app.js",
		"style.css", "wizard.css", "overview.css", "onboarding.css", "diagnostics.css", "settings.css",
	} {
		res := get("/secret/assets/"+asset, false)
		if res.Code != http.StatusOK || res.Body.Len() == 0 {
			t.Fatalf("embedded UI asset %q unavailable: status %d", asset, res.Code)
		}
	}
	boot := get("/secret/api/bootstrap", true)
	if boot.Code != 200 {
		t.Fatalf("bootstrap returned %d", boot.Code)
	}
	var x struct {
		Replays []replaySummary `json:"replays"`
	}
	if err := json.Unmarshal(boot.Body.Bytes(), &x); err != nil {
		t.Fatal(err)
	}
	if len(x.Replays) != 1 || x.Replays[0].Result != "win" {
		t.Fatalf("replay not parsed: %+v", x.Replays)
	}
	if len(x.Replays[0].Players) != 2 || x.Replays[0].Selected != "Проверка" {
		t.Fatalf("game data not exposed: %+v", x.Replays[0])
	}
	// A parsed replay is not a successful player test if the configured identity is absent.
	missing, err := newDashboard(p, Config{PlayerName: "NotPresent", ReplayRoot: rp})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "http://127.0.0.1:12345/secret/api/replay/check", bytes.NewReader([]byte("{}")))
	request.Host = "127.0.0.1:12345"
	request.Header.Set("X-Companion", "1")
	request.Header.Set("Origin", "http://127.0.0.1:12345")
	response := httptest.NewRecorder()
	missing.routes("127.0.0.1:12345", "secret").ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("replay check returned %d: %s", response.Code, response.Body.String())
	}
	var checked map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &checked); err != nil {
		t.Fatal(err)
	}
	if string(checked["selected"]) != "" || len(checked["playerError"]) == 0 {
		t.Fatalf("missing-player check must report playerError without selected player: %s", response.Body.String())
	}
	missingReplays := missing.currentReplays()
	if len(missingReplays) != 1 || missingReplays[0].SelectionError == "" ||
		missingReplays[0].Result != "unknown" || missingReplays[0].Error != "" {
		t.Fatalf("replay selection failure should be separate from parse errors: %+v", missingReplays)
	}
	before, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fixture, before) {
		t.Fatal("game replay modified")
	}
}



// Twitch Device Flow must back off on slow_down instead of continuously
// requesting tokens at the original fixed interval.
func TestDevicePollBackoff(t *testing.T) {
	interval := 5*time.Second
	slow := &twitchHTTPError{Status:400,Message:"slow_down"}
	pending := &twitchHTTPError{Status:400,Message:"authorization_pending"}
	next, retry := devicePollRetry(interval, pending)
	if !retry || next != interval { t.Fatalf("pending must keep interval: %s %t",next,retry) }
	next, retry = devicePollRetry(interval, slow)
	if !retry || next != 10*time.Second { t.Fatalf("slow_down must add 5 seconds: %s %t",next,retry) }
	next, retry = devicePollRetry(next, slow)
	if !retry || next != 15*time.Second { t.Fatalf("consecutive slow_down must accumulate: %s %t",next,retry) }
	next, retry = devicePollRetry(next, &twitchHTTPError{Status:403,Message:"invalid_client"})
	if retry || next != 15*time.Second { t.Fatalf("unrecoverable errors must stop polling: %s %t",next,retry) }
}

func TestDashboardPhases(t *testing.T) {
	for _, tt := range []struct {
		hasToken, validated, pending bool
		want string
	}{
		{false, false, false, "disconnected"},
		{false, false, true, "authorizing"},
		{true, false, false, "unverified"},
		{true, true, false, "connected"},
		{true, true, true, "authorizing"},
	} {
		if got := connectionPhase(tt.hasToken, tt.validated, tt.pending); got != tt.want {
			t.Fatalf("connectionPhase(%v,%v,%v) = %q, want %q", tt.hasToken, tt.validated, tt.pending, got, tt.want)
		}
	}
	for _, tt := range []struct {
		running, active bool
		want string
	}{
		{false, false, "off"},
		{false, true, "off"},
		{true, false, "watching"},
		{true, true, "tracking"},
	} {
		if got := monitorPhase(tt.running, tt.active); got != tt.want {
			t.Fatalf("monitorPhase(%v,%v) = %q, want %q", tt.running, tt.active, got, tt.want)
		}
	}
	now := time.Unix(1_700_000_000, 0)
	if phase, seconds := predictionPhase(nil, 2, now); phase != "none" || seconds != 0 {
		t.Fatalf("no prediction: %s %d", phase, seconds)
	}
	p := &Prediction{StartedAt: now.Add(-30 * time.Second).UnixMilli(), DryRun: true}
	if phase, seconds := predictionPhase(p, 2, now); phase != "simulation" || seconds != 0 {
		t.Fatalf("simulation: %s %d", phase, seconds)
	}
	p.DryRun = false
	if phase, seconds := predictionPhase(p, 2, now); phase != "window_estimate" || seconds != 90 {
		t.Fatalf("window estimate: %s %d", phase, seconds)
	}
	if phase, seconds := predictionPhase(p, 2, now.Add(100*time.Second)); phase != "awaiting_result" || seconds != 0 {
		t.Fatalf("expired window: %s %d", phase, seconds)
	}
}

func TestConfigSaveKeepsVerifiedTwitchIdentity(t *testing.T) {
	dir := t.TempDir()
	p := paths{dir: dir, config: filepath.Join(dir, "config.json"), secret: filepath.Join(dir, "tokens.dpapi"), state: filepath.Join(dir, "state.json")}
	d, err := newDashboard(p, Config{RealMode: true})
	if err != nil { t.Fatal(err) }
	// Completed device authorization: token persisted, identity verified.
	tok := Token{AccessToken: "at", RefreshToken: "rt", ExpiresIn: 3600, SavedAt: time.Now().Unix()}
	d.engine.twitch.mu.Lock()
	if err := d.engine.twitch.saveToken(tok); err != nil { t.Fatal(err) }
	d.engine.twitch.login = "demo"
	d.engine.twitch.broadcasterID = "42"
	d.engine.twitch.mu.Unlock()
	// Saving settings (e.g. finishing setup or the launch step) rebuilds the
	// Twitch client from disk; the verified login must survive it.
	if err := d.engine.updateConfig(Config{RealMode: true, WizardDone: true}); err != nil { t.Fatal(err) }
	st := d.status()
	if st["connected"] != true || st["twitchPhase"] != "connected" || st["channel"] != "demo" {
		t.Fatalf("config save dropped the verified Twitch identity: %#v", st)
	}
	// Switching to another OAuth application invalidates the identity: the
	// stored token belongs to the previous app and must be re-authorized.
	if err := d.engine.updateConfig(Config{ClientID: "other-app", RealMode: true, WizardDone: true}); err != nil { t.Fatal(err) }
	st = d.status()
	if st["connected"] != false || st["channel"] != "" {
		t.Fatalf("identity must not survive an OAuth app change: %#v", st)
	}
}


func TestTwitchReauthorizationCannotStealActivePrediction(t *testing.T) {
	d := manualDashboard(t, Config{ClientID: "client", RealMode: true, Lang: "en"})
	initial := Token{AccessToken: "original", RefreshToken: "original-refresh", ExpiresIn: 3600, SavedAt: time.Now().Unix()}
	d.engine.twitch.mu.Lock()
	if err := d.engine.twitch.saveToken(initial); err != nil { d.engine.twitch.mu.Unlock(); t.Fatal(err) }
	d.engine.twitch.login, d.engine.twitch.broadcasterID = "original-login", "channel-A"
	d.engine.twitch.mu.Unlock()
	started := time.Now().UnixMilli()
	if err := d.engine.updateAndSave(State{Active: &Prediction{ID: "pred-A", BroadcasterID: "channel-A", WinID:"yes", LossID:"no", StartedAt:started}}); err != nil { t.Fatal(err) }

	oldClient := httpClient
	defer func() { httpClient = oldClient }()
	userID := "channel-B"
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/oauth2/validate" { t.Errorf("unexpected request %s",req.URL); return nil, fmt.Errorf("unexpected request") }
		if req.Header.Get("Authorization") != "OAuth candidate" { t.Errorf("wrong candidate credential: %s",req.Header.Get("Authorization")) }
		return manualResponse(fmt.Sprintf(`{"client_id":"client","user_id":%q,"login":"candidate-login","scopes":["channel:manage:predictions"]}`, userID)), nil
	})}
	candidate := Token{AccessToken:"candidate", RefreshToken:"candidate-refresh", ExpiresIn:3600, SavedAt:time.Now().Unix()}
	if _, err := d.engine.installTwitchToken(context.Background(), "client", candidate); err == nil {
		t.Fatal("replacing an active prediction's broadcaster was permitted")
	}
	d.engine.twitch.mu.Lock()
	got := d.engine.twitch.token
	login := d.engine.twitch.login
	d.engine.twitch.mu.Unlock()
	if got.AccessToken != "original" || login != "original-login" {
		t.Fatalf("rejected auth modified credentials: %+v login=%s", got, login)
	}
	reopened, err := newTwitch(d.p, "client", "en")
	if err != nil || reopened.token.AccessToken != "original" { t.Fatalf("persisted original token lost: %+v %v", reopened, err) }
	if d.engine.state.Active == nil || d.engine.state.Active.ID != "pred-A" { t.Fatal("active prediction lost") }

	userID = "channel-A"
	if me, err := d.engine.installTwitchToken(context.Background(), "client", candidate); err != nil || me.UserID != "channel-A" {
		t.Fatalf("same broadcaster must be able to reconnect: %+v %v", me, err)
	}
	reopened, err = newTwitch(d.p, "client", "en")
	if err != nil || reopened.token.AccessToken != "candidate" { t.Fatalf("replacement was not persisted: %+v %v", reopened, err) }
}

func TestActiveRealPredictionBlocksClientIDChange(t *testing.T) {
	d := manualDashboard(t, Config{ClientID:"client", RealMode:true})
	if err := d.engine.updateAndSave(State{Active:&Prediction{ID:"pred", BroadcasterID:"A", StartedAt:time.Now().UnixMilli()}}); err != nil { t.Fatal(err) }
	if err := d.engine.updateConfig(Config{ClientID:"different",RealMode:true}); err == nil { t.Fatal("client changed with active prediction") }
	if d.engine.cfg.ClientID != "client" { t.Fatal("config changed despite refusal") }
	if err := d.engine.updateConfig(Config{ClientID:"client",RealMode:true,WizardDone:true}); err != nil { t.Fatalf("unrelated config change blocked: %v", err) }
}

func TestFailedCandidateValidationPreservesCurrentToken(t *testing.T) {
	d := manualDashboard(t, Config{ClientID:"client",RealMode:true})
	d.engine.twitch.mu.Lock()
	if err := d.engine.twitch.saveToken(Token{AccessToken:"keep", RefreshToken:"keep-refresh", ExpiresIn:3600,SavedAt:time.Now().Unix()}); err != nil { t.Fatal(err) }
	d.engine.twitch.mu.Unlock()
	oldClient:=httpClient
	defer func(){httpClient=oldClient}()
	httpClient=&http.Client{Transport:manualTransport(func(r *http.Request)(*http.Response,error){
		return manualResponse(`{"client_id":"wrong","user_id":"A","scopes":["channel:manage:predictions"]}`),nil
	})}
	if _,err:=d.engine.installTwitchToken(context.Background(),"client",Token{AccessToken:"candidate",RefreshToken:"refresh"});err==nil { t.Fatal("wrong OAuth client accepted") }
	reopened,err:=newTwitch(d.p,"client","en")
	if err!=nil || reopened.token.AccessToken!="keep" {t.Fatalf("token overwritten: %+v %v",reopened,err)}
}

func TestUnverifiedTwitchTokenNotShownAsConnected(t *testing.T) {
	dir := t.TempDir()
	p := paths{dir: dir, config: filepath.Join(dir, "config.json"), secret: filepath.Join(dir, "tokens.dpapi"), state: filepath.Join(dir, "state.json")}
	d, err := newDashboard(p, Config{})
	if err != nil { t.Fatal(err) }
	d.engine.twitch.mu.Lock()
	d.engine.twitch.token.AccessToken = "not-validated"
	d.engine.twitch.mu.Unlock()
	st := d.status()
	if st["connected"] != false || st["twitchPhase"] != "unverified" {
		t.Fatalf("token should not mean a verified connection: %#v", st)
	}
	d.engine.twitch.mu.Lock()
	d.engine.twitch.login = "demo"
	d.engine.twitch.mu.Unlock()
	st = d.status()
	if st["connected"] != true || st["twitchPhase"] != "connected" {
		t.Fatalf("verified connection not reflected: %#v", st)
	}
}

func TestStructuredDiagnostics(t *testing.T) {
	dir := t.TempDir()
	p := paths{dir: dir, config: filepath.Join(dir, "config.json"), secret: filepath.Join(dir, "tokens.dpapi"), state: filepath.Join(dir, "state.json")}
	d, err := newDashboard(p, Config{})
	if err != nil {
		t.Fatal(err)
	}
	d.logEvent("error", "prediction_create", "Prediction API failed")
	d.logEvent("warning", "replay_process", "Player not found")
	d.logEvent("invalid", "unknown", "Normal note")
	status := d.status()
	logs, ok := status["logs"].([]event)
	if !ok || len(logs) < 3 {
		t.Fatalf("events not present in status: %#v", status["logs"])
	}
	last := logs[len(logs)-3:]
	if last[0].Level != "error" || last[0].Code != "prediction_create" ||
		last[1].Level != "warning" || last[1].Code != "replay_process" ||
		last[2].Level != "info" {
		t.Fatalf("incorrect diagnostic levels: %+v", last)
	}
	data, err := json.Marshal(last)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"level":"error"`)) || !bytes.Contains(data, []byte(`"code":"prediction_create"`)) {
		t.Fatalf("structured fields not available to UI: %s", data)
	}
}
