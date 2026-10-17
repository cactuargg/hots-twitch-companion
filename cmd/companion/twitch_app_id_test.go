package main

import (
	"path/filepath"
	"testing"
)

func TestTwitchPublicAppIDFallbackAndOverride(t *testing.T) {
	previous := defaultTwitchClientID
	defer func() { defaultTwitchClientID = previous }()
	defaultTwitchClientID = "public_app_id_123456789012345"
	if got := (Config{}).twitchClientID(); got != defaultTwitchClientID {
		t.Fatalf("missing built-in Client ID: %q", got)
	}
	if got := (Config{ClientID: "  private_override  "}).twitchClientID(); got != "private_override" {
		t.Fatalf("custom Client ID should override the app default: %q", got)
	}
	cfg := Config{Lang: "ru"}
	dir := t.TempDir()
	paths := paths{dir: dir, config: filepath.Join(dir, "config.json"), state: filepath.Join(dir, "state.json"), secret: filepath.Join(dir, "tokens.dpapi")}
	eng, err := newEngine(paths, cfg, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if eng.twitch.ClientID() != defaultTwitchClientID {
		t.Fatalf("engine is not using the built-in public app: %q", eng.twitch.ClientID())
	}
	if err := eng.updateConfig(Config{Lang: "ru", ClientID: "my_own_app"}); err != nil {
		t.Fatal(err)
	}
	if eng.twitch.ClientID() != "my_own_app" {
		t.Fatalf("custom ID not honored: %q", eng.twitch.ClientID())
	}
	if err := eng.updateConfig(Config{Lang: "ru"}); err != nil {
		t.Fatal(err)
	}
	if eng.twitch.ClientID() != defaultTwitchClientID {
		t.Fatalf("clearing override did not restore public app: %q", eng.twitch.ClientID())
	}
	defaultTwitchClientID = ""
	if got := (Config{}).twitchClientID(); got != "" {
		t.Fatalf("unexpected id with no builtin or override: %q", got)
	}
}

func TestStatusOnlyReportsPublicAppAvailabilityNotClientID(t *testing.T) {
	previous := defaultTwitchClientID
	defer func() { defaultTwitchClientID = previous }()
	defaultTwitchClientID = "public_abc1234567890123"
	d := manualDashboard(t, Config{Lang: "en"})
	st := d.status()
	if st["hasTwitchApp"] != true {
		t.Fatalf("expected available built-in app: %+v", st)
	}
	if _, exists := st["clientId"]; exists {
		t.Fatal("the status endpoint should not expose Client ID")
	}
	defaultTwitchClientID = ""
	// Config and engine now don't use any ID.
	d.engine.mu.Lock()
	d.engine.cfg.ClientID = ""
	d.engine.mu.Unlock()
	if available := d.status()["hasTwitchApp"]; available != false {
		t.Fatalf("no ID is available: %v", available)
	}
}
