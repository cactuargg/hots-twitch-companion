package correlation

import (
 "errors"
 "os"
 "path/filepath"
 "testing"
 "time"
 "hots-twitch-companion/internal/hots/discovery"
)

func TestReplayInventoryRejectsChangesAndNonReplayFiles(t *testing.T){
 dir:=t.TempDir()
 when:=time.Now().Add(-time.Minute).Truncate(time.Second)
 file:=filepath.Join(dir,"match.StormReplay")
 raw:=make([]byte,64)
 if err:=os.WriteFile(file,raw,0600);err!=nil{t.Fatal(err)}
 if err:=os.Chtimes(file,when,when);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"ignored.txt"),raw,0600);err!=nil{t.Fatal(err)}
 files,err:=CorrelationReplayFiles(dir,when.Add(-time.Second))
 if err!=nil||len(files)!=1||files[0].Path!=file{t.Fatalf("wrong candidates %+v %v",files,err)}
 decoded,err:=ReadStableReplay(files[0])
 if err!=nil||len(decoded)!=64{t.Fatalf("stable replay rejected: %v",err)}
 if err:=os.WriteFile(file,append(raw,1),0600);err!=nil{t.Fatal(err)}
 if _,err:=ReadStableReplay(files[0]);err==nil{t.Fatal("modified replay treated as original")}
}

func TestReplayInventoryFailureDoesNotAllowPartialCorrelation(t *testing.T){
 dir:=t.TempDir()
 missing:=filepath.Join(dir,"missing")
 if _,err:=CorrelationReplayFiles(missing,time.Time{});!errors.Is(err,os.ErrNotExist){t.Fatalf("missing explicit replay directory tolerated: %v",err)}
 if _,err:=ReadStableReplay(discovery.ReplayFile{Path:missing});err==nil{t.Fatal("missing replay accepted")}
}
