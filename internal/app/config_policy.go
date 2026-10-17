// Package app contains application orchestration policies that can be
// tested independently before moving the Engine and its dependencies here.
package app

import "hots-twitch-companion/internal/config"

// LiveConfigChangeAllowed reports whether a running monitor can safely
// accept the new configuration without rebuilding its dependencies.
// Only language or wizard completion may change during active monitoring.
func LiveConfigChangeAllowed(current, next config.Data) bool {
 if current.Language() == next.Language() && current.WizardDone == next.WizardDone {
  return false
 }
 return current.ClientID == next.ClientID &&
  current.PlayerName == next.PlayerName &&
  current.PlayerToonHandle == next.PlayerToonHandle &&
  current.ReplayRoot == next.ReplayRoot &&
  current.RealMode == next.RealMode &&
  current.PredictionWindowMin == next.PredictionWindowMin &&
  current.PredictionTitle == next.PredictionTitle &&
  current.PredictionWin == next.PredictionWin &&
  current.PredictionLoss == next.PredictionLoss
}
