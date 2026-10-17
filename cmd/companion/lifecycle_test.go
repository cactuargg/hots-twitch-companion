package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func remoteResponseTest(id, status, winner string) *http.Response {
	raw, _ := json.Marshal(map[string]any{"data": []map[string]string{{"id": id, "status": status, "winning_outcome_id": winner}}})
	return manualResponse(string(raw))
}

func liveLifecycleTest(t *testing.T) (*Engine, paths) {
	t.Helper()
	e, p := scanFixture(t, time.Now().Add(-time.Minute))
	e.cfg.ClientID = "client"
	e.cfg.RealMode = true
	e.twitch.clientID = "client"
	active := *e.state.Active
	active.ID = "tracked"
	active.DryRun = false
	active.BroadcasterID = "channel"
	active.WinID = "win-id"
	active.LossID = "loss-id"
	s := e.state
	s.Active = &active
	if err := e.updateAndSave(s); err != nil {
		t.Fatal(err)
	}
	e.twitch.token = Token{AccessToken: "fake", ExpiresIn: 3600, SavedAt: time.Now().Unix()}
	return e, p
}

func TestLiveSettlementTransitions(t *testing.T) {
	old := httpClient
	defer func() { httpClient = old }()
	for _, tc := range []struct {
		name, remote, outcome, expected string
		patches                         int
	}{
		{"active win", "ACTIVE", "win", "win", 1},
		{"locked loss", "LOCKED", "loss", "loss", 1},
		{"active refund", "ACTIVE", "cancel", "cancel", 1},
		{"locked refund", "LOCKED", "cancel", "cancel", 1},
		{"already same winner", "RESOLVED", "win", "win", 0},
		{"externally refunded", "CANCELED", "win", "cancel", 0},
		{"external opposite winner", "RESOLVED", "loss", "win", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, p := liveLifecycleTest(t)
			patches := 0
			httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
				if req.URL.Query().Get("id") != "tracked" && req.Method == "GET" {
					t.Fatal("GET not scoped to tracked ID")
				}
				if req.Method == "GET" {
					return remoteResponseTest("tracked", tc.remote, "win-id"), nil
				}
				patches++
				persisted, err := loadState(p)
				if err != nil || persisted.Active == nil || persisted.Active.Settlement == nil || persisted.Active.Settlement.Outcome != tc.outcome {
					t.Fatalf("effect before durable intent: %+v %v", persisted, err)
				}
				var body map[string]any
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["id"] != "tracked" || body["broadcaster_id"] != "channel" {
					t.Fatalf("wrong target: %+v", body)
				}
				winner := "win-id"
				if tc.outcome == "loss" {
					winner = "loss-id"
				}
				status := "RESOLVED"
				if tc.outcome == "cancel" {
					status = "CANCELED"
					winner = ""
					if body["winning_outcome_id"] != nil {
						t.Fatal("refund selects winner")
					}
				}
				return remoteResponseTest("tracked", status, winner), nil
			})}
			s := e.state
			if err := e.doSettle(context.Background(), &s, tc.outcome, "manual"); err != nil {
				t.Fatal(err)
			}
			persisted, err := loadState(p)
			if err != nil || persisted.Active != nil || !strings.HasPrefix(persisted.LastResult, tc.expected+" (") || patches != tc.patches {
				t.Fatalf("bad transition: %+v %v patches=%d", persisted, err, patches)
			}
		})
	}
}

func TestLostPatchResponseIsVerifiedWithoutSecondPatch(t *testing.T) {
	e, p := liveLifecycleTest(t)
	old := httpClient
	defer func() { httpClient = old }()
	patches, gets := 0, 0
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method == "PATCH" {
			patches++
			return nil, errors.New("response lost after commit")
		}
		gets++
		if gets == 1 {
			return remoteResponseTest("tracked", "ACTIVE", ""), nil
		}
		return remoteResponseTest("tracked", "RESOLVED", "win-id"), nil
	})}
	s := e.state
	if err := e.doSettle(context.Background(), &s, "win", "manual"); err != nil {
		t.Fatal(err)
	}
	persisted, err := loadState(p)
	if err != nil || persisted.Active != nil || gets != 2 || patches != 1 {
		t.Fatalf("bad response recovery: %+v %v gets=%d patches=%d", persisted, err, gets, patches)
	}
}

func TestUnknownSettlementKeepsOutcomeAcrossRestart(t *testing.T) {
	e, p := liveLifecycleTest(t)
	old := httpClient
	defer func() { httpClient = old }()
	patches := 0
	lost := false
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method == "PATCH" {
			patches++
			lost = true
			return nil, errors.New("connection lost")
		}
		if lost {
			return nil, errors.New("offline")
		}
		return remoteResponseTest("tracked", "LOCKED", ""), nil
	})}
	s := e.state
	if err := e.doSettle(context.Background(), &s, "win", "manual"); err == nil {
		t.Fatal("unknown result reported success")
	}
	saved, err := loadState(p)
	if err != nil || saved.Active == nil || saved.Active.Settlement == nil || saved.Active.Settlement.Outcome != "win" {
		t.Fatalf("lost durable intent: %+v %v", saved, err)
	}
	if err := e.doSettle(context.Background(), &s, "loss", "manual"); err == nil || patches != 1 {
		t.Fatal("changed unresolved outcome")
	}
	if err := e.doSettle(context.Background(), &s, "win", "manual"); err == nil || patches != 1 {
		t.Fatal("ignored retry delay")
	}
	saved.Active.Settlement.RetryAt = 0
	if err := saveState(p, saved); err != nil {
		t.Fatal(err)
	}
	restarted, err := newEngine(p, e.cfg, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	restarted.twitch.token = e.twitch.token
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Fatal("already committed outcome was patched twice")
		}
		return remoteResponseTest("tracked", "RESOLVED", "win-id"), nil
	})}
	temp := t.TempDir()
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	lastError := ""
	lastLog := time.Time{}
	restarted.scanTick(context.Background(), time.Now(), &lastError, &lastLog)
	if restarted.state.Active != nil || !strings.HasPrefix(restarted.state.LastResult, "win (") {
		t.Fatal("restart did not confirm durable decision")
	}
}

func TestBadSuccessBodiesAndRemoteErrorsStayPending(t *testing.T) {
	old := httpClient
	defer func() { httpClient = old }()
	for _, tc := range []struct {
		name, reply string
		status      int
		rate        bool
	}{
		{"empty success", `{"data":[]}`, 200, false},
		{"wrong ID", `{"data":[{"id":"other","status":"RESOLVED","winning_outcome_id":"win-id"}]}`, 200, false},
		{"contradictory status", `{"data":[{"id":"tracked","status":"CANCELED"}]}`, 200, false},
		{"server error", `{"message":"try later"}`, 503, false},
		{"rate limit", `{"message":"slow down"}`, 429, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, p := liveLifecycleTest(t)
			httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
				if req.Method == "GET" {
					return remoteResponseTest("tracked", "ACTIVE", ""), nil
				}
				response := manualResponse(tc.reply)
				response.StatusCode = tc.status
				if tc.rate {
					response.Header = http.Header{"Retry-After": []string{"120"}}
				}
				return response, nil
			})}
			s := e.state
			if err := e.doSettle(context.Background(), &s, "win", "manual"); err == nil {
				t.Fatal("unconfirmed transition accepted")
			}
			saved, err := loadState(p)
			if err != nil || saved.Active == nil || saved.Active.Settlement == nil {
				t.Fatalf("lost state: %+v %v", saved, err)
			}
			if tc.rate && saved.Active.Settlement.RetryAt < time.Now().Add(115*time.Second).UnixMilli() {
				t.Fatal("ignored Twitch Retry-After")
			}
		})
	}
}

func TestPersistenceFailuresDoNotForgetRemoteDecision(t *testing.T) {
	e, p := liveLifecycleTest(t)
	old := httpClient
	defer func() { httpClient = old }()
	calls := 0
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("must not call Twitch")
	})}
	e.p.state = filepath.Join(p.dir, "missing", "state.json")
	s := e.state
	if err := e.doSettle(context.Background(), &s, "win", "manual"); err == nil || calls != 0 || e.state.Active.Settlement != nil {
		t.Fatal("unsaved intent reached Twitch or memory advanced")
	}
	e.p = p
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method == "GET" {
			return remoteResponseTest("tracked", "ACTIVE", ""), nil
		}
		e.p.state = filepath.Join(p.dir, "missing", "state.json")
		return remoteResponseTest("tracked", "RESOLVED", "win-id"), nil
	})}
	s = e.state
	if err := e.doSettle(context.Background(), &s, "win", "manual"); err == nil || e.state.Active == nil {
		t.Fatal("forgot active ID after failed final save")
	}
	restored, err := newEngine(p, e.cfg, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	restored.twitch.token = e.twitch.token
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Fatal("duplicate PATCH after failed final save")
		}
		return remoteResponseTest("tracked", "RESOLVED", "win-id"), nil
	})}
	recovered := restored.state
	if err := restored.doSettle(context.Background(), &recovered, "win", "manual"); err != nil || restored.state.Active != nil {
		t.Fatalf("failed final-save recovery: %v", err)
	}
}

func lobbyFileTest(t *testing.T, raw []byte, at time.Time) string {
	t.Helper()
	root := filepath.Join(os.TempDir(), "Heroes of the Storm", "TempWriteReplayP1")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "replay.server.battlelobby")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestQuickSequentialMatchesKeepNextLobby(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	now := time.Now()
	e, p := scanFixture(t, now)
	e.state.Active = nil
	first := realLobbyTest(t)
	lobby, err := parseBattleLobby(first)
	if err != nil {
		t.Fatal(err)
	}
	e.cfg.PlayerToonHandle = lobby.Players[0].ToonHandle
	lobbyFileTest(t, first, now.Add(-30*time.Second))
	lastError := ""
	lastLog := time.Time{}
	e.scanTick(context.Background(), now.Add(-time.Minute), &lastError, &lastLog)
	if e.state.Active == nil {
		t.Fatal("first Prediction missing")
	}
	original := *e.state.Active
	// Deliberate synthetic tail change: same roster, new exact lobby content.
	second := append(append([]byte(nil), first...), 0)
	lobbyFileTest(t, second, now.Add(-5*time.Second))
	e.scanTick(context.Background(), now.Add(-time.Minute), &lastError, &lastLog)
	if e.state.PendingLobby == nil || e.state.Active == nil {
		t.Fatal("next lobby not queued durably")
	}
	restarted, err := newEngine(p, e.cfg, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	e = restarted
	writeCorrelationFile(t, p.dir, "first", correlationMPQTest(correlationDetailsTest(t, first, now.Add(-30*time.Second), true), first), now.Add(-10*time.Second))
	e.scanTick(context.Background(), time.Now(), &lastError, &lastLog)
	if e.state.Active == nil || e.state.Active.Match.Lobby.SHA256 != contentSHA(second) || e.state.Active.StartedAt == original.StartedAt || !strings.HasPrefix(e.state.LastResult, "win (") {
		t.Fatalf("next game lost or cooldown applied: %+v", e.state)
	}
	if err := e.manualOutcome(context.Background(), "loss", original.ID, original.StartedAt); err == nil {
		t.Fatal("stale first-game button settled next game")
	}
	writeCorrelationFile(t, p.dir, "second", correlationMPQTest(correlationDetailsTest(t, second, now.Add(-5*time.Second), false), second), time.Now().Add(-9*time.Second))
	// Align second file mtime with the observed loading time, allowing the 2s
	// tolerance without sleeping in the test.
	secondPath := filepath.Join(p.dir, "second.StormReplay")
	at := now.Add(-6 * time.Second)
	if err := os.Chtimes(secondPath, at, at); err != nil {
		t.Fatal(err)
	}
	s := e.state
	e.scanReplay(context.Background(), &s, now.Add(10*time.Second), &lastError, &lastLog)
	if e.state.Active != nil || !strings.HasPrefix(e.state.LastResult, "loss (") {
		t.Fatalf("second match did not settle: %+v %+v", e.state, e.scan)
	}
	// Rewriting identical contents cannot create a third Prediction.
	lobbyFileTest(t, second, now.Add(-4*time.Second))
	e.scanTick(context.Background(), now.Add(-time.Minute), &lastError, &lastLog)
	if e.state.Active != nil {
		t.Fatal("same lobby created twice")
	}
	lobbyFileTest(t, first, now.Add(-3*time.Second))
	e.scanTick(context.Background(), now.Add(-time.Minute), &lastError, &lastLog)
	if e.state.Active != nil {
		t.Fatal("an earlier consumed lobby was reused after the next match")
	}
}



func TestThreeRapidLobbiesSurviveActivePredictionAndRestart(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	now := time.Now()
	e, p := scanFixture(t, now)
	s := e.state
	s.Active = nil
	if err := e.updateAndSave(s); err != nil { t.Fatal(err) }

	first := realLobbyTest(t)
	second := append(append([]byte(nil), first...), 0)
	third := append(append([]byte(nil), second...), 1)
	lastError := ""
	lastLog := time.Time{}
	started := now.Add(-time.Minute)

	lobbyFileTest(t, first, now.Add(-18*time.Second))
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if e.state.Active == nil || e.state.Active.Match == nil ||
		e.state.Active.Match.Lobby.SHA256 != contentSHA(first) {
		t.Fatalf("first lobby not created: %+v", e.state)
	}
	lobbyFileTest(t, second, now.Add(-12*time.Second))
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	lobbyFileTest(t, third, now.Add(-6*time.Second))
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if e.state.PendingLobby == nil || e.state.PendingLobby.Snapshot.SHA256 != contentSHA(second) ||
		len(e.state.PendingLobbies) != 1 || e.state.PendingLobbies[0].Snapshot.SHA256 != contentSHA(third) {
		t.Fatalf("second lobby overwritten by third: %+v", e.state)
	}
	// Re-reading the same lobby must not enqueue a duplicate.
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if len(e.state.PendingLobbies) != 1 { t.Fatalf("duplicate queue entry: %+v",e.state) }
	restarted, err := newEngine(p, e.cfg, func(string){})
	if err != nil { t.Fatal(err) }
	e = restarted
	original := e.state.Active
	if err := e.manualOutcome(context.Background(), "win", original.ID, original.StartedAt); err != nil { t.Fatal(err) }
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if e.state.Active == nil || e.state.Active.Match == nil ||
		e.state.Active.Match.Lobby.SHA256 != contentSHA(second) ||
		e.state.PendingLobby == nil || e.state.PendingLobby.Snapshot.SHA256 != contentSHA(third) ||
		len(e.state.PendingLobbies) != 0 {
		t.Fatalf("FIFO did not create second lobby: %+v",e.state)
	}
	secondActive := e.state.Active
	if err := e.manualOutcome(context.Background(), "win", secondActive.ID, secondActive.StartedAt); err != nil {t.Fatal(err)}
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if e.state.Active == nil || e.state.Active.Match == nil ||
		e.state.Active.Match.Lobby.SHA256 != contentSHA(third) ||
		e.state.PendingLobby != nil || len(e.state.PendingLobbies) != 0 {
		t.Fatalf("FIFO did not create third lobby: %+v",e.state)
	}
	thirdActive:=e.state.Active
	if err := e.manualOutcome(context.Background(), "win", thirdActive.ID, thirdActive.StartedAt); err != nil {t.Fatal(err)}
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if e.state.Active != nil || e.state.PendingLobby != nil {
		t.Fatal("handled lobby was recreated")
	}
}

func TestPreflightFailureRetriesPendingLobbyWithoutDuplicatePost(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	now := time.Now()
	e, p := liveLifecycleTest(t)
	e.cfg.PredictionWin = "Win"
	e.cfg.PredictionLoss = "Loss"
	s := e.state
	s.Active = nil
	s.LastCreatedLobbySHA = ""
	s.RecentLobbySHAs = nil
	if err := e.updateAndSave(s); err != nil { t.Fatal(err) }
	lobbyFileTest(t, realLobbyTest(t), now.Add(-5*time.Second))

	oldClient := httpClient
	defer func() { httpClient = oldClient }()
	validations, posts := 0, 0
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Path == "/oauth2/validate":
			validations++
			if validations == 1 {
				response := manualResponse(`{"message":"temporarily unavailable"}`)
				response.StatusCode = 503
				return response, nil
			}
			return manualResponse(`{"client_id":"client","user_id":"channel","scopes":["channel:manage:predictions"]}`), nil
		case req.Method == "GET":
			return manualResponse(`{"data":[]}`), nil
		case req.Method == "POST":
			posts++
			saved, err := loadState(p)
			if err != nil || saved.PendingLobby != nil || saved.LastCreatedLobbySHA == "" {
				t.Fatalf("POST without durable consumption: %+v %v", saved, err)
			}
			return manualResponse(`{"data":[{"id":"created","status":"ACTIVE","outcomes":[{"id":"w","title":"Win"},{"id":"l","title":"Loss"}]}]}`), nil
		}
		t.Fatalf("unexpected request %s %s", req.Method, req.URL)
		return nil, nil
	})}
	lastError := ""
	lastLog := time.Time{}
	started := now.Add(-time.Minute)
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if e.state.PendingLobby == nil || e.state.LastCreatedLobbySHA != "" || posts != 0 {
		t.Fatalf("preflight consumed retryable lobby: %+v posts=%d", e.state, posts)
	}
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if e.state.Active == nil || e.state.Active.ID != "created" || e.state.PendingLobby != nil || posts != 1 || e.state.UnconfirmedCreation != nil {
		t.Fatalf("preflight retry failed: %+v posts=%d", e.state, posts)
	}
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if posts != 1 { t.Fatalf("duplicate prediction POST: %d", posts) }
}

func TestUncertainCreationIsNeverAutomaticallyRepeated(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	now := time.Now()
	e, p := liveLifecycleTest(t)
	s := e.state
	s.Active = nil
	if err := e.updateAndSave(s); err != nil {
		t.Fatal(err)
	}
	lobbyFileTest(t, realLobbyTest(t), now.Add(-5*time.Second))
	old := httpClient
	defer func() { httpClient = old }()
	posts := 0
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/oauth2/validate" {
			return manualResponse(`{"client_id":"client","user_id":"channel","scopes":["channel:manage:predictions"]}`), nil
		}
		if req.Method == "GET" {
			return manualResponse(`{"data":[]}`), nil
		}
		posts++
		saved, err := loadState(p)
		if err != nil || saved.PendingLobby != nil || saved.LastCreatedLobbySHA == "" || saved.UnconfirmedCreation == nil {
			t.Fatal("creation event and uncertainty not durably recorded")
		}
		return nil, errors.New("POST response lost")
	})}
	lastError := ""
	lastLog := time.Time{}
	started := now.Add(-time.Minute)
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if e.state.UnconfirmedCreation == nil || e.state.Active != nil {
		t.Fatal("uncertain POST must retain durable warning without claiming an active Prediction")
	}
	restarted, err := newEngine(p, e.cfg, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	restarted.twitch.token = e.twitch.token
	restarted.scanTick(context.Background(), started, &lastError, &lastLog)
	if posts != 1 || restarted.state.Active != nil {
		t.Fatalf("unknown creation retried/adopted: posts=%d %+v", posts, restarted.state)
	}
}


// An uncertain POST must fail closed for ALL later lobbies, including across
// restart. Only an explicit, remote-verified reconciliation can remove it.
func TestUnconfirmedCreationBlocksOtherLobbiesUntilTwitchChecked(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	now := time.Now()
	e, p := liveLifecycleTest(t)
	s := e.state
	s.Active = nil
	s.LastCreatedLobbySHA = ""
	s.RecentLobbySHAs = nil
	s.UnconfirmedCreation = &CreationAttempt{LobbySHA:"old-lobby",BroadcasterID:"channel",AttemptedAt:now.Add(-time.Minute).UnixMilli()}
	if err := e.updateAndSave(s); err != nil {t.Fatal(err)}
	fresh := realLobbyTest(t)
	lobbyFileTest(t,fresh,now.Add(-5*time.Second))
	old := httpClient
	defer func(){httpClient=old}()
	requests:=0
	remoteStatus:="ACTIVE"
	remoteUser:="channel"
	httpClient=&http.Client{Transport:manualTransport(func(req *http.Request)(*http.Response,error){
		requests++
		if req.URL.Path=="/oauth2/validate" {
			return manualResponse(fmt.Sprintf(`{"client_id":"client","user_id":%q,"scopes":["channel:manage:predictions"]}`,remoteUser)),nil
		}
		if req.Method=="GET" {
			if remoteStatus=="ACTIVE" {return manualResponse(`{"data":[{"id":"remote-open","status":"ACTIVE"}]}`),nil}
			return manualResponse(`{"data":[]}`),nil
		}
		t.Fatalf("unexpected mutation during reconciliation: %s",req.Method)
		return nil,nil
	})}
	lastError:=""
	lastLog:=time.Time{}
	started:=now.Add(-time.Minute)
	e.scanTick(context.Background(),started,&lastError,&lastLog)
	if requests!=0 || e.state.Active!=nil || e.state.PendingLobby==nil {
		t.Fatalf("new lobby created or lost despite uncertain prior POST: %+v calls=%d",e.state,requests)
	}
	if err:=e.start();err==nil {t.Fatal("automation restarted with an unresolved creation")}
	if err:=e.updateConfig(Config{ClientID:"another",RealMode:true});err==nil {t.Fatal("OAuth app switched during reconciliation")}
	if err:=e.acknowledgeUnconfirmedCreation(context.Background());err==nil {t.Fatal("acknowledged despite live Twitch Prediction")}
	if e.state.UnconfirmedCreation==nil {t.Fatal("failed acknowledgment cleared marker")}
	remoteStatus="none"
	remoteUser="another-channel"
	if err:=e.acknowledgeUnconfirmedCreation(context.Background());err==nil {t.Fatal("another broadcaster was accepted")}
	remoteUser="channel"
	restarted,err:=newEngine(p,e.cfg,func(string){})
	if err!=nil {t.Fatal(err)}
	restarted.twitch.token=e.twitch.token
	if err:=restarted.acknowledgeUnconfirmedCreation(context.Background());err!=nil {t.Fatal(err)}
	persisted,err:=loadState(p)
	if err!=nil || persisted.UnconfirmedCreation!=nil || persisted.PendingLobby==nil {t.Fatalf("reconciliation did not persist or forgot next lobby: %+v %v",persisted,err)}
}

func TestCreationRefusesOpenOrMalformedPredictions(t *testing.T) {
	old := httpClient
	defer func() { httpClient = old }()
	for _, tc := range []struct {
		name, existing, reply string
		success               bool
		posts                 int
	}{
		{"create", "", `{"data":[{"id":"created","status":"ACTIVE","outcomes":[{"id":"w","title":"W"},{"id":"l","title":"L"}]}]}`, true, 1},
		{"already active", "ACTIVE", "", false, 0},
		{"already locked", "LOCKED", "", false, 0},
		{"empty response", "", `{"data":[]}`, false, 1},
		{"unexpected terminal", "", `{"data":[{"id":"created","status":"RESOLVED","outcomes":[{"id":"w","title":"W"},{"id":"l","title":"L"}]}]}`, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, _ := liveLifecycleTest(t)
			posts := 0
			httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/oauth2/validate" {
					return manualResponse(`{"client_id":"client","user_id":"channel","scopes":["channel:manage:predictions"]}`), nil
				}
				if req.Method == "GET" {
					if tc.existing != "" {
						return remoteResponseTest("other", tc.existing, ""), nil
					}
					return manualResponse(`{"data":[]}`), nil
				}
				posts++
				var body map[string]any
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["prediction_window"] != float64(120) {
					t.Fatalf("wrong prediction window: %+v", body)
				}
				return manualResponse(tc.reply), nil
			})}
			p, err := e.twitch.create(context.Background(), 2, "Test", "W", "L")
			if (err == nil) != tc.success || (p != nil) != tc.success || posts != tc.posts {
				t.Fatalf("creation %+v %v posts=%d", p, err, posts)
			}
		})
	}
}

func TestAuthFailureAndCanceledContextNeverPatch(t *testing.T) {
	e, p := liveLifecycleTest(t)
	old := httpClient
	defer func() { httpClient = old }()
	requests := 0
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method == "PATCH" {
			t.Fatal("revoked auth reached PATCH")
		}
		response := manualResponse(`{"message":"revoked"}`)
		response.StatusCode = 401
		return response, nil
	})}
	s := e.state
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := e.doSettle(ctx, &s, "win", "manual"); err == nil || requests != 0 {
		t.Fatal("canceled context mutated remote state")
	}
	if err := e.doSettle(context.Background(), &s, "win", "manual"); err == nil {
		t.Fatal("revoked auth accepted")
	}
	saved, err := loadState(p)
	if err != nil || saved.Active == nil || saved.Active.Settlement == nil || requests != 1 {
		t.Fatalf("auth failure lost intent: %+v %v calls=%d", saved, err, requests)
	}
}

func TestStaleQueuedLobbyAndTimeoutCannotChangePendingOutcome(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	e, p := liveLifecycleTest(t)
	s := e.state
	active := *s.Active
	active.StartedAt = time.Now().Add(-101 * time.Minute).UnixMilli()
	active.Settlement = &SettlementAttempt{Outcome: "win", Source: "manual"}
	s.Active = &active
	s.PendingLobby = &LobbyEvent{Key: "stale", Snapshot: bindingTest(t, realLobbyTest(t), time.Now().Add(-3*time.Minute)).Lobby}
	if err := e.updateAndSave(s); err != nil {
		t.Fatal(err)
	}
	old := httpClient
	defer func() { httpClient = old }()
	calls := 0
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != "GET" {
			t.Fatal("timeout overrode pending win or created stale match")
		}
		return remoteResponseTest("tracked", "RESOLVED", "win-id"), nil
	})}
	lastError := ""
	lastLog := time.Time{}
	e.scanTick(context.Background(), time.Now(), &lastError, &lastLog)
	saved, err := loadState(p)
	if err != nil || saved.Active != nil || saved.PendingLobby != nil || !strings.HasPrefix(saved.LastResult, "win (") || calls != 1 {
		t.Fatalf("bad timeout/queue recovery: %+v %v calls=%d", saved, err, calls)
	}
}
