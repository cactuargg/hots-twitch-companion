package replay

import (
	"errors"
	"hots-twitch-companion/internal/i18n"
	"strings"
)

func Parse(path string) (Replay, error) {
	raw, e := ReadMember(path, "replay.details")
	if e != nil {
		return Replay{}, e
	}
	return ParseDetails(raw)
}

func FindOutcome(r Replay, playerName, toon, lang string) (Player, error) {
	var matches []Player
	for _, p := range r.Players {
		if toon != "" && strings.EqualFold(p.ToonHandle, toon) || toon == "" && strings.EqualFold(p.Name, strings.TrimSpace(playerName)) {
			matches = append(matches, p)
		}
	}
	if len(matches) != 1 {
		return Player{}, errors.New(i18n.Text(lang, "replay.player_matches", len(matches)))
	}
	if matches[0].Result == "unknown" {
		return Player{}, errors.New(i18n.Text(lang, "replay.unknown_outcome"))
	}
	return matches[0], nil
}
