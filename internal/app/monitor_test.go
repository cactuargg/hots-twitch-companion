package app

import (
 "testing"
 "hots-twitch-companion/internal/config"
 "hots-twitch-companion/internal/prediction"
)

func TestMonitorStartBlockReason(t *testing.T) {
 base:=config.Data{PlayerName:"player",RealMode:true}
 tests:=[]struct{name string; running bool; cfg config.Data; client bool; state prediction.State; want string}{
  {"allowed",false,base,true,prediction.State{},""},
  {"already running",true,base,true,prediction.State{},"automation.already_running"},
  {"no player",false,config.Data{RealMode:true},true,prediction.State{},"automation.need_player"},
  {"handle only",false,config.Data{PlayerToonHandle:"handle"},false,prediction.State{},""},
  {"missing OAuth",false,base,false,prediction.State{},"automation.need_client_id"},
  {"simulation without OAuth",false,config.Data{PlayerName:"player"},false,prediction.State{},""},
  {"mode conflict",false,base,true,prediction.State{Active:&prediction.Prediction{DryRun:true}},"automation.mode_conflict"},
  {"same mode",false,base,true,prediction.State{Active:&prediction.Prediction{DryRun:false}},""},
 }
 for _,tt:=range tests {t.Run(tt.name,func(t *testing.T){if got:=MonitorStartBlockReason(tt.running,tt.cfg,tt.client,tt.state);got!=tt.want{t.Fatalf("got %q want %q",got,tt.want)}})}
}
