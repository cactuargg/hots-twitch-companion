// Package app owns the Prediction lifecycle, monitoring and recovery.
// Engine coordinates injected game, API and storage dependencies.
package app

import (
	"hots-twitch-companion/internal/config"
)

// LiveConfigChangeAllowed reports whether a running monitor can safely
// accept the new configuration without rebuilding its dependencies.
// Presentation and future prediction settings may change during monitoring.
// Identity, replay source, OAuth application and mode remain fixed.
func LiveConfigChangeAllowed(current, next config.Data) bool {
	return current.ClientID == next.ClientID &&
		current.PlayerName == next.PlayerName &&
		current.PlayerToonHandle == next.PlayerToonHandle &&
		current.ReplayRoot == next.ReplayRoot &&
		current.RealMode == next.RealMode
}
