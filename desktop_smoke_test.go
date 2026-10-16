package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Exercise the real loopback listener and shutdown, with a callback replacing
// the visible Edge launcher. No live Twitch account or game is required.
func TestDesktopServerStartsServesUIAndExits(t *testing.T) {
	d := manualDashboard(t, Config{Lang: "en"})
	client := &http.Client{Timeout: 3 * time.Second}
	err := d.runDesktop(func(root string) error {
		response, err := client.Get(root)
		if err != nil {
			return err
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			return err
		}
		if response.StatusCode != 200 || !strings.Contains(string(body), "HotS Twitch Companion") || response.Header.Get("Content-Security-Policy") == "" {
			t.Fatal("embedded UI or security headers missing")
		}
		req, err := http.NewRequest("GET", root+"api/status", nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Companion", "1")
		response, err = client.Do(req)
		if err != nil {
			return err
		}
		var status map[string]any
		err = json.NewDecoder(response.Body).Decode(&status)
		response.Body.Close()
		if err != nil {
			return err
		}
		if response.StatusCode != 200 || status["version"] != "0.8.0" || status["active"] != false {
			t.Fatalf("bad startup status: %+v", status)
		}
		parsed, err := url.Parse(root)
		if err != nil {
			return err
		}
		req, err = http.NewRequest("POST", root+"api/quit", strings.NewReader("{}"))
		if err != nil {
			return err
		}
		req.Header.Set("X-Companion", "1")
		req.Header.Set("Origin", parsed.Scheme+"://"+parsed.Host)
		response, err = client.Do(req)
		if err != nil {
			return err
		}
		response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatal("shutdown failed")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
