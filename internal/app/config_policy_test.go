package app

import (
	"hots-twitch-companion/internal/config"
	"testing"
)

func TestLiveConfigChangeAllowed(t *testing.T) {
	base := config.Data{ClientID: "client", PlayerName: "player", ReplayRoot: "replays", Lang: "en", PredictionTitle: "Match"}
	tests := []struct {
		name string
		edit func(*config.Data)
		want bool
	}{
		{"unchanged", func(*config.Data) {}, true},
		{"language", func(c *config.Data) { c.Lang = "ru" }, true},
		{"wizard", func(c *config.Data) { c.WizardDone = true }, true},
		{"player", func(c *config.Data) { c.PlayerName = "other" }, false},
		{"oauth", func(c *config.Data) { c.ClientID = "other" }, false},
		{"replay", func(c *config.Data) { c.ReplayRoot = "other" }, false},
		{"prediction", func(c *config.Data) { c.PredictionTitle = "other" }, true},
		{"duration", func(c *config.Data) { c.PredictionWindowMin = 30 }, true},
		{"outcomes", func(c *config.Data) { c.PredictionWin = "Yes"; c.PredictionLoss = "No" }, true},
		{"toon", func(c *config.Data) { c.PlayerToonHandle = "other" }, false},
		{"mode", func(c *config.Data) { c.RealMode = !c.RealMode }, false},
		{"language and player", func(c *config.Data) { c.Lang = "ru"; c.PlayerName = "other" }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := base
			tt.edit(&next)
			if got := LiveConfigChangeAllowed(base, next); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
