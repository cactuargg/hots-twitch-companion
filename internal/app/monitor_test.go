package app

import (
	"hots-twitch-companion/internal/config"
	predmodel "hots-twitch-companion/internal/prediction"
	"testing"
)

func TestMonitorStartBlockReason(t *testing.T) {
	base := config.Data{PlayerName: "player", RealMode: true}
	tests := []struct {
		name    string
		running bool
		cfg     config.Data
		client  bool
		state   predmodel.State
		want    string
	}{
		{"allowed", false, base, true, predmodel.State{}, ""},
		{"already running", true, base, true, predmodel.State{}, "automation.already_running"},
		{"no player", false, config.Data{RealMode: true}, true, predmodel.State{}, "automation.need_player"},
		{"handle only", false, config.Data{PlayerToonHandle: "handle"}, false, predmodel.State{}, ""},
		{"missing OAuth", false, base, false, predmodel.State{}, "automation.need_client_id"},
		{"simulation without OAuth", false, config.Data{PlayerName: "player"}, false, predmodel.State{}, ""},
		{"mode conflict", false, base, true, predmodel.State{Active: &predmodel.Prediction{DryRun: true}}, "automation.mode_conflict"},
		{"same mode", false, base, true, predmodel.State{Active: &predmodel.Prediction{DryRun: false}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MonitorStartBlockReason(tt.running, tt.cfg, tt.client, tt.state); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
