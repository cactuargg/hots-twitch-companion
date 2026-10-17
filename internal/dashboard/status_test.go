package dashboard

import (
	"hots-twitch-companion/internal/prediction"
	"testing"
	"time"
)

func TestConnectionAndMonitorPhases(t *testing.T) {
	for _, tc := range []struct {
		has, valid, pending bool
		want                string
	}{
		{false, false, false, "disconnected"},
		{true, false, false, "unverified"},
		{true, true, false, "connected"},
		{true, true, true, "authorizing"},
	} {
		if got := ConnectionPhase(tc.has, tc.valid, tc.pending); got != tc.want {
			t.Fatalf("connection %v: %s", tc, got)
		}
	}
	if MonitorPhase(false, false) != "off" || MonitorPhase(true, false) != "watching" || MonitorPhase(true, true) != "tracking" {
		t.Fatal("monitor phases changed")
	}
}

func TestPredictionWindowBoundary(t *testing.T) {
	now := time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)
	p := &prediction.Prediction{ID: "id", StartedAt: now.UnixMilli()}
	for _, tc := range []struct {
		at      time.Time
		phase   string
		seconds int
	}{
		{now, "window_estimate", 120},
		{now.Add(119 * time.Second), "window_estimate", 1},
		{now.Add(120 * time.Second), "awaiting_result", 0},
		{now.Add(121 * time.Second), "awaiting_result", 0},
	} {
		got, seconds := PredictionPhase(p, 2, tc.at)
		if got != tc.phase || seconds != tc.seconds {
			t.Fatalf("phase at %s = %s/%d, want %s/%d", tc.at, got, seconds, tc.phase, tc.seconds)
		}
	}
	if status, _ := PredictionPhase(nil, 2, now); status != "none" {
		t.Fatalf("nil prediction is %s", status)
	}
	p.DryRun = true
	if status, _ := PredictionPhase(p, 2, now); status != "simulation" {
		t.Fatalf("dry-run prediction is %s", status)
	}
}
