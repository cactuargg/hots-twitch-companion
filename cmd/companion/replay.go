package main

// Read-only MPQ and Blizzard versioned-data parser, based on public file formats.
// Reads replay.details and replay.server.battlelobby for correlation.
// No game-process access or file modifications.
import (
	replay "hots-twitch-companion/internal/hots/replay"
	"errors"
	"strings"
)

const maxReplay = replay.MaxReplay

func readMember(path,name string) ([]byte,error) { return replay.ReadMember(path,name) }
func readMPQMember(data []byte,name string) ([]byte,error) { return replay.ReadMPQMember(data,name) }
func unpackMPQ(data []byte) ([]byte,error) { return replay.UnpackCompressed(data) }

// Domain records and versioned-data decoding live in internal/hots/replay.
type Player = replay.Player
type Replay = replay.Replay

func parseReplay(path string) (Replay, error) {
	raw, e := readMember(path, "replay.details")
	if e != nil {
		return Replay{}, e
	}
	return parseReplayDetails(raw)
}


func parseReplayDetails(raw []byte) (Replay,error) { return replay.ParseDetails(raw) }

func findOutcome(r Replay, playerName, toon, lang string) (Player, error) {
	var matches []Player
	for _, p := range r.Players {
		if toon != "" && strings.EqualFold(p.ToonHandle, toon) || toon == "" && strings.EqualFold(p.Name, strings.TrimSpace(playerName)) {
			matches = append(matches, p)
		}
	}
	if len(matches) != 1 {
		return Player{}, errors.New(tr(lang, "replay.player_matches", len(matches)))
	}
	if matches[0].Result == "unknown" {
		return Player{}, errors.New(tr(lang, "replay.unknown_outcome"))
	}
	return matches[0], nil
}
