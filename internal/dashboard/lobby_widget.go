package dashboard

import (
	"fmt"
	"hots-twitch-companion/internal/hots/lobby"
	"hots-twitch-companion/internal/replaystats"
	"os"
	"strconv"
	"strings"
	"time"
)

type widgetCounts struct {
	Games   int      `json:"games"`
	Wins    int      `json:"wins"`
	Losses  int      `json:"losses"`
	WinRate *float64 `json:"winRate"`
}
type widgetPerson struct {
	Name     string        `json:"name"`
	Hero     string        `json:"hero"`
	Overall  widgetCounts  `json:"overall"`
	OnHero   *widgetCounts `json:"onHero"`
	Meetings int           `json:"meetings"`
}
type lobbyWidgetState struct {
	Enabled     bool           `json:"enabled"`
	Phase       string         `json:"phase"`
	Language    string         `json:"language"`
	Mode        uint32         `json:"mode"`
	HeroPending bool           `json:"heroPending"`
	Streamer    widgetPerson   `json:"streamer"`
	Familiar    []widgetPerson `json:"familiar"`
}

func counts(row replaystats.Row) widgetCounts {
	v := widgetCounts{Games: row.Games, Wins: row.Wins, Losses: row.Losses}
	if row.Games > 0 {
		rate := row.WinRate
		v.WinRate = &rate
	}
	return v
}
func personWidget(p replaystats.Person, code, language string, heroAllowed bool) widgetPerson {
	v := widgetPerson{Name: p.Overall.Name, Overall: counts(p.Overall), Meetings: p.Meetings}
	if !heroAllowed {
		return v
	}
	v.Hero = lobby.HeroName(code, language)
	if v.Hero != "" {
		c := widgetCounts{}
		for _, h := range p.Heroes {
			if lobby.HeroMatches(code, h.Name) {
				c.Games += h.Games
				c.Wins += h.Wins
				c.Losses += h.Losses
			}
		}
		if c.Games > 0 {
			rate := 100 * float64(c.Wins) / float64(c.Games)
			c.WinRate = &rate
		}
		v.OnHero = &c
	}
	return v
}
func (d *Dashboard) lobbyWidget() lobbyWidgetState {
	cfg := d.Config()
	result := lobbyWidgetState{Phase: "waiting", Language: cfg.Language(), Familiar: []widgetPerson{}}
	_, prefs, err := d.overlays.store.DisplaySettings()
	if err != nil {
		result.Phase = "unavailable"
		return result
	}
	result.Enabled = prefs.LobbyWidgetEnabled
	if !result.Enabled {
		result.Phase = "disabled"
		return result
	}
	o := d.overlays
	o.mu.Lock()
	snap, path, phase, player, checked, observed := o.widgetLobby, o.widgetPath, o.phase, o.lastPlayer, o.checkedAt, o.observedAt
	o.mu.Unlock()
	now := time.Now()
	selected := strings.ToLower(cfg.PlayerToonHandle)
	if selected == "" || player != selected || snap == nil || snap.Data == nil || snap.Status != "roster_parsed" || (phase != "loading_or_game" && phase != "excluded_mode") || checked > now.UnixMilli() || now.UnixMilli()-checked > 15000 || observed > now.UnixMilli() || now.UnixMilli()-observed > 100*time.Minute.Milliseconds() {
		return result
	}
	st, err := os.Stat(path)
	if err != nil || st.ModTime().UnixMilli() != snap.ModifiedAt || st.Size() != snap.Size || d.statistics.LobbyCompleted(snap.SHA256) {
		return result
	}
	roster := snap.Data.Players
	found := 0
	ids := []string{}
	seen := map[string]bool{}
	for _, p := range roster {
		if p.SlotType != "Humn" || p.Role != "Part" || p.ToonHandle == "" {
			continue
		}
		id := strings.ToLower(p.ToonHandle)
		if seen[id] {
			return result
		}
		seen[id] = true
		ids = append(ids, id)
		if id == selected {
			found++
		}
	}
	if found != 1 {
		return result
	}
	key := fmt.Sprintf("%s|%s|%s|%s|%d", selected, snap.SHA256, cfg.PlayerName, cfg.Language(), d.statistics.Revision())
	o.mu.Lock()
	if o.widgetCacheKey == key {
		cached := o.widgetCache
		o.mu.Unlock()
		return cached
	}
	o.mu.Unlock()
	people, err := d.statistics.People(strconv.FormatUint(uint64(snap.Data.AMMID), 10), selected, ids)
	if err != nil {
		result.Phase = "unavailable"
		return result
	}
	result.Phase = "live"
	result.Mode = snap.Data.AMMID
	result.HeroPending = result.Mode == 50101
	for _, p := range roster {
		id := strings.ToLower(p.ToonHandle)
		if p.SlotType != "Humn" || p.Role != "Part" || !seen[id] {
			continue
		}
		person, known := people[id]
		if id != selected && !known {
			continue
		}
		v := personWidget(person, p.Hero, cfg.Language(), result.Mode != 50101)
		if strings.EqualFold(v.Name, p.ToonHandle) {
			v.Name = ""
		}
		if id == selected {
			if cfg.PlayerName != "" {
				v.Name = cfg.PlayerName
			}
			result.Streamer = v
			if v.Hero == "" {
				result.HeroPending = true
			}
		} else {
			result.Familiar = append(result.Familiar, v)
		}
	}
	// Only display names and aggregate counts leave this endpoint. No account IDs,
	// source files, lobby hashes, tokens, or full replay database are serialized.
	o.mu.Lock()
	o.widgetCacheKey = key
	o.widgetCache = result
	o.mu.Unlock()
	return result
}
