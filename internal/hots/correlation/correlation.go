// Package correlation validates exact HotS player and lobby evidence.
// All functions are side-effect free; no filesystem or Twitch dependency.
package correlation

import (
 "errors"
 "strings"
 "hots-twitch-companion/internal/hots/lobby"
 "hots-twitch-companion/internal/hots/replay"
 "hots-twitch-companion/internal/prediction"
)

func BindPredictionMatch(lobby *lobby.Snapshot, key, toon string) *prediction.MatchBinding {
	binding := &prediction.MatchBinding{Version: 1, LobbyKey: key, PlayerToon: strings.TrimSpace(toon)}
	if lobby != nil {
		copyLobby := *lobby
		if lobby.Data != nil {
			data := *lobby.Data
			data.Players = append([]lobby.LobbyPlayer(nil), data.Players...)
			copyLobby.Data = &data
		}
		binding.Lobby = &copyLobby
	}
	return binding
}

func ObserveActiveLobby(active *prediction.Prediction, next *lobby.Snapshot, key string) *prediction.Prediction {
	m := active.Match
	if m == nil || m.Lobby == nil || next == nil || m.Lobby.SHA256 == "" || m.Lobby.SHA256 != next.SHA256 || key == m.LobbyKey {
		return active
	}
	// The same bytes were observed again as a new lobby event. We cannot prove
	// whether this was a rewrite or a repeated match; keep that uncertainty durable.
	return AmbiguousMatch(active)
}

func AmbiguousMatch(active *prediction.Prediction) *prediction.Prediction {
	copyPrediction := *active
	copyMatch := *active.Match
	copyMatch.Ambiguous = true
	copyPrediction.Match = &copyMatch
	return &copyPrediction
}

func ParticipantToons(lobby *lobby.BattleLobby) (map[string]bool, error) {
	if lobby == nil || lobby.TeamSize != "5v5" {
		return nil, errors.New("automatic correlation requires a 5v5 lobby")
	}
	if !lobby.Custom {
		switch lobby.AMMID {
		case 50001, 50051, 50091, 50101:
		default:
			return nil, errors.New("unsupported matchmaking mode")
		}
	}
	toons := map[string]bool{}
	humans := 0
	for _, p := range lobby.Players {
		if p.Role == "Watc" {
			continue
		}
		if p.Role != "Part" {
			return nil, errors.New("unexpected lobby role")
		}
		if p.SlotType == "Comp" {
			if p.ToonHandle != "" {
				return nil, errors.New("AI slot carries a ToonHandle")
			}
			continue
		}
		if p.SlotType != "Humn" || p.ToonHandle == "" || !strings.Contains(p.ToonHandle, "-Hero-") || toons[p.ToonHandle] {
			return nil, errors.New("incomplete or non-human roster")
		}
		toons[p.ToonHandle] = true
		humans++
	}
	if !lobby.Custom && humans != 10 {
		return nil, errors.New("expected ten unique participants")
	}
	if humans == 0 {
		return nil, errors.New("no human participants")
	}
	return toons, nil
}

// Blizzard ToonHandles are "<region>-Hero-<realm>-<id>". AI participants get
// synthetic handles without the Hero program and carry no verified identity.
func IsHeroHandle(toon string) bool { return strings.Contains(toon, "-Hero-") }

func CorrelatedOutcome(recording replay.Replay, lobby *lobby.BattleLobby, toon string) (replay.Player, error) {
	toons, err := ParticipantToons(lobby)
	if err != nil {
		return replay.Player{}, err
	}
	if !toons[toon] {
		return replay.Player{}, errors.New("selected ToonHandle is not a participant")
	}
	seen := map[string]bool{}
	counts := [2]int{}
	results := [2]string{}
	var selected replay.Player
	record := func(p replay.Player) error {
		if p.Team < 0 || p.Team > 1 || (p.Result != "win" && p.Result != "loss") {
			return errors.New("invalid replay participant, team or outcome")
		}
		if results[p.Team] != "" && results[p.Team] != p.Result {
			return errors.New("inconsistent team outcome")
		}
		results[p.Team] = p.Result
		counts[p.Team]++
		return nil
	}
	for _, p := range replay.Players {
		if toons[p.ToonHandle] {
			if seen[p.ToonHandle] {
				return replay.Player{}, errors.New("duplicate replay participant")
			}
			seen[p.ToonHandle] = true
			if err := record(p); err != nil {
				return replay.Player{}, err
			}
			if p.ToonHandle == toon {
				selected = p
			}
			continue
		}
		// The replay is already bound to this exact lobby by byte-identical
		// embedded BattleLobby content. In custom lobbies remaining entries
		// are AI participants: identity is not verifiable, team and outcome
		// still are. Any additional Hero handle means an unaccounted player.
		if !lobby.Custom || IsHeroHandle(p.ToonHandle) {
			return replay.Player{}, errors.New("replay participant is not in the lobby roster")
		}
		if err := record(p); err != nil {
			return replay.Player{}, err
		}
	}
	if len(seen) != len(toons) {
		return replay.Player{}, errors.New("replay and lobby participant counts differ")
	}
	if lobby.Custom {
		if counts[0] == 0 || counts[1] == 0 || results[0] == results[1] {
			return replay.Player{}, errors.New("inconsistent custom-game result")
		}
	} else if counts[0] != 5 || counts[1] != 5 || results[0] == results[1] {
		return replay.Player{}, errors.New("inconsistent 5v5 result")
	}
	if selected.ToonHandle == "" {
		return replay.Player{}, errors.New("selected participant is missing from the replay")
	}
	return selected, nil
}

