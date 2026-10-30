package storage

import (
 "errors"
 "os"
 "path/filepath"
 "testing"
 "hots-twitch-companion/internal/prediction"
)

func TestResetRemovesOnlyCompanionData(t *testing.T) {
 dir:=t.TempDir()
 config:=filepath.Join(dir,"config.json")
 state:=filepath.Join(dir,"state.json")
 token:=filepath.Join(dir,"tokens.dpapi")
 marker:=filepath.Join(dir,ResetMarker)
 for _,name:=range []string{config,state,token,marker,filepath.Join(dir,"replay-statistics.json"),filepath.Join(dir,"replay-statistics.json.new"),config+".new",state+".new",token+".new"}{
  if err:=os.WriteFile(name,[]byte("secret"),0600);err!=nil{t.Fatal(err)}
 }
 keep:=filepath.Join(dir,"unrelated.txt")
 if err:=os.WriteFile(keep,[]byte("keep"),0600);err!=nil{t.Fatal(err)}
 webview:=filepath.Join(dir,"webview2")
 if err:=os.MkdirAll(webview,0700);err!=nil{t.Fatal(err)}
 if err:=ApplyProfileReset(dir,config,state,token);err!=nil{t.Fatal(err)}
 for _,file:=range []string{config,state,token,marker,filepath.Join(dir,"replay-statistics.json"),filepath.Join(dir,"replay-statistics.json.new"),config+".new",state+".new",token+".new"}{
  if _,err:=os.Stat(file);!errors.Is(err,os.ErrNotExist){t.Fatalf("profile data survived: %s %v",file,err)}
 }
 if _,err:=os.Stat(webview);!errors.Is(err,os.ErrNotExist){t.Fatal("WebView data survived reset")}
 if _,err:=os.Stat(keep);err!=nil{t.Fatal("reset deleted unrelated file",err)}
}

func TestResetFailureRetainsMarker(t *testing.T) {
 dir:=t.TempDir()
 state:=filepath.Join(dir,"state.json")
 marker:=filepath.Join(dir,ResetMarker)
 if err:=os.WriteFile(marker,[]byte("true"),0600);err!=nil{t.Fatal(err)}
 // A nonempty directory cannot be removed as a file on any supported OS;
 // this exercises a partial reset without permissions-dependent behavior.
 if err:=os.Mkdir(state,0700);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(state,"blocker"),[]byte("keep"),0600);err!=nil{t.Fatal(err)}
 if err:=ApplyProfileReset(dir,filepath.Join(dir,"config.json"),state,filepath.Join(dir,"token"));err==nil{t.Fatal("expected removal error")}
 if _,err:=os.Stat(marker);err!=nil{t.Fatal("failed reset lost durable marker",err)}
}

func TestLegacySimulationMigrationPreservesRealEvidence(t *testing.T) {
 file:=filepath.Join(t.TempDir(),"state.json")
 s:=prediction.State{
  Active:&prediction.Prediction{ID:"real-id",DryRun:false},
  UnconfirmedCreation:&prediction.CreationAttempt{LobbySHA:"pending",BroadcasterID:"channel"},
  LastCreatedLobbySHA:"handled",
 }
 if err:=SaveJSON(file,s);err!=nil{t.Fatal(err)}
 before,err:=os.ReadFile(file);if err!=nil{t.Fatal(err)}
 if err:=MigrateLegacySimulation(file);err!=nil{t.Fatal(err)}
 after,err:=os.ReadFile(file);if err!=nil{t.Fatal(err)}
 if string(after)!=string(before){t.Fatal("migration rewrote live wager state")}
 s.Active.DryRun=true
 s.UnconfirmedCreation=nil
 if err:=SaveJSON(file,s);err!=nil{t.Fatal(err)}
 if err:=MigrateLegacySimulation(file);err!=nil{t.Fatal(err)}
 var got prediction.State
 if err:=LoadJSON(file,&got);err!=nil{t.Fatal(err)}
 if got.Active!=nil || got.LastCreatedLobbySHA!="handled" {t.Fatalf("legacy migration discarded unrelated state: %+v",got)}
}
