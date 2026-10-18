// Package config owns the persistent Companion settings and profile paths.
// Application-specific linker defaults stay in the Windows command package.
package config

import (
 "os"
 "path/filepath"
 "strings"
 "hots-twitch-companion/internal/i18n"
 "hots-twitch-companion/internal/storage"
)

type Data struct {
	DisableUpdateChecks bool `json:"disableUpdateChecks"`
	ClientID            string `json:"clientId"`
	PlayerName          string `json:"playerName"`
	PlayerToonHandle    string `json:"playerToonHandle"`
	ReplayRoot          string `json:"replayRoot"`
	RealMode            bool   `json:"realMode"`
	Lang                string `json:"lang"`
	PredictionWindowMin int    `json:"predictionWindowMin"`
	WizardDone          bool   `json:"wizardDone"`
	PredictionTitle     string `json:"predictionTitle"`
	PredictionWin       string `json:"predictionWin"`
	PredictionLoss      string `json:"predictionLoss"`
}


// TwitchClientID resolves an explicitly selected OAuth client or a release's
// publicly embedded Client ID. Never pass a Client Secret here.
func (c Data) TwitchClientID(defaultID string) string {
 if id := strings.TrimSpace(c.ClientID); id != "" { return id }
 return strings.TrimSpace(defaultID)
}

const DefaultPredictionWindow = 2
const MaxPredictionWindow = 30

func (c Data) Language() string { return i18n.Normalize(c.Lang) }

func (c Data) PredictionWindowMinutes() int {
 if c.PredictionWindowMin < 1 { return DefaultPredictionWindow }
 if c.PredictionWindowMin > MaxPredictionWindow { return MaxPredictionWindow }
 return c.PredictionWindowMin
}

func (c Data) PredictionTexts() (title, win, loss string) {
 title = strings.TrimSpace(c.PredictionTitle)
 win = strings.TrimSpace(c.PredictionWin)
 loss = strings.TrimSpace(c.PredictionLoss)
 if title == "" {title = i18n.Text(c.Language(), "prediction.title")}
 if win == "" {win = i18n.Text(c.Language(), "prediction.win")}
 if loss == "" {loss = i18n.Text(c.Language(), "prediction.loss")}
 return title,win,loss
}

// Paths identifies files within an existing or separately selected profile.
// It does not create directories or modify files.
type Paths struct { Dir, Config, Secret, State string }

func NewPaths(dir string) Paths {
 return Paths{Dir:dir,Config:filepath.Join(dir,"config.json"),Secret:filepath.Join(dir,"tokens.dpapi"),State:filepath.Join(dir,"state.json")}
}

func PathsForUser() (Paths,error) {
 home:=os.Getenv("LOCALAPPDATA")
 if home=="" {
  var err error
  home,err=os.UserConfigDir()
  if err!=nil {return Paths{},err}
 }
 return NewPaths(filepath.Join(home,"HotSTwitchCompanion")),nil
}

func Read(p Paths) (Data,error) {
 var c Data
 err:=storage.LoadJSON(p.Config,&c)
 // Preserve desktop migration: real Twitch mode is enforced even if the
 // legacy persisted profile contains realMode=false.
 c.RealMode=true
 return c,err
}

func Write(p Paths,c Data) error {
 return storage.SaveJSON(p.Config,c)
}
