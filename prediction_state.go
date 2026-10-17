package main

import (
	"time"
	predstate "hots-twitch-companion/internal/prediction"
)

// Transitional root wrappers preserve call sites while Engine/UI packages
// migrate. The actual state models and mutations live in internal/prediction.
func attemptedCreationState(s State, event *LobbyEvent, broadcasterID string, real bool, at time.Time) (State, error) {
	return predstate.AttemptedCreation(s,event,broadcasterID,real,at)
}
func confirmedCreationState(s State, prediction *Prediction, binding *MatchBinding) (State,error) {
	return predstate.ConfirmedCreation(s,prediction,binding)
}
func pendingSettlementState(s State, outcome,source string) (State,error) {
	return predstate.PendingSettlement(s,outcome,source)
}
func retrySettlementState(s State, at time.Time, earliestRetryAt int64) State {
	return predstate.RetrySettlement(s,at,earliestRetryAt)
}
func completedPredictionState(s State,outcome,source string,at time.Time) (State,error) {
	return predstate.Completed(s,outcome,source,at)
}
