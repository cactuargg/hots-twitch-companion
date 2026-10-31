package replaystats

import (
	"hots-twitch-companion/internal/hots/lobby"
	"math"
	"strings"
)

// Estimate is a deliberately conservative, uncalibrated local-history heuristic.
// Coverage counts players with >=5 observations or heroes with >=20 observations.
type Estimate struct {
	Percent  float64 `json:"percent"`
	Matches  int     `json:"matches"`
	Coverage int     `json:"coverage"`
}

type tally struct{ games, wins int }

func (t tally) logOdds() float64 { return math.Log(float64(t.wins+10) / float64(t.games-t.wins+10)) }

// EstimateLobby uses only completed, consistent human 5v5 matches predating the
// current lobby. The current outcome cannot update its own prediction.
func (s *Store) EstimateLobby(b *lobby.BattleLobby, streamer string, before int64) *Estimate {
	if b == nil || b.Custom || b.TeamSize != "5v5" {
		return nil
	}
	switch b.AMMID {
	case 50091, 50001, 50051, 50101:
	default:
		return nil
	}
	roster := []lobby.LobbyPlayer{}
	team, seen, sizes := "", map[string]bool{}, map[string]int{}
	for _, p := range b.Players {
		if p.Role != "Part" {
			continue
		}
		id := strings.ToLower(p.ToonHandle)
		if p.SlotType != "Humn" || !humanToon.MatchString(id) || seen[id] || (p.Team != "T1" && p.Team != "T2") {
			return nil
		}
		seen[id] = true
		sizes[p.Team]++
		roster = append(roster, p)
		if strings.EqualFold(id, streamer) {
			team = p.Team
		}
	}
	if team == "" || sizes["T1"] != 5 || sizes["T2"] != 5 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil
	}
	players, heroes, personal := map[string]tally{}, map[string]tally{}, map[string]tally{}
	codes, resolved := map[string]bool{}, map[string]string{}
	if b.AMMID != 50101 {
		for _, p := range roster {
			if lobby.HeroName(p.Hero, "en") != "" {
				codes[p.Hero] = true
			}
		}
	}
	heroCode := func(name string) string {
		if code, ok := resolved[name]; ok {
			return code
		}
		for code := range codes {
			if lobby.HeroMatches(code, name) {
				resolved[name] = code
				return code
			}
		}
		resolved[name] = ""
		return ""
	}
	matches := 0
	for _, m := range s.data.Matches {
		if m.Mode != b.AMMID || m.Replay.RecordedAt >= before || len(m.Replay.Players) != 10 {
			continue
		}
		ids, teams, results := map[string]bool{}, map[int64]int{}, map[int64]string{}
		valid := true
		for _, p := range m.Replay.Players {
			id := strings.ToLower(p.ToonHandle)
			if !humanToon.MatchString(id) || ids[id] || (p.Team != 0 && p.Team != 1) || !validResult(p.Result) || (results[p.Team] != "" && results[p.Team] != p.Result) {
				valid = false
				break
			}
			ids[id] = true
			teams[p.Team]++
			results[p.Team] = p.Result
		}
		if !valid || teams[0] != 5 || teams[1] != 5 || results[0] == results[1] {
			continue
		}
		matches++
		for _, p := range m.Replay.Players {
			id := strings.ToLower(p.ToonHandle)
			add := func(v tally) tally {
				v.games++
				if p.Result == "win" {
					v.wins++
				}
				return v
			}
			players[id] = add(players[id])
			// Resolve localized hero names once per distinct name, not per replay.
			if code := heroCode(p.Hero); code != "" {
				heroes[code] = add(heroes[code])
				personal[id+"|"+code] = add(personal[id+"|"+code])
			}
		}
	}
	if matches < 20 {
		return nil
	}
	scores, covered := map[string]float64{}, map[string]int{}
	for _, p := range roster {
		id := strings.ToLower(p.ToonHandle)
		base := players[id]
		value := 0.0
		known := false
		if base.games >= 5 {
			value = base.logOdds()
			known = true
		}
		if b.AMMID != 50101 && lobby.HeroName(p.Hero, "en") != "" {
			h := heroes[p.Hero]
			ph := personal[id+"|"+p.Hero]
			if h.games >= 20 {
				value += 0.25 * h.logOdds()
				known = true
			}
			if base.games >= 5 && ph.games >= 5 {
				value += 0.25 * (ph.logOdds() - base.logOdds())
			}
		}
		scores[p.Team] += value / 5
		if known {
			covered[p.Team]++
		}
	}
	coverage := covered["T1"] + covered["T2"]
	if coverage < 6 || covered["T1"] < 2 || covered["T2"] < 2 {
		return nil
	}
	other := "T1"
	if team == "T1" {
		other = "T2"
	}
	p := 100 / (1 + math.Exp(-(scores[team] - scores[other])))
	p = math.Max(20, math.Min(80, p))
	return &Estimate{Percent: math.Round(p), Matches: matches, Coverage: coverage}
}
