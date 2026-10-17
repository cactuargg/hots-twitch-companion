package storage

import (
 "errors"
 "os"
 "path/filepath"
 "testing"
 "hots-twitch-companion/internal/prediction"
)

func TestFileStateStoreRoundTripAndMissingFile(t *testing.T) {
 path:=filepath.Join(t.TempDir(),"state.json")
 store:=FileStateStore{Path:path}
 empty,err:=store.Load()
 if err!=nil||empty.Active!=nil{t.Fatalf("missing state was not empty: %+v %v",empty,err)}
 state:=prediction.State{Active:&prediction.Prediction{ID:"remote-active",BroadcasterID:"owner"},LastCreatedLobbySHA:"handled"}
 if err:=store.Save(state);err!=nil{t.Fatal(err)}
 loaded,err:=store.Load()
 if err!=nil||loaded.Active==nil||loaded.Active.ID!="remote-active"||loaded.LastCreatedLobbySHA!="handled"{t.Fatalf("durable state changed: %+v %v",loaded,err)}
 if _,err:=os.Stat(path+".new");!errors.Is(err,os.ErrNotExist){t.Fatalf("temporary state file remains: %v",err)}
}

func TestFileStateStoreFailedSaveRetainsPreviousPrediction(t *testing.T) {
 dir:=t.TempDir()
 path:=filepath.Join(dir,"state.json")
 store:=FileStateStore{Path:path}
 if err:=store.Save(prediction.State{Active:&prediction.Prediction{ID:"previous"}});err!=nil{t.Fatal(err)}
 broken:=FileStateStore{Path:filepath.Join(dir,"missing","state.json")}
 if err:=broken.Save(prediction.State{});err==nil{t.Fatal("unexpected write success")}
 previous,err:=store.Load()
 if err!=nil||previous.Active==nil||previous.Active.ID!="previous"{t.Fatalf("write error damaged the wager: %+v %v",previous,err)}
}
