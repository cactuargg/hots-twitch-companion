package main

import "hots-twitch-companion/internal/hots/lobby"

// Compatibility aliases keep serialized state and existing call sites intact.
// The parser implementation and its bit reader live in internal/hots/lobby.
type LobbyPlayer = lobby.LobbyPlayer
type BattleLobby = lobby.BattleLobby

const maxLobbyBytes = lobby.MaxBytes

func parseBattleLobby(raw []byte) (BattleLobby, error) {
	return lobby.Parse(raw)
}
