package storage

import (
  "errors"
  "os"
  "path/filepath"
  "hots-twitch-companion/internal/prediction"
)

// ResetMarker schedules deletion at next process startup, after taking
// the exclusive profile lock and before loading any credentials or WebView.
const ResetMarker = ".reset-profile"

// ApplyProfileReset removes only files owned by Companion. On failure the
// marker stays in place so the next startup can retry safely.
func ApplyProfileReset(dir, configFile, stateFile, tokenFile string) error {
 marker:=filepath.Join(dir,ResetMarker)
 if _,err:=os.Stat(marker);errors.Is(err,os.ErrNotExist){return nil}else if err!=nil{return err}
 for _,name:=range []string{configFile,stateFile,tokenFile,configFile+".new",stateFile+".new",tokenFile+".new",marker+".new"}{
  if err:=os.Remove(name);err!=nil && !errors.Is(err,os.ErrNotExist){return err}
 }
 if err:=os.RemoveAll(filepath.Join(dir,"webview2"));err!=nil{return err}
 return os.Remove(marker)
}

// MigrateLegacySimulation preserves every real Prediction, consumed lobby,
// settlement intent, and uncertain remote creation while dropping only a
// pre-existing local simulation. It intentionally does not rewrite state when
// no simulated Prediction exists.
func MigrateLegacySimulation(stateFile string) error {
 var state prediction.State
 if err:=LoadJSON(stateFile,&state);err!=nil{return err}
 if state.Active!=nil && state.Active.DryRun {
  state.Active=nil
  state.PendingLobby=nil
  return SaveJSON(stateFile,state)
 }
 return nil
}
