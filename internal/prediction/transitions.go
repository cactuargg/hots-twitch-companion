package prediction

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

// The following transitions are pure: they only prepare the next State.
// The Engine is responsible for persist-before-effect ordering and for
// invoking Twitch only after StateStore.Save succeeds.

// attemptedCreationState marks an event as consumed and records the uncertain
// request before the first irreversible POST. The remaining lobby queue is
// preserved and SHA history is cloned rather than mutating the old snapshot.
func AttemptedCreation(s State, event *LobbyEvent, broadcasterID string, real bool, at time.Time) (State, error) {
	if s.Active != nil || s.UnconfirmedCreation != nil || event == nil || event.Snapshot == nil ||
		event.Snapshot.SHA256 == "" || s.PendingLobby == nil ||
		s.PendingLobby.Snapshot == nil || s.PendingLobby.Snapshot.SHA256 != event.Snapshot.SHA256 {
		return State{}, errors.New("cannot attempt creation outside the pending lobby state")
	}
	if real && broadcasterID == "" {
		return State{}, errors.New("cannot create a real Prediction without a verified broadcaster")
	}
	next := s
	PopPendingLobby(&next)
	next.LastCreatedLobbySHA = event.Snapshot.SHA256
	next.RecentLobbySHAs = append(append([]string(nil), s.RecentLobbySHAs...), event.Snapshot.SHA256)
	if len(next.RecentLobbySHAs) > 64 {
		next.RecentLobbySHAs = next.RecentLobbySHAs[len(next.RecentLobbySHAs)-64:]
	}
	next.LastLobby = event.Snapshot
	if real {
		next.UnconfirmedCreation = &CreationAttempt{
			LobbySHA: event.Snapshot.SHA256,
			BroadcasterID: broadcasterID,
			AttemptedAt: at.UnixMilli(),
		}
	}
	return next, nil
}

// confirmedCreationState never mutates the supplied Prediction or State.
// The new remote ID and removal of uncertainty are saved in one state write.
func ConfirmedCreation(s State, prediction *Prediction, binding *MatchBinding) (State, error) {
	if s.Active != nil || prediction == nil || prediction.ID == "" {
		return State{}, errors.New("cannot confirm creation with missing or existing Prediction")
	}
	if !prediction.DryRun {
		if prediction.BroadcasterID == "" {
			return State{}, errors.New("real Prediction has no broadcaster")
		}
		if s.UnconfirmedCreation != nil && s.UnconfirmedCreation.BroadcasterID != prediction.BroadcasterID {
			return State{}, errors.New("confirmed broadcaster differs from attempted creation")
		}
	}
	next := s
	active := *prediction
	active.Match = binding
	next.Active = &active
	next.UnconfirmedCreation = nil
	return next, nil
}

// pendingSettlementState latches the outcome once. Later retries cannot
// replace the chosen Win/Loss/Cancel or the original source.
func PendingSettlement(s State, outcome, source string) (State, error) {
	if s.Active == nil || (outcome != "win" && outcome != "loss" && outcome != "cancel") {
		return State{}, errors.New("invalid settlement intent")
	}
	if s.Active.Settlement != nil {
		if s.Active.Settlement.Outcome != outcome {
			return State{}, errors.New("conflicting settlement outcome")
		}
		return s, nil
	}
	next := s
	active := *s.Active
	active.Settlement = &SettlementAttempt{Outcome: outcome, Source: source}
	next.Active = &active
	return next, nil
}

// retrySettlementState preserves the immutable decision while updating only
// retry scheduling. The Twitch rate-limit deadline takes precedence.
func RetrySettlement(s State, at time.Time, earliestRetryAt int64) State {
	next := s
	active := *s.Active
	retry := *active.Settlement
	retry.Failures++
	retry.RetryAt = at.Add(time.Duration(min(retry.Failures, 6)) * 10 * time.Second).UnixMilli()
	retry.RetryAt = max(retry.RetryAt, earliestRetryAt)
	active.Settlement = &retry
	next.Active = &active
	return next
}

func Completed(s State, outcome, source string, at time.Time) (State, error) {
	if s.Active == nil {
		return State{}, errors.New("cannot complete without an active Prediction")
	}
	next := s
	next.LastSettlementAt = at.UnixMilli()
	next.LastResult = fmt.Sprintf("%s (%s)", outcome, filepath.Base(source))
	next.Active = nil
	return next, nil
}

// PopPendingLobby advances the persistent FIFO without altering the older snapshot.
func PopPendingLobby(s *State) {
	s.PendingLobby = nil
	if len(s.PendingLobbies) > 0 {
		s.PendingLobby = s.PendingLobbies[0]
		s.PendingLobbies = append([]*LobbyEvent(nil), s.PendingLobbies[1:]...)
	}
}
