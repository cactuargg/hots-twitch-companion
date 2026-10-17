package main

import "hots-twitch-companion/internal/hots/lobby"

// Shared state still uses the same JSON representation and fields.
type LobbySnapshot = lobby.Snapshot

func captureBattleLobby(path string) *LobbySnapshot { return lobby.Capture(path) }
