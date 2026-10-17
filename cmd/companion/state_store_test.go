package main

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

type failureInjectingStateStore struct {
	state      State
	shouldFail func(State) bool
	saves      int
}

func (s *failureInjectingStateStore) Load() (State, error) { return s.state, nil }
func (s *failureInjectingStateStore) Save(state State) error {
	s.saves++
	if s.shouldFail != nil && s.shouldFail(state) {
		return errors.New("injected durable write failure")
	}
	s.state = state
	return nil
}

func TestStateStoreFailureCannotSendPredictionCreationPOST(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TEMP", temp)
	t.Setenv("TMP", temp)
	p := paths{dir: temp, config: filepath.Join(temp, "config.json"), state: filepath.Join(temp, "state.json"), secret: filepath.Join(temp, "tokens.dpapi")}
	cfg := Config{ClientID: "client", RealMode: true, PlayerName: "Tester", Lang: "en"}
	store := &failureInjectingStateStore{shouldFail: func(s State) bool { return s.UnconfirmedCreation != nil && s.Active == nil }}
	e, err := newEngineWithStore(p, cfg, func(string) {}, store)
	if err != nil {
		t.Fatal(err)
	}
	testTokenPaths.Store(e.twitch, e.p.secret)
	seedToken(e.twitch, Token{AccessToken: "token", RefreshToken: "refresh", ExpiresIn: 3600, SavedAt: time.Now().Unix()})
	now := time.Now()
	lobbyFileTest(t, realLobbyTest(t), now.Add(-5*time.Second))

	oldClient := httpClient
	defer func() { httpClient = oldClient }()
	posts := 0
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/oauth2/validate" {
			return manualResponse(`{"client_id":"client","user_id":"channel","scopes":["channel:manage:predictions"]}`), nil
		}
		if req.Method == "GET" {
			return manualResponse(`{"data":[]}`), nil
		}
		if req.Method == "POST" {
			posts++
			return manualResponse(`{"data":[{"id":"pred","status":"ACTIVE","outcomes":[{"id":"win","title":"Win"},{"id":"loss","title":"Loss"}]}]}`), nil
		}
		t.Fatalf("unexpected Twitch request %s", req.Method)
		return nil, nil
	})}
	lastError := ""
	lastLog := time.Time{}
	started := now.Add(-time.Minute)
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if posts != 0 || store.state.PendingLobby == nil || store.state.UnconfirmedCreation != nil {
		t.Fatalf("failed write allowed POST or discarded pending event: posts=%d state=%+v", posts, store.state)
	}
	restarted, err := newEngineWithStore(p, cfg, func(string) {}, store)
	if err != nil {
		t.Fatal(err)
	}
	testTokenPaths.Store(restarted.twitch, restarted.p.secret)
	seedToken(restarted.twitch, testToken(e.twitch))
	store.shouldFail = nil
	restarted.scanTick(context.Background(), started, &lastError, &lastLog)
	if posts != 1 || store.state.Active == nil || store.state.Active.ID != "pred" ||
		store.state.UnconfirmedCreation != nil || store.state.PendingLobby != nil {
		t.Fatalf("recovery failed or duplicate request: posts=%d state=%+v", posts, store.state)
	}
	restarted.scanTick(context.Background(), started, &lastError, &lastLog)
	if posts != 1 {
		t.Fatalf("created duplicate Prediction after retry: posts=%d", posts)
	}
}

func TestStateStoreLoadFailureStopsStartup(t *testing.T) {
	temp := t.TempDir()
	store := &rejectingLoadStore{}
	p := paths{dir: temp, config: filepath.Join(temp, "config.json"), state: filepath.Join(temp, "state.json"), secret: filepath.Join(temp, "tokens.dpapi")}
	if _, err := newEngineWithStore(p, Config{}, func(string) {}, store); err == nil {
		t.Fatal("corrupt state must stop startup before Twitch client construction")
	}
}

type rejectingLoadStore struct{}

func (*rejectingLoadStore) Load() (State, error) { return State{}, errors.New("broken journal") }
func (*rejectingLoadStore) Save(State) error     { return nil }
