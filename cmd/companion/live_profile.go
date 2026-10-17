package main

// Only a legacy local simulation can be discarded. Real Predictions, consumed
// lobbies and settlement recovery evidence must survive this UI migration.
func migrateLiveProfile(p paths, cfg Config) error {
	s, err := loadState(p)
	if err != nil {
		return err
	}
	if s.Active != nil && s.Active.DryRun {
		s.Active = nil
		s.PendingLobby = nil
		if err := saveState(p, s); err != nil {
			return err
		}
	}
	return writeConfig(p, cfg)
}
