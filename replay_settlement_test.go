package main

import (
 "bytes"
 "compress/zlib"
 "context"
 "encoding/json"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

func writeScanReplay(t *testing.T, root, name string, mod time.Time, valid bool) {
 t.Helper()
 data:=bytes.Repeat([]byte("not-a-valid-replay"),20)
 if valid {
  details:=detailsTest()
  var z bytes.Buffer
  zw:=zlib.NewWriter(&z)
  if _,err:=zw.Write(details);err!=nil {t.Fatal(err)}
  if err:=zw.Close();err!=nil {t.Fatal(err)}
  data=buildTestMPQ(append([]byte{0x02},z.Bytes()...),details)
 }
 file:=filepath.Join(root,name+".StormReplay")
 if err:=os.WriteFile(file,data,0600);err!=nil {t.Fatal(err)}
 if err:=os.Chtimes(file,mod,mod);err!=nil {t.Fatal(err)}
}
func scanFixture(t *testing.T,activeAt time.Time) (*Engine,paths) {
 t.Helper()
 root:=t.TempDir()
 p:=paths{dir:root,config:filepath.Join(root,"config.json"),state:filepath.Join(root,"state.json"),secret:filepath.Join(root,"tokens.dpapi")}
 e,err:=newEngine(p,Config{PlayerName:"Проверка",ReplayRoot:root,Lang:"ru"},func(string){})
 if err!=nil {t.Fatal(err)}
 if err:=e.updateAndSave(State{Active:&Prediction{ID:"SIMULATION",DryRun:true,StartedAt:activeAt.UnixMilli()}});err!=nil {t.Fatal(err)}
 return e,p
}
func TestScanResolvesMatchingReplayEvenAfterElevenNewerUnrelatedFiles(t *testing.T) {
	now := time.Now()
	e, p, lobby := boundScanFixture(t, now.Add(-90*time.Second))
	// The bound replay is older than History's twelve newest files.
	writeCorrelationFile(t, p.dir, "01-win", correlationMPQTest(correlationDetailsTest(t, lobby, now.Add(-90*time.Second), true), lobby), now.Add(-40*time.Second))
	for i := 0; i < 13; i++ {
		name := "distractor-" + string(rune('a'+i))
		writeCorrelationFile(t, p.dir, name, correlationMPQTest(detailsTest(), realARAMLobbyTest(t)), now.Add(time.Duration(-30+i)*time.Second))
	}
	lastError := ""
	lastLog := time.Time{}
	e.scanTick(context.Background(), now.Add(-120*time.Second), &lastError, &lastLog)
	e.mu.Lock()
	s, scan := e.state, e.scan
	e.mu.Unlock()
	if s.Active != nil || !strings.HasPrefix(s.LastResult, "win (") {
		t.Fatalf("visible win was not settled: state=%+v scan=%+v", s, scan)
	}
	if scan.Code != "settled" || scan.File != "01-win.StormReplay" {
		t.Fatalf("wrong scan status: %+v", scan)
	}
}
func TestOldVisibleReplayDoesNotSettleNewerPrediction(t *testing.T) {
	now := time.Now()
	e, p, _ := boundScanFixture(t, now.Add(-35*time.Second))
	writeScanReplay(t, p.dir, "old-win", now.Add(-3*time.Minute), true)
	lastError := ""
	lastLog := time.Time{}
	e.scanTick(context.Background(), now.Add(-45*time.Second), &lastError, &lastLog)
	e.mu.Lock()
	s, scan := e.state, e.scan
	e.mu.Unlock()
	if s.Active == nil {
		t.Fatal("predating match wrongly distributed viewer points")
	}
	if scan.Code != "waiting_replay" {
		t.Fatalf("wrong diagnostic: %+v", scan)
	}
}
func TestRecheckSettlesCompletedReplayWhileMonitoringStopped(t *testing.T) {
	now := time.Now()
	e, p, lobby := boundScanFixture(t, now.Add(-55*time.Second))
	writeCorrelationFile(t, p.dir, "completed-win", correlationMPQTest(correlationDetailsTest(t, lobby, now.Add(-55*time.Second), true), lobby), now.Add(-18*time.Second))
	scan, err := e.recheckReplay(context.Background(), "SIMULATION", e.state.Active.StartedAt)
	if err != nil {
		t.Fatal(err)
	}
	if scan.Code != "settled" {
		t.Fatalf("recheck did not settle: %+v", scan)
	}
	e.mu.Lock()
	s := e.state
	e.mu.Unlock()
	if s.Active != nil || !strings.HasPrefix(s.LastResult, "win (") {
		t.Fatalf("manual recheck did not persist result: %+v", s)
	}
	if _, err = e.recheckReplay(context.Background(), "SIMULATION", now.Add(-55*time.Second).UnixMilli()); err == nil {
		t.Fatal("stale recheck was accepted")
	}
}
func TestRecheckAPIRequiresActivePredictionIdentity(t *testing.T){
 now:=time.Now()
 d:=manualDashboard(t,Config{PlayerName:"Проверка"})
 prediction:=&Prediction{ID:"SIMULATION",DryRun:true,StartedAt:now.Add(-20*time.Second).UnixMilli()}
 if err:=d.engine.updateAndSave(State{Active:prediction});err!=nil {t.Fatal(err)}
 for _,tc:=range []struct{stamp int64; allow bool; want int}{
  {prediction.StartedAt,false,403},
  {prediction.StartedAt-1,true,400},
  {prediction.StartedAt,true,200},
 }{
  body,_:=json.Marshal(map[string]any{"id":prediction.ID,"startedAt":tc.stamp})
  req:=httptest.NewRequest("POST","http://127.0.0.1:12345/secret/api/prediction/recheck",bytes.NewReader(body))
  req.Host="127.0.0.1:12345"
  req.Header.Set("Origin","http://127.0.0.1:12345")
  if tc.allow {req.Header.Set("X-Companion","1")}
  w:=httptest.NewRecorder()
  d.routes("127.0.0.1:12345","secret").ServeHTTP(w,req)
  if w.Code!=tc.want {t.Fatalf("wanted %d got %d: %s",tc.want,w.Code,w.Body.String())}
 }
}