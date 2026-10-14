package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	before, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fixture, before) {
		t.Fatal("game replay modified")
	}
}
