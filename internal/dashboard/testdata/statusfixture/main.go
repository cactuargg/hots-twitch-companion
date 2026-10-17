// Emits a real application status for the Node contract test. No background
// services, user profiles or Twitch requests are used.
package main

import (
	"encoding/json"
	"hots-twitch-companion/internal/app"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/dashboard"
	"hots-twitch-companion/internal/prediction"
	"os"
	"time"
)

type stateStore struct{}

func (stateStore) Load() (prediction.State, error) {
	return prediction.State{Active: &prediction.Prediction{ID: "fixture", StartedAt: time.Now().Add(-30 * time.Second).UnixMilli(), WindowSeconds: 120}}, nil
}
func (stateStore) Save(prediction.State) error { panic("read-only contract fixture") }
func main() {
	e, err := app.New(config.Paths{}, config.Data{RealMode: true, PredictionWindowMin: 30}, app.Options{Store: stateStore{}})
	if err != nil {
		panic(err)
	}
	d := dashboard.New(e, config.Paths{}, "fixture")
	if err := json.NewEncoder(os.Stdout).Encode(d.Status()); err != nil {
		panic(err)
	}
}
