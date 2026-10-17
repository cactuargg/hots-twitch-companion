package config

import (
 "encoding/json"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestLegacyJSONRoundtripAndForcedRealMode(t *testing.T) {
 p:=NewPaths(t.TempDir())
 before:=`{"clientId":"old-app","playerName":"Me","playerToonHandle":"2-Hero-1-44","replayRoot":"D:/Replays","realMode":false,"lang":"ru","predictionWindowMin":31,"wizardDone":true,"predictionTitle":"Custom","predictionWin":"Victory","predictionLoss":"Defeat"}`
 if err:=os.WriteFile(p.Config,[]byte(before),0600);err!=nil{t.Fatal(err)}
 cfg,err:=Read(p)
 if err!=nil {t.Fatal(err)}
 if !cfg.RealMode || cfg.ClientID!="old-app" || cfg.PlayerToonHandle!="2-Hero-1-44" ||
 cfg.ReplayRoot!="D:/Replays" || !cfg.WizardDone || cfg.PredictionWindowMinutes()!=30 {
  t.Fatalf("legacy config changed: %+v",cfg)
 }
 title,win,loss:=cfg.PredictionTexts()
 if title!="Custom" || win!="Victory" || loss!="Defeat" {t.Fatalf("overrides changed: %s %s %s",title,win,loss)}
 if err:=Write(p,cfg);err!=nil{t.Fatal(err)}
 raw,err:=os.ReadFile(p.Config);if err!=nil{t.Fatal(err)}
 var fields map[string]json.RawMessage
 if err:=json.Unmarshal(raw,&fields);err!=nil{t.Fatal(err)}
 for _,name:=range []string{"clientId","playerName","playerToonHandle","replayRoot","realMode","lang","predictionWindowMin","wizardDone","predictionTitle","predictionWin","predictionLoss"}{
  if _,ok:=fields[name];!ok{t.Fatalf("lost JSON field %q: %s",name,raw)}
 }
}

func TestConfigDefaultsAndPaths(t *testing.T){
 c:=Data{Lang:"xx"}
 if c.Language()!="en" || c.PredictionWindowMinutes()!=2 || c.TwitchClientID(" default ")!="default"{t.Fatalf("invalid defaults: %+v",c)}
 if title,win,loss:=c.PredictionTexts();title==""||win==""||loss==""{t.Fatal("localized prediction defaults missing")}
 c.ClientID=" personal "
 if c.TwitchClientID("default")!="personal"{t.Fatal("explicit Client ID overwritten")}
 dir:=t.TempDir();p:=NewPaths(dir)
 if p.Dir!=dir || p.Config!=filepath.Join(dir,"config.json") || p.State!=filepath.Join(dir,"state.json") || p.Secret!=filepath.Join(dir,"tokens.dpapi"){t.Fatalf("profile path changed: %+v",p)}
 t.Setenv("LOCALAPPDATA",dir)
 got,err:=PathsForUser();if err!=nil{t.Fatal(err)}
 if !strings.HasPrefix(got.Dir,dir) || got.Dir!=filepath.Join(dir,"HotSTwitchCompanion"){t.Fatalf("unexpected user profile: %+v",got)}
}

func TestConfigWriteFailureRetainsCommittedFile(t *testing.T){
 p:=NewPaths(t.TempDir())
 if err:=Write(p,Data{ClientID:"original",RealMode:true});err!=nil{t.Fatal(err)}
 fail:=p;fail.Config=filepath.Join(p.Dir,"missing","config.json")
 if err:=Write(fail,Data{ClientID:"replacement"});err==nil{t.Fatal("expected failed write")}
 got,err:=Read(p)
 if err!=nil||got.ClientID!="original"{t.Fatalf("previous config overwritten on failure: %+v %v",got,err)}
}
