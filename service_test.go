package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if err := d.engine.start(); err != nil {
		t.Fatal(err)
	}
	defer d.engine.stopWatch()
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
	before, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fixture, before) {
		t.Fatal("game replay modified")
	}
}
