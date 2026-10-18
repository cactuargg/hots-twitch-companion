package app

import (
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/prediction"
	"testing"
)

func TestUpdateCannotInterruptPredictionOrMonitoring(t *testing.T) {
	for _, kind := range []string{"running", "active", "uncertain"} {
		t.Run(kind, func(t *testing.T) {
			d := manualDashboard(t, config.Data{RealMode: true})
			e := d.engine
			switch kind {
			case "running":
				e.running = true
			case "active":
				e.state.Active = &prediction.Prediction{ID: "active"}
			case "uncertain":
				e.state.UnconfirmedCreation = &prediction.CreationAttempt{LobbySHA: "uncertain"}
			}
			if err := e.BeginUpdate(func() error { t.Fatal("installer invoked before safe idle"); return nil }); err == nil {
				t.Fatal("unsafe installation allowed")
			}
		})
	}
	d := manualDashboard(t, config.Data{PlayerName: "demo", RealMode: true})
	e := d.engine
	if err := e.BeginUpdate(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := e.Start(); err == nil {
		e.StopWatch()
		t.Fatal("monitor started after update hand-off")
	}
}
