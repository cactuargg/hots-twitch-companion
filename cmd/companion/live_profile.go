package main

import "hots-twitch-companion/internal/storage"

// Compatibility adapter while Dashboard bootstrapping still lives in main.
func migrateLiveProfile(p paths,cfg Config) error {
 if err:=storage.MigrateLegacySimulation(p.state);err!=nil{return err}
 return writeConfig(p,cfg)
}
