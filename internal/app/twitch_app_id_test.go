package app

import (
	"hots-twitch-companion/internal/config"
	"path/filepath"
	"testing"
)

func TestTwitchPublicAppIDFallbackAndOverride(t *testing.T) {
	previous := defaultTwitchClientID
	defer func() { defaultTwitchClientID = previous }()
	defaultTwitchClientID = "public_app_id_123456789012345"
	if got := (config.Data{}).TwitchClientID(defaultTwitchClientID); got != defaultTwitchClientID {
		t.Fatalf("missing built-in Client ID: %q", got)
	}
	if got := (config.Data{ClientID: "  private_override  "}).TwitchClientID(defaultTwitchClientID); got != "private_override" {
		t.Fatalf("custom Client ID should override the app default: %q", got)
	}
	cfg := config.Data{Lang: "ru"}
	dir := t.TempDir()
	profile := config.Paths{Dir: dir, Config: filepath.Join(dir, "config.json"), State: filepath.Join(dir, "state.json"), Secret: filepath.Join(dir, "tokens.dpapi")}
	eng, err := newEngine(profile, cfg, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if eng.twitch.ClientID() != defaultTwitchClientID {
		t.Fatalf("engine is not using the built-in public app: %q", eng.twitch.ClientID())
	}
	if err := eng.UpdateConfig(config.Data{Lang: "ru", ClientID: "my_own_app"}); err != nil {
		t.Fatal(err)
	}
	if eng.twitch.ClientID() != "my_own_app" {
		t.Fatalf("custom ID not honored: %q", eng.twitch.ClientID())
	}
	if err := eng.UpdateConfig(config.Data{Lang: "ru"}); err != nil {
		t.Fatal(err)
	}
	if eng.twitch.ClientID() != defaultTwitchClientID {
		t.Fatalf("clearing override did not restore public app: %q", eng.twitch.ClientID())
	}
	defaultTwitchClientID = ""
	if got := (config.Data{}).TwitchClientID(defaultTwitchClientID); got != "" {
		t.Fatalf("unexpected id with no builtin or override: %q", got)
	}
}

func TestStatusOnlyReportsPublicAppAvailabilityNotClientID(t *testing.T) {
	previous := defaultTwitchClientID
	defer func() { defaultTwitchClientID = previous }()
	defaultTwitchClientID = "public_abc1234567890123"
	d := manualDashboard(t, config.Data{Lang: "en"})
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
	d.engine.defaultClientID = ""
	d.engine.mu.Unlock()
	if available := d.status()["hasTwitchApp"]; available != false {
		t.Fatalf("no ID is available: %v", available)
	}
}
