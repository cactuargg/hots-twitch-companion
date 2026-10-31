package dashboard

import (
	"hots-twitch-companion/internal/sessions"
	"time"
)

type chanceDisplay struct {
	Percent      float64 `json:"percent"`
	Label        string  `json:"label"`
	VisibleForMS int64   `json:"visibleForMS"`
}

func chanceTiming(p sessions.Preferences) (int, int, int) {
	a, b, c := p.ChanceInitialSeconds, p.ChanceIntervalSeconds, p.ChanceDurationSeconds
	if a == 0 {
		a = 15
	}
	if b == 0 {
		b = 120
	}
	if c == 0 {
		c = 10
	}
	return a, b, c
}

// A short display lease prevents stale percentages persisting if OBS loses
// contact. Timing is anchored to BattleLobby creation, not browser page load.
func scheduledChance(s lobbyWidgetState, p sessions.Preferences, now int64) *chanceDisplay {
	if !p.ChanceEnabled || s.Phase != "live" || s.Estimate == nil || s.StartedAt <= 0 || now < s.StartedAt {
		return nil
	}
	a, b, c := chanceTiming(p)
	elapsed := now - s.StartedAt
	remaining := int64(0)
	if elapsed < int64(a)*1000 {
		remaining = int64(a)*1000 - elapsed
	} else {
		offset := elapsed % (int64(b) * 1000)
		if elapsed >= int64(b)*1000 && offset < int64(c)*1000 {
			remaining = int64(c)*1000 - offset
		}
	}
	if remaining <= 0 {
		return nil
	}
	if remaining > 5000 {
		remaining = 5000
	}
	label := "Win estimate"
	if s.Language == "ru" {
		label = "Оценка победы"
	}
	return &chanceDisplay{Percent: s.Estimate.Percent, Label: label, VisibleForMS: remaining}
}

func (d *Dashboard) scoreChance(p sessions.Preferences) *chanceDisplay {
	if !p.ChanceEnabled {
		return nil
	}
	return scheduledChance(d.currentLobby(true), p, time.Now().UnixMilli())
}
