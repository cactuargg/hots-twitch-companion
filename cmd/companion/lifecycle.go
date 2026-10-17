package main

import (
 "fmt"
 twitchpkg "hots-twitch-companion/internal/twitch"
	transport "hots-twitch-companion/internal/twitch/transport"
	predstate "hots-twitch-companion/internal/prediction"
	"context"
	"errors"
	"net/http"
	"time"
)

type SettlementAttempt = predstate.SettlementAttempt

type twitchHTTPError = transport.HTTPError
func responseHTTPError(response *http.Response, message string) error {
	return transport.ResponseHTTPError(response,message)
}

type LobbyEvent = predstate.LobbyEvent

func lobbyWasHandled(s *State, hash string) bool {
	if hash == "" {
		return false
	}
	if hash == s.LastCreatedLobbySHA {
		return true
	}
	for _, previous := range s.RecentLobbySHAs {
		if previous == hash {
			return true
		}
	}
	return false
}

// PendingLobby is the oldest unconsumed event for compatibility with
// previously saved state. PendingLobbies holds the remaining FIFO entries.
const maxPendingLobbies = 32

func pendingLobbySeen(s *State, sha string) bool {
	if sha == "" {
		return false
	}
	if s.PendingLobby != nil && s.PendingLobby.Snapshot != nil && s.PendingLobby.Snapshot.SHA256 == sha {
		return true
	}
	for _, pending := range s.PendingLobbies {
		if pending != nil && pending.Snapshot != nil && pending.Snapshot.SHA256 == sha {
			return true
		}
	}
	return false
}

func enqueueLobby(s *State, event *LobbyEvent) bool {
	if event == nil || event.Snapshot == nil || event.Snapshot.SHA256 == "" ||
		lobbyWasHandled(s, event.Snapshot.SHA256) || pendingLobbySeen(s, event.Snapshot.SHA256) {
		return false
	}
	if s.PendingLobby == nil {
		s.PendingLobby = event
		return true
	}
	if len(s.PendingLobbies) >= maxPendingLobbies-1 {
		return false
	}
	// Clone to keep the previously persisted State snapshot immutable.
	s.PendingLobbies = append(append([]*LobbyEvent(nil), s.PendingLobbies...), event)
	return true
}

func popPendingLobby(s *State) { predstate.PopPendingLobby(s) }

type remoteFinishedError = twitchpkg.RemoteFinishedError

func exactPrediction(response *apiResponse,id string)(apiPrediction,error) {
 return twitchpkg.ExactPrediction(response,id)
}
func terminalPrediction(actual apiPrediction,status,winner string)error {
 return twitchpkg.TerminalPrediction(actual,status,winner)
}
func remoteOutcome(actual apiPrediction,p *Prediction)string {
 return twitchpkg.RemoteOutcome(actual,p)
}

func (e *Engine) commitSettlement(s *State, outcome, source string) error {
	next, err := completedPredictionState(*s, outcome, source, e.clock.Now())
	if err != nil {
		return err
	}
	if err := e.updateAndSave(next); err != nil {
		return err
	}
	*s = next
	return nil
}

func (e *Engine) settlePrediction(ctx context.Context, s *State, outcome, source string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Active == nil {
		return errors.New(tr(e.cfg.lang(), "settle.none_active"))
	}
	if outcome != "win" && outcome != "loss" && outcome != "cancel" {
		return errors.New(tr(e.cfg.lang(), "manual.invalid"))
	}
	if s.Active.Settlement != nil && s.Active.Settlement.Outcome != outcome {
		return errors.New(tr(e.cfg.lang(), "settle.pending_conflict"))
	}
	if s.Active.DryRun {
		return e.commitSettlement(s, outcome, source)
	}
	if s.Active.Settlement == nil {
		next, err := pendingSettlementState(*s, outcome, source)
		if err != nil {
			return err
		}
		if err := e.updateAndSave(next); err != nil {
			return err
		}
		*s = next
	}
	intent := s.Active.Settlement
	if intent.RetryAt > e.clock.Now().UnixMilli() {
		return errors.New(tr(e.cfg.lang(), "settle.retry_wait"))
	}
	err := e.predictions.end(ctx, s.Active, intent.Outcome)
	var finished *remoteFinishedError
	if errors.As(err, &finished) {
		return e.commitSettlement(s, remoteOutcome(finished.Prediction, s.Active), "Twitch")
	}
	if err != nil {
		minRetryAt := int64(0)
		var rateLimit *twitchHTTPError
		if errors.As(err, &rateLimit) {
			minRetryAt = rateLimit.RetryAt
		}
		next := retrySettlementState(*s, e.clock.Now(), minRetryAt)
		if saveErr := e.updateAndSave(next); saveErr != nil {
			return fmt.Errorf("%v; save retry: %w", err, saveErr)
		}
		*s = next
		return err
	}
	return e.commitSettlement(s, intent.Outcome, intent.Source)
}

// A fresh next lobby is retained while the preceding Prediction completes.
// Consume the event durably before POST: a timeout must not recreate the same
// Prediction on a later poll or after restart. Unknown remote creations are
// handled on Twitch; we never adopt a Prediction merely by its title.
func (e *Engine) scanLifecycle(ctx context.Context, s *State, started, now time.Time, lastError *string, lastLog *time.Time) {
	path, key, modified, err := findLobby()
	if err == nil && now.Sub(modified) > 2500*time.Millisecond && (key != s.LastLobbyKey || s.LastLobby != nil && s.LastLobby.Status == "read_error") {
		snapshot := captureBattleLobby(path)
		if snapshot.ModifiedAt > 0 && (snapshot.ModifiedAt != modified.UnixMilli() || now.Sub(time.UnixMilli(snapshot.ModifiedAt)) < 2500*time.Millisecond) {
			return
		}
		next := *s
		next.LastLobby = snapshot
		next.LastLobbyKey = key
		if next.Active != nil {
			next.Active = observeActiveLobby(next.Active, snapshot, key)
		}
		fresh := !modified.Before(started.Add(-2*time.Second)) && now.Sub(modified) < 120*time.Second
		sameActive := next.Active != nil && next.Active.Match != nil && next.Active.Match.Lobby != nil && next.Active.Match.Lobby.SHA256 == snapshot.SHA256
		if fresh && snapshot.Status != "read_error" && !sameActive && !lobbyWasHandled(&next, snapshot.SHA256) {
			if !enqueueLobby(&next, &LobbyEvent{Key: key, Snapshot: snapshot}) &&
				!pendingLobbySeen(&next, snapshot.SHA256) && !lobbyWasHandled(&next, snapshot.SHA256) {
				e.report("warning", "lobby_queue_full", "Pending lobby queue is full")
			}
		}
		if err := e.updateAndSave(next); err != nil {
			e.report("error", "lobby_save", err.Error())
			return
		}
		*s = next
	}
	if ctx.Err() != nil {
		return
	}
	if s.Active != nil {
		if intent := s.Active.Settlement; intent != nil {
			if err := e.doSettle(ctx, s, intent.Outcome, intent.Source); err != nil {
				e.setReplayScan("settle_error", "", err.Error())
				return
			}
		} else if now.Sub(time.UnixMilli(s.Active.StartedAt)) > 100*time.Minute {
			if err := e.doSettle(ctx, s, "cancel", "timeout"); err != nil {
				e.setReplayScan("settle_error", "", err.Error())
				return
			}
		} else {
			e.scanReplay(ctx, s, now, lastError, lastLog)
		}
	}
	// An unconfirmed remote POST may have already created a live wager.
	// Do not create ANY other Prediction until the streamer has reconciled
	// the uncertain operation with Twitch, even when a newer lobby arrives.
	if s.UnconfirmedCreation != nil && s.Active == nil {
		e.setReplayScan("create_error", "", "Verify the unconfirmed Prediction on Twitch before resuming automation")
		return
	}
	if s.Active != nil || s.PendingLobby == nil || ctx.Err() != nil {
		return
	}
	event := s.PendingLobby
	next := *s
	popPendingLobby(&next)
	if event.Snapshot == nil || now.Sub(time.UnixMilli(event.Snapshot.ModifiedAt)) > 120*time.Second || time.UnixMilli(event.Snapshot.ModifiedAt).After(now) {
		if err := e.updateAndSave(next); err == nil {
			*s = next
		}
		return
	}
	// Keep the lobby pending while read-only Twitch checks fail. Once the
	// creation attempt is persisted, a POST failure remains non-retryable.
	broadcasterID := ""
	if e.cfg.RealMode {
		me, err := e.predictions.preflightCreate(ctx)
		if err != nil {
			e.setReplayScan("create_error", "", err.Error())
			return
		}
		broadcasterID = me.UserID
	}
	next, err = attemptedCreationState(*s, event, broadcasterID, e.cfg.RealMode, e.clock.Now())
	if err != nil {
		e.report("error", "prediction_create", err.Error())
		return
	}
	if err := e.updateAndSave(next); err != nil {
		e.report("error", "lobby_save", err.Error())
		return
	}
	*s = next
	if err := e.doCreatePrepared(ctx, s, event.Key, broadcasterID); err != nil {
		e.setReplayScan("create_error", "", err.Error())
		e.report("error", "prediction_create", tr(e.cfg.lang(), "lifecycle.create_unconfirmed", err.Error()))
		return
	}
	e.log(tr(e.cfg.lang(), "log.new_game", event.Snapshot.File))
}
