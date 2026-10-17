package twitch

import (
 "context"
 "errors"
 "fmt"
 "hots-twitch-companion/internal/prediction"
)

type RemoteFinishedError struct{ Prediction APIPrediction }

func (e *RemoteFinishedError) Error() string { return "Prediction was already finalized on Twitch" }

func ExactPrediction(response *APIResponse, id string) (APIPrediction, error) {
	if response == nil || len(response.Data) != 1 || response.Data[0].ID != id {
		return APIPrediction{}, errors.New("Twitch did not return the exact Prediction")
	}
	return response.Data[0], nil
}

func TerminalPrediction(actual APIPrediction, status, winner string) error {
	if actual.Status == status && (status == "CANCELED" || actual.WinningOutcomeID == winner) {
		return nil
	}
	if actual.Status == "CANCELED" || actual.Status == "RESOLVED" {
		return &RemoteFinishedError{actual}
	}
	return errors.New("Twitch has not confirmed the requested final status")
}

// Read-before-write and exact response validation apply to first attempts and
// recovery alike. A lost response is never grounds to choose another outcome.
// FinishPrediction reads back Twitch status before and after PATCH to prevent
// duplicate or contradictory point settlements following network failures.
func FinishPrediction(ctx context.Context, p *prediction.Prediction, outcome string,
 read func(context.Context,string,string)(*APIResponse,error),
 patch func(context.Context,map[string]any)(*APIResponse,error)) error {
	if p == nil || p.ID == "" || p.BroadcasterID == "" || (outcome != "win" && outcome != "loss" && outcome != "cancel") {
		return errors.New("invalid Prediction or outcome")
	}
	status, winner := "RESOLVED", p.LossID
	if outcome == "win" {
		winner = p.WinID
	}
	if outcome == "cancel" {
		status = "CANCELED"
		winner = ""
	}
	if status == "RESOLVED" && (winner == "" || p.WinID == p.LossID) {
		return errors.New("invalid Prediction outcome IDs")
	}
	response, err := read(ctx, p.BroadcasterID, p.ID)
	if err != nil {
		return err
	}
	actual, err := ExactPrediction(response, p.ID)
	if err != nil {
		return err
	}
	if actual.Status == "RESOLVED" || actual.Status == "CANCELED" {
		return TerminalPrediction(actual, status, winner)
	}
	if actual.Status != "ACTIVE" && actual.Status != "LOCKED" {
		return errors.New("unsupported Twitch Prediction status")
	}
	body := map[string]any{"broadcaster_id": p.BroadcasterID, "id": p.ID, "status": status}
	if winner != "" {
		body["winning_outcome_id"] = winner
	}
	var result APIResponse
	requestErr := func()error {r,err:=patch(ctx,body);if r!=nil {result=*r};return err}()
	if requestErr == nil {
		confirmed, err := ExactPrediction(&result, p.ID)
		if err == nil {
			err = TerminalPrediction(confirmed, status, winner)
		}
		if err == nil {
			return nil
		}
		// A contradictory PATCH body is not a verified external finalization.
		// Only the follow-up GET may supply RemoteFinishedError on this path.
		requestErr = fmt.Errorf("unexpected Twitch response: %v", err)
	}
	// This request may have committed even if PATCH timed out or its body was bad.
	response, err = read(ctx, p.BroadcasterID, p.ID)
	if err == nil {
		actual, err = ExactPrediction(response, p.ID)
		if err == nil && (actual.Status == "RESOLVED" || actual.Status == "CANCELED") {
			return TerminalPrediction(actual, status, winner)
		}
	}
	return requestErr
}

func RemoteOutcome(actual APIPrediction, p *prediction.Prediction) string {
	if actual.Status == "CANCELED" {
		return "cancel"
	}
	if actual.WinningOutcomeID == p.WinID {
		return "win"
	}
	if actual.WinningOutcomeID == p.LossID {
		return "loss"
	}
	return "external"
}

