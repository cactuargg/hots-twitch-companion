package app

import (
	"bytes"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/prediction"
	"os"
	"testing"
)

func TestGameProfileSwitchBlockedAndPreservesSharedState(t *testing.T) {
	for _, mode := range []string{"running", "active", "uncertain", "stopped"} {
		t.Run(mode, func(t *testing.T) {
			e, p := liveLifecycleTest(t)
			original := e.Config()
			// Ensure a durable initial config exists to check failed switches never write it.
			if err := config.Write(p, original); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(p.Config)
			next := original.Clone()
			next.GameProfiles = append(next.GameProfiles, config.GameProfile{ID: "second", Name: "Second", PlayerName: "Alt", PlayerToonHandle: "2-Hero-1-900", ReplayRoot: "E:/Replays"})
			next.ActiveGameProfile = "second"
			next.PlayerName = "Alt"
			next.PlayerToonHandle = "2-Hero-1-900"
			next.ReplayRoot = "E:/Replays"
			switch mode {
			case "running":
				e.state.Active = nil
				e.running = true
			case "uncertain":
				e.state.Active = nil
				e.state.UnconfirmedCreation = &prediction.CreationAttempt{BroadcasterID: "channel"}
			case "stopped":
				e.state.Active = nil
				e.state.PendingLobby = &prediction.LobbyEvent{Key: "old-player"}
			}
			tokenBefore, _ := os.ReadFile(p.Secret)
			err := e.UpdateConfig(next)
			if mode != "stopped" {
				after, _ := os.ReadFile(p.Config)
				if err == nil || !bytes.Equal(before, after) || e.Config().PlayerToonHandle != original.PlayerToonHandle {
					t.Fatalf("unsafe switch: %+v %v", e.Config(), err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			restored, err := config.Read(p)
			if err != nil || restored.ActiveGameProfile != "second" || restored.ClientID != original.ClientID || restored.PredictionTitle != original.PredictionTitle || e.state.PendingLobby != nil {
				t.Fatalf("switch state: %+v %v", restored, err)
			}
			tokenAfter, _ := os.ReadFile(p.Secret)
			if !bytes.Equal(tokenBefore, tokenAfter) {
				t.Fatal("Twitch credentials changed")
			}
			snapshot := e.Config()
			snapshot.GameProfiles[0].Name = "mutated"
			if e.Config().GameProfiles[0].Name == "mutated" {
				t.Fatal("config snapshot aliases live profiles")
			}
			// Existing clients may save general settings without the new profile fields.
			restored.GameProfiles = nil
			restored.ActiveGameProfile = ""
			restored.Lang = "ru"
			if err := e.UpdateConfig(restored); err != nil {
				t.Fatal(err)
			}
			if len(e.Config().GameProfiles) != 2 || e.Config().ActiveGameProfile != "second" {
				t.Fatal("legacy config save dropped profiles")
			}
		})
	}
}
