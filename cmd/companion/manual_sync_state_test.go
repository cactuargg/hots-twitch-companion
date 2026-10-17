package main

import (
	"context"
	"testing"
	"time"
)

type testManualSyncAPI struct {
	*fakePredictionAPI
	reply apiResponse
}

func (f *testManualSyncAPI) Prediction(context.Context, string, string) (*apiResponse, error) {
	return &f.reply, nil
}

func TestManualSyncUsesSameCompletionTransitionAsAutomaticSettlement(t *testing.T) {
	for _, tc := range []struct {
		name, status, winner, expected string
	}{
		{"winner", "RESOLVED", "win-id", "win"},
		{"loser", "RESOLVED", "loss-id", "loss"},
		{"unknown external result", "RESOLVED", "other", "external"},
		{"remote refund", "CANCELED", "", "cancel"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, p := liveLifecycleTest(t)
			at := time.Date(2026, 10, 17, 22, 10, 0, 0, time.UTC)
			e.clock = &manualClock{current: at}
			e.predictions = &testManualSyncAPI{
				fakePredictionAPI: &fakePredictionAPI{},
				reply: apiResponse{Data: []apiPrediction{{
					ID: e.state.Active.ID, Status: tc.status, WinningOutcomeID: tc.winner,
				}}},
			}
			id, started := e.state.Active.ID, e.state.Active.StartedAt
			if err := e.manualOutcome(context.Background(), "sync", id, started); err != nil {
				t.Fatal(err)
			}
			saved, err := loadState(p)
			if err != nil {
				t.Fatal(err)
			}
			if saved.Active != nil || saved.LastSettlementAt != at.UnixMilli() ||
				saved.LastResult != tc.expected+" (Twitch)" {
				t.Fatalf("manual sync diverged from shared settlement: %+v", saved)
			}
		})
	}
}

func TestManualSyncCannotCompleteRemoteOpenPrediction(t *testing.T) {
	e, p := liveLifecycleTest(t)
	e.predictions = &testManualSyncAPI{
		fakePredictionAPI: &fakePredictionAPI{},
		reply:             apiResponse{Data: []apiPrediction{{ID: e.state.Active.ID, Status: "ACTIVE"}}},
	}
	id, started := e.state.Active.ID, e.state.Active.StartedAt
	if err := e.manualOutcome(context.Background(), "sync", id, started); err == nil {
		t.Fatal("allowed manual sync to complete open remote Prediction")
	}
	saved, err := loadState(p)
	if err != nil || saved.Active == nil || saved.Active.ID != id {
		t.Fatalf("failed remote verification erased active Prediction: %+v %v", saved, err)
	}
}
