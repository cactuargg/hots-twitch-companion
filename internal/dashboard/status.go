// Package dashboard owns the embedded UI, guarded HTTP API and status projections.
// It accesses the application through operations and independent snapshots.
package dashboard

import (
	"hots-twitch-companion/internal/prediction"
	"time"
)

func ConnectionPhase(hasToken, validated, pending bool) string {
	if pending {
		return "authorizing"
	}
	if !hasToken {
		return "disconnected"
	}
	if validated {
		return "connected"
	}
	return "unverified"
}

func MonitorPhase(running, tracking bool) string {
	if !running {
		return "off"
	}
	if tracking {
		return "tracking"
	}
	return "watching"
}

func PredictionPhase(p *prediction.Prediction, windowMinutes int, now time.Time) (string, int) {
	if p == nil {
		return "none", 0
	}
	if p.DryRun {
		return "simulation", 0
	}
	// The betting window is an estimate: remote Twitch status is not polled here.
	deadline := time.UnixMilli(p.StartedAt).Add(time.Duration(windowMinutes) * time.Minute)
	remaining := deadline.Sub(now)
	if remaining <= 0 {
		return "awaiting_result", 0
	}
	return "window_estimate", int((remaining + time.Second - 1) / time.Second)
}
