package app

import (
	"context"
	"errors"
	"hots-twitch-companion/internal/config"
	predmodel "hots-twitch-companion/internal/prediction"
	twitchpkg "hots-twitch-companion/internal/twitch"
	"path/filepath"
	"testing"
	"time"
)

type fakePredictionAPI struct {
	preflightCalls, createCalls, endCalls int
	preflightError                        error
	onCreate                              func() error
	onEnd                                 func(*predmodel.Prediction, string) error
}

func (f *fakePredictionAPI) PreflightCreate(context.Context) (twitchpkg.Identity, error) {
	f.preflightCalls++
	if f.preflightError != nil {
		return twitchpkg.Identity{}, f.preflightError
	}
	return twitchpkg.Identity{UserID: "channel"}, nil
}
func (f *fakePredictionAPI) CreatePrepared(_ context.Context, broadcaster string, _ int, _, win, loss string) (*predmodel.Prediction, error) {
	f.createCalls++
	if f.onCreate != nil {
		if err := f.onCreate(); err != nil {
			return nil, err
		}
	}
	return &predmodel.Prediction{ID: "fake-prediction", BroadcasterID: broadcaster, WinID: "win-01", LossID: "loss-01", StartedAt: time.Now().UnixMilli()}, nil
}
func (f *fakePredictionAPI) Create(ctx context.Context, _ int, _, _, _ string) (*predmodel.Prediction, error) {
	me, err := f.PreflightCreate(ctx)
	if err != nil {
		return nil, err
	}
	return f.CreatePrepared(ctx, me.UserID, 2, "", "", "")
}
func (f *fakePredictionAPI) Prediction(context.Context, string, string) (*twitchpkg.APIResponse, error) {
	return &twitchpkg.APIResponse{}, nil
}
func (f *fakePredictionAPI) End(_ context.Context, p *predmodel.Prediction, outcome string) error {
	f.endCalls++
	if f.onEnd != nil {
		return f.onEnd(p, outcome)
	}
	return nil
}

func TestInjectedPredictionAPIRetriesOnlyPreflightAndPersistsBeforePost(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TEMP", temp)
	t.Setenv("TMP", temp)
	p := config.Paths{Dir: temp, Config: filepath.Join(temp, "config.json"), State: filepath.Join(temp, "state.json"), Secret: filepath.Join(temp, "tokens.dpapi")}
	e, err := newEngine(p, config.Data{RealMode: true, PlayerName: "Tester", Lang: "en"}, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakePredictionAPI{preflightError: errors.New("temporary validation outage")}
	fake.onCreate = func() error {
		s, err := loadState(p)
		if err != nil {
			return err
		}
		if s.Active != nil || s.UnconfirmedCreation == nil || s.LastCreatedLobbySHA == "" || s.PendingLobby != nil {
			return errors.New("POST attempted before the lobby and uncertainty marker were committed")
		}
		return nil
	}
	e.predictions = fake
	now := time.Now()
	lobbyFileTest(t, realLobbyTest(t), now.Add(-5*time.Second))
	lastError := ""
	lastLog := time.Time{}
	started := now.Add(-time.Minute)
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if fake.createCalls != 0 || e.state.PendingLobby == nil || e.state.UnconfirmedCreation != nil {
		t.Fatalf("preflight error consumed lobby: calls=%d state=%+v", fake.createCalls, e.state)
	}
	fake.preflightError = nil
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if fake.createCalls != 1 || e.state.Active == nil || e.state.UnconfirmedCreation != nil {
		t.Fatalf("confirmed creation was not durably committed: calls=%d state=%+v", fake.createCalls, e.state)
	}
	e.scanTick(context.Background(), started, &lastError, &lastLog)
	if fake.createCalls != 1 {
		t.Fatalf("duplicate create: %d", fake.createCalls)
	}
}

func TestInjectedPredictionAPISeesDurableSettlementIntent(t *testing.T) {
	e, p := liveLifecycleTest(t)
	fake := &fakePredictionAPI{}
	fake.onEnd = func(predict *predmodel.Prediction, outcome string) error {
		s, err := loadState(p)
		if err != nil {
			return err
		}
		if s.Active == nil || s.Active.Settlement == nil || s.Active.Settlement.Outcome != outcome ||
			s.Active.ID != predict.ID {
			return errors.New("remote settlement started without durable intent")
		}
		return nil
	}
	e.predictions = fake
	s := e.state
	if err := e.doSettle(context.Background(), &s, "win", "manual"); err != nil {
		t.Fatal(err)
	}
	if fake.endCalls != 1 || e.state.Active != nil || s.Active != nil {
		t.Fatalf("settlement failed: calls=%d state=%+v", fake.endCalls, e.state)
	}
}
