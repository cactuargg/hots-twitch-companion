package app

import (
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/prediction"
)

// UnconfirmedCreationBlock is not localized in the existing desktop flow.
const UnconfirmedCreationBlock = "creation.unconfirmed"

// MonitorStartBlockReason returns the first failed precondition in the same
// order as the original Engine start implementation. No state is mutated.
func MonitorStartBlockReason(running bool, cfg config.Data, hasTwitchClientID bool, state prediction.State) string {
	if running {
		return "automation.already_running"
	}
	if state.UnconfirmedCreation != nil {
		return UnconfirmedCreationBlock
	}
	if cfg.PlayerName == "" && cfg.PlayerToonHandle == "" {
		return "automation.need_player"
	}
	if !hasTwitchClientID && cfg.RealMode {
		return "automation.need_client_id"
	}
	if state.Active != nil && state.Active.DryRun == cfg.RealMode {
		return "automation.mode_conflict"
	}
	return ""
}
