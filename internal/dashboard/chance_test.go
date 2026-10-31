package dashboard

import (
	"hots-twitch-companion/internal/replaystats"
	"hots-twitch-companion/internal/sessions"
	"testing"
)

func TestChanceScheduleLifecycleAndLease(t *testing.T) {
	s := lobbyWidgetState{Phase: "live", StartedAt: 100000, Language: "ru", Estimate: &replaystats.Estimate{Percent: 62}}
	p := sessions.Preferences{ChanceEnabled: true}
	for _, tc := range []struct {
		elapsed int64
		visible bool
	}{{0, true}, {14999, true}, {15000, false}, {119999, false}, {120000, true}, {129999, true}, {130000, false}, {240000, true}} {
		got := scheduledChance(s, p, s.StartedAt+tc.elapsed)
		if (got != nil) != tc.visible {
			t.Fatalf("at %d: %+v", tc.elapsed, got)
		}
		if got != nil && (got.VisibleForMS <= 0 || got.VisibleForMS > 5000 || got.Percent != 62) {
			t.Fatal(got)
		}
	}
	for _, phase := range []string{"waiting", "unavailable", "disabled"} {
		s.Phase = phase
		if scheduledChance(s, p, s.StartedAt) != nil {
			t.Fatal("old match visible", phase)
		}
	}
	s.Phase = "live"
	p.ChanceEnabled = false
	if scheduledChance(s, p, s.StartedAt) != nil {
		t.Fatal("disabled visible")
	}
	p.ChanceEnabled = true
	s.Estimate = nil
	if scheduledChance(s, p, s.StartedAt) != nil {
		t.Fatal("missing estimate visible")
	}
}

func TestCurrentMatchPanelIndependentOfOBS(t *testing.T) {
	d := widgetFixture(t)
	if err := d.overlays.store.Update(func(s *sessions.State) error { s.Preferences.LobbyWidgetEnabled = false; return nil }); err != nil {
		t.Fatal(err)
	}
	if d.lobbyWidget().Phase != "disabled" || d.currentLobby(true).Phase != "live" {
		t.Fatal("app panel depends on OBS preference")
	}
	d.overlays.phase = "between_games"
	if d.currentLobby(true).Phase != "waiting" {
		t.Fatal("completed match still live")
	}
}
