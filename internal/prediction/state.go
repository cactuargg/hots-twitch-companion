// Package prediction holds durable state types and pure transitions for
// Twitch Predictions. It never imports the application, HTTP, or storage.
package prediction

import "hots-twitch-companion/internal/hots/lobby"

type MatchBinding struct {
	Version    int             `json:"version"`
	Lobby      *lobby.Snapshot `json:"lobby"`
	LobbyKey   string          `json:"lobbyKey"`
	PlayerToon string          `json:"playerToon"`
	Ambiguous  bool            `json:"ambiguous,omitempty"`
}

type SettlementAttempt struct {
	Outcome  string `json:"outcome"`
	Source   string `json:"source"`
	Failures int    `json:"failures,omitempty"`
	RetryAt  int64  `json:"retryAt,omitempty"`
}

type Prediction struct {
	ID            string             `json:"id"`
	BroadcasterID string             `json:"broadcasterId"`
	WinID         string             `json:"winId"`
	LossID        string             `json:"lossId"`
	StartedAt     int64              `json:"startedAt"`
	WindowSeconds int                `json:"windowSeconds,omitempty"`
	DryRun        bool               `json:"dryRun"`
	Match         *MatchBinding      `json:"match,omitempty"`
	Settlement    *SettlementAttempt `json:"settlement,omitempty"`
}

type CreationAttempt struct {
	LobbySHA      string `json:"lobbySHA"`
	BroadcasterID string `json:"broadcasterId"`
	AttemptedAt   int64  `json:"attemptedAt"`
}

type LobbyEvent struct {
	Key      string          `json:"key"`
	Snapshot *lobby.Snapshot `json:"snapshot"`
}

type State struct {
	UnconfirmedCreation *CreationAttempt `json:"unconfirmedCreation,omitempty"`
	Active              *Prediction      `json:"active"`
	LastLobby           *lobby.Snapshot  `json:"lastLobby,omitempty"`
	LastLobbyKey        string           `json:"lastLobbyKey"`
	LastSettlementAt    int64            `json:"lastSettlementAt"`
	LastResult          string           `json:"lastResult"`
	PendingLobby        *LobbyEvent      `json:"pendingLobby,omitempty"`
	PendingLobbies      []*LobbyEvent    `json:"pendingLobbies,omitempty"`
	LastCreatedLobbySHA string           `json:"lastCreatedLobbySHA,omitempty"`
	RecentLobbySHAs     []string         `json:"recentLobbySHAs,omitempty"`
}
