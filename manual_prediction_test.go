package main

import (
 "bytes"

 "context"
 "encoding/json"
 "io"
 "net/http"
 "net/http/httptest"

 "path/filepath"
 "strings"
 "testing"
 "time"
)

type manualTransport func(*http.Request) (*http.Response,error)
func (f manualTransport) RoundTrip(req *http.Request) (*http.Response,error) { return f(req) }
func manualResponse(body string) *http.Response {
 return &http.Response{StatusCode:200,Body:io.NopCloser(strings.NewReader(body)),Header:make(http.Header)}
}
func manualDashboard(t *testing.T,c Config) *dashboard {
 t.Helper()
 dir:=t.TempDir()
 p:=paths{dir:dir,config:filepath.Join(dir,"config.json"),secret:filepath.Join(dir,"tokens.dpapi"),state:filepath.Join(dir,"state.json")}
 d,err:=newDashboard(p,c)
 if err!=nil { t.Fatal(err) }
 return d
}
func manualPost(d *dashboard, id string, started int64, outcome string, authorized bool) *httptest.ResponseRecorder {
 body,_:=json.Marshal(map[string]any{"outcome":outcome,"id":id,"startedAt":started})
 r:=httptest.NewRequest("POST","http://127.0.0.1:12345/secret/api/prediction/manual",bytes.NewReader(body))
 r.Host="127.0.0.1:12345"
 r.Header.Set("Origin","http://127.0.0.1:12345")
 if authorized { r.Header.Set("X-Companion","1") }
 w:=httptest.NewRecorder()
 d.routes("127.0.0.1:12345","secret").ServeHTTP(w,r)
 return w
}
func TestManualSimulationAndStaleProtection(t *testing.T) {
 d:=manualDashboard(t,Config{Lang:"ru",PlayerName:"Tester"})
 started:=time.Now().Add(-25*time.Second).UnixMilli()
 s:=State{Active:&Prediction{ID:"SIMULATION",StartedAt:started,DryRun:true}}
 if err:=d.engine.updateAndSave(s);err!=nil {t.Fatal(err)}
 st:=d.status()
 if st["activePredictionId"]!="SIMULATION"||st["activePredictionStartedAt"]!=started {t.Fatalf("wrong status: %+v",st)}
 if w:=manualPost(d,"SIMULATION",started,"win",false);w.Code!=403 {t.Fatalf("unauthorized request: %d",w.Code)}
 if w:=manualPost(d,"SIMULATION",started-1,"win",true);w.Code==200 {t.Fatal("stale timestamp was accepted")}
 if w:=manualPost(d,"SIMULATION",started,"something",true);w.Code==200 {t.Fatal("invalid outcome accepted")}
 if w:=manualPost(d,"SIMULATION",started,"sync",true);w.Code==200 {t.Fatal("sync must not apply to simulation")}
 if w:=manualPost(d,"SIMULATION",started,"win",true);w.Code!=200 {t.Fatalf("manual win: %d %s",w.Code,w.Body.String())}
 d.engine.mu.Lock()
 after:=d.engine.state
 d.engine.mu.Unlock()
 if after.Active!=nil||!strings.HasPrefix(after.LastResult,"win (manual") {t.Fatalf("not settled: %+v",after)}
 var persisted State
 if err:=loadJSON(d.p.state,&persisted);err!=nil||persisted.Active!=nil {t.Fatalf("not persisted: %+v %v",persisted,err)}
 if w:=manualPost(d,"SIMULATION",started,"loss",true);w.Code==200 {t.Fatal("double manual settlement accepted")}
 if w:=manualPost(d,"SIMULATION",started,"cancel",true);w.Code==200 {t.Fatal("double cancel accepted")}
}

func TestManualRealOutcomesAndRemoteSync(t *testing.T) {
 oldClient:=httpClient
 defer func(){httpClient=oldClient}()
 for _,tc:=range []struct{choice,status,winner,expected string}{
  {"win","RESOLVED","winning","win"},
  {"loss","RESOLVED","losing","loss"},
  {"cancel","CANCELED","","cancel"},
  {"sync","RESOLVED","winning","win"},
  {"sync","CANCELED","","cancel"},
 } {
  t.Run(tc.choice+"-"+tc.status+"-"+tc.winner,func(t *testing.T){
   d:=manualDashboard(t,Config{ClientID:"client",RealMode:true,Lang:"en"})
   started:=time.Now().Add(-20*time.Second).UnixMilli()
   prediction:=&Prediction{ID:"prediction-1",StartedAt:started,BroadcasterID:"channel-1",WinID:"winning",LossID:"losing"}
   if err:=d.engine.updateAndSave(State{Active:prediction});err!=nil {t.Fatal(err)}
   d.engine.twitch.token=Token{AccessToken:"fake-token",ExpiresIn:3600,SavedAt:time.Now().Unix()}
   requests:=0
   httpClient=&http.Client{Timeout:3*time.Second,Transport:manualTransport(func(req *http.Request)(*http.Response,error){
    requests++
    if req.Header.Get("Authorization")!="Bearer fake-token"||req.Header.Get("Client-Id")!="client" {
     t.Errorf("incorrect Twitch auth headers")
    }
    if tc.choice=="sync"{
     if req.Method!="GET"||req.URL.Path!="/helix/predictions" { t.Errorf("sync must read Twitch state: %s %s",req.Method,req.URL.Path) }
     return manualResponse(`{"data":[{"id":"prediction-1","status":"`+tc.status+`","winning_outcome_id":"`+tc.winner+`"}]}`),nil
    }
    if req.Method!="PATCH"||req.URL.Path!="/helix/predictions" {t.Errorf("manual settle must PATCH Twitch: %s %s",req.Method,req.URL.Path)}
    var b map[string]any
    if err:=json.NewDecoder(req.Body).Decode(&b);err!=nil {t.Error(err)}
    if b["status"]!=tc.status {t.Errorf("wrong Twitch status: %v",b)}
    if tc.choice!="cancel"&&b["winning_outcome_id"]!=tc.winner {t.Errorf("wrong winning outcome: %v",b)}
    if tc.choice=="cancel"&&b["winning_outcome_id"]!=nil {t.Errorf("cancel must not select a winner: %v",b)}
    return manualResponse(`{"data":[]}`),nil
   })}
   if w:=manualPost(d,prediction.ID,started,tc.choice,true);w.Code!=200 {t.Fatalf("manual API returned %d: %s",w.Code,w.Body.String())}
   if requests!=1 {t.Fatalf("unexpected Twitch calls: %d",requests)}
   d.engine.mu.Lock()
   state:=d.engine.state
   d.engine.mu.Unlock()
   if state.Active!=nil||!strings.HasPrefix(state.LastResult,tc.expected+" (") {t.Fatalf("wrong outcome: %+v",state)}
  })
 }
}

func TestManualSyncNeverClearsOpenRemotePrediction(t *testing.T) {
 old:=httpClient
 defer func(){httpClient=old}()
 d:=manualDashboard(t,Config{ClientID:"client",RealMode:true})
 p:=&Prediction{ID:"pending",BroadcasterID:"channel",StartedAt:time.Now().UnixMilli()}
 if err:=d.engine.updateAndSave(State{Active:p});err!=nil {t.Fatal(err)}
 d.engine.twitch.token=Token{AccessToken:"fake",ExpiresIn:3600,SavedAt:time.Now().Unix()}
 httpClient=&http.Client{Transport:manualTransport(func(req *http.Request)(*http.Response,error){
  return manualResponse(`{"data":[{"id":"pending","status":"ACTIVE"}]}`),nil
 })}
 if w:=manualPost(d,p.ID,p.StartedAt,"sync",true);w.Code==200 {t.Fatal("remote ACTIVE must not be discarded")}
 d.engine.mu.Lock();remaining:=d.engine.state.Active;d.engine.mu.Unlock()
 if remaining==nil {t.Fatal("lost open real Prediction")}
}

func TestScanTickResolvesShortGameWithout75SecondDelay(t *testing.T) {
	now := time.Now()
	e, p, lobby := boundScanFixture(t, now.Add(-30*time.Second))
	writeCorrelationFile(t, p.dir, "Fast", correlationMPQTest(correlationDetailsTest(t, lobby, now.Add(-30*time.Second), true), lobby), now.Add(-10*time.Second))
	lastError := ""
	lastLog := time.Time{}
	e.scanTick(context.Background(), now.Add(-35*time.Second), &lastError, &lastLog)
	if e.state.Active != nil || !strings.HasPrefix(e.state.LastResult, "win (") {
		t.Fatalf("short game did not settle: %+v %+v", e.state, e.scan)
	}
}
