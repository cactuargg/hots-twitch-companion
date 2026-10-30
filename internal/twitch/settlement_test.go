package twitch

import (
	"context"
	"encoding/json"
	"errors"
	"hots-twitch-companion/internal/prediction"
	"testing"
)

func TestFinishPredictionVerifiesExactRemoteOutcome(t *testing.T) {
	p := &prediction.Prediction{ID: "wager", BroadcasterID: "channel", WinID: "win", LossID: "loss"}
	for _, tc := range []struct{ outcome, status, winner string }{
		{"win", "RESOLVED", "win"}, {"loss", "RESOLVED", "loss"}, {"cancel", "CANCELED", ""},
	} {
		t.Run(tc.outcome, func(t *testing.T) {
			reads, patches := 0, 0
			read := func(context.Context, string, string) (*APIResponse, error) {
				reads++
				return pointsResponse("wager", "ACTIVE", "win", "loss", 100, 200), nil
			}
			patch := func(_ context.Context, body map[string]any) (*APIResponse, error) {
				patches++
				if body["status"] != tc.status || body["broadcaster_id"] != "channel" || body["id"] != "wager" {
					t.Fatalf("bad settlement body: %+v", body)
				}
				if tc.winner != "" && body["winning_outcome_id"] != tc.winner {
					t.Fatalf("wrong winner: %+v", body)
				}
				if tc.winner == "" && body["winning_outcome_id"] != nil {
					t.Fatal("cancel included winner")
				}
				return &APIResponse{Data: []APIPrediction{{ID: "wager", Status: tc.status, WinningOutcomeID: tc.winner}}}, nil
			}
			if err := FinishPrediction(context.Background(), p, tc.outcome, read, patch); err != nil {
				t.Fatal(err)
			}
			if reads != 1 || patches != 1 {
				t.Fatalf("unexpected read/patch %d/%d", reads, patches)
			}
		})
	}
}

func TestFinishPredictionLostPatchResponseDoesNotRepay(t *testing.T) {
	p := &prediction.Prediction{ID: "one", BroadcasterID: "owner", WinID: "yes", LossID: "no"}
	reads, patches := 0, 0
	read := func(context.Context, string, string) (*APIResponse, error) {
		reads++
		status := "ACTIVE"
		winner := ""
		if reads > 1 {
			status = "RESOLVED"
			winner = "yes"
		}
		if status == "ACTIVE" {
			return pointsResponse("one", status, "yes", "no", 100, 200), nil
		}
		return &APIResponse{Data: []APIPrediction{{ID: "one", Status: status, WinningOutcomeID: winner}}}, nil
	}
	patch := func(context.Context, map[string]any) (*APIResponse, error) {
		patches++
		return nil, errors.New("lost response")
	}
	if err := FinishPrediction(context.Background(), p, "win", read, patch); err != nil {
		t.Fatal(err)
	}
	if patches != 1 || reads != 2 {
		t.Fatalf("lost-response recovery: %d GET and %d PATCH", reads, patches)
	}
}

func TestFinishPredictionRefusesRemoteIDMismatchAndConflictingResolution(t *testing.T) {
	p := &prediction.Prediction{ID: "one", BroadcasterID: "owner", WinID: "yes", LossID: "no"}
	patches := 0
	patch := func(context.Context, map[string]any) (*APIResponse, error) { patches++; return nil, nil }
	readWrong := func(context.Context, string, string) (*APIResponse, error) {
		return &APIResponse{Data: []APIPrediction{{ID: "other", Status: "ACTIVE"}}}, nil
	}
	if err := FinishPrediction(context.Background(), p, "win", readWrong, patch); err == nil || patches != 0 {
		t.Fatalf("mismatched remote id: %v, patches=%d", err, patches)
	}
	readCanceled := func(context.Context, string, string) (*APIResponse, error) {
		return &APIResponse{Data: []APIPrediction{{ID: "one", Status: "CANCELED"}}}, nil
	}
	err := FinishPrediction(context.Background(), p, "win", readCanceled, patch)
	var finished *RemoteFinishedError
	if !errors.As(err, &finished) || patches != 0 || RemoteOutcome(finished.Prediction, p) != "cancel" {
		t.Fatalf("conflicting finalization %v, patches=%d", err, patches)
	}
}

func pointsResponse(id, status, win, loss string, a, b int64) *APIResponse {
	raw, _ := json.Marshal(map[string]any{"data": []any{map[string]any{"id": id, "status": status, "outcomes": []any{map[string]any{"id": win, "channel_points": a}, map[string]any{"id": loss, "channel_points": b}}}}})
	var r APIResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		panic(err)
	}
	return &r
}

func TestFinishPredictionZeroPointsAndMissingData(t *testing.T) {
	p := &prediction.Prediction{ID: "one", BroadcasterID: "owner", WinID: "yes", LossID: "no"}
	for _, outcome := range []string{"win", "loss"} {
		for _, tc := range []struct {
			name     string
			a, b     int64
			missing  bool
			wantZero bool
		}{
			{"empty win", 0, 200, false, true}, {"empty loss", 100, 0, false, true}, {"both empty", 0, 0, false, true},
			{"both funded", 100, 200, false, false}, {"missing points", 100, 200, true, false},
		} {
			t.Run(outcome+"/"+tc.name, func(t *testing.T) {
				r := pointsResponse("one", "LOCKED", "yes", "no", tc.a, tc.b)
				if tc.missing {
					r.Data[0].Outcomes[0].ChannelPoints = nil
				}
				patches := 0
				err := FinishPrediction(context.Background(), p, outcome, func(context.Context, string, string) (*APIResponse, error) { return r, nil }, func(_ context.Context, body map[string]any) (*APIResponse, error) {
					patches++
					winner := "yes"
					if outcome == "loss" {
						winner = "no"
					}
					return &APIResponse{Data: []APIPrediction{{ID: "one", Status: "RESOLVED", WinningOutcomeID: winner}}}, nil
				})
				if tc.wantZero {
					if !errors.Is(err, ErrZeroPoints) || patches != 0 {
						t.Fatalf("err=%v patches=%d", err, patches)
					}
				} else if tc.missing {
					if err == nil || patches != 0 {
						t.Fatalf("missing points: %v, patches=%d", err, patches)
					}
				} else if err != nil || patches != 1 {
					t.Fatalf("funded: %v, patches=%d", err, patches)
				}
			})
		}
	}
}
