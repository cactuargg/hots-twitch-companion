package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"
)

type SettlementAttempt struct {
	Outcome  string `json:"outcome"`
	Source   string `json:"source"`
	Failures int    `json:"failures,omitempty"`
	RetryAt  int64  `json:"retryAt,omitempty"`
}

type twitchHTTPError struct {
	Status  int
	Message string
	RetryAt int64
}

func (e *twitchHTTPError) Error() string {
	return fmt.Sprintf("Twitch HTTP %d: %s", e.Status, e.Message)
}
func responseHTTPError(response *http.Response, message string) error {
	failure := &twitchHTTPError{Status: response.StatusCode, Message: message}
	if response.StatusCode == 429 {
		if seconds, err := strconv.ParseInt(response.Header.Get("Retry-After"), 10, 64); err == nil && seconds >= 0 && seconds < 86400 {
			failure.RetryAt = time.Now().Add(time.Duration(seconds) * time.Second).UnixMilli()
		} else if at, err := http.ParseTime(response.Header.Get("Retry-After")); err == nil {
			failure.RetryAt = at.UnixMilli()
		}
		if reset, err := strconv.ParseInt(response.Header.Get("Ratelimit-Reset"), 10, 64); err == nil && reset > 0 && reset < 1<<40 {
			failure.RetryAt = max(failure.RetryAt, reset*1000)
		}
	}
	return failure
}

type LobbyEvent struct {
	Key      string         `json:"key"`
	Snapshot *LobbySnapshot `json:"snapshot"`
}

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

type remoteFinishedError struct{ Prediction apiPrediction }

func (e *remoteFinishedError) Error() string { return "Prediction was already finalized on Twitch" }

func exactPrediction(response *apiResponse, id string) (apiPrediction, error) {
	if response == nil || len(response.Data) != 1 || response.Data[0].ID != id {
		return apiPrediction{}, errors.New("Twitch did not return the exact Prediction")
	}
	return response.Data[0], nil
}

func terminalPrediction(actual apiPrediction, status, winner string) error {
	if actual.Status == status && (status == "CANCELED" || actual.WinningOutcomeID == winner) {
		return nil
	}
	if actual.Status == "CANCELED" || actual.Status == "RESOLVED" {
		return &remoteFinishedError{actual}
	}
	return errors.New("Twitch has not confirmed the requested final status")
}

// Read-before-write and exact response validation apply to first attempts and
// recovery alike. A lost response is never grounds to choose another outcome.
func (t *Twitch) finishPrediction(ctx context.Context, p *Prediction, outcome string) error {
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
	response, err := t.prediction(ctx, p.BroadcasterID, p.ID)
	if err != nil {
		return err
	}
	actual, err := exactPrediction(response, p.ID)
	if err != nil {
		return err
	}
	if actual.Status == "RESOLVED" || actual.Status == "CANCELED" {
		return terminalPrediction(actual, status, winner)
	}
	if actual.Status != "ACTIVE" && actual.Status != "LOCKED" {
		return errors.New("unsupported Twitch Prediction status")
	}
	body := map[string]any{"broadcaster_id": p.BroadcasterID, "id": p.ID, "status": status}
	if winner != "" {
		body["winning_outcome_id"] = winner
	}
	var result apiResponse
	requestErr := t.req(ctx, "PATCH", "/predictions", body, &result)
	if requestErr == nil {
		confirmed, err := exactPrediction(&result, p.ID)
		if err == nil {
			err = terminalPrediction(confirmed, status, winner)
		}
		if err == nil {
			return nil
		}
		// A contradictory PATCH body is not a verified external finalization.
		// Only the follow-up GET may supply remoteFinishedError on this path.
		requestErr = fmt.Errorf("unexpected Twitch response: %v", err)
	}
	// This request may have committed even if PATCH timed out or its body was bad.
	response, err = t.prediction(ctx, p.BroadcasterID, p.ID)
	if err == nil {
		actual, err = exactPrediction(response, p.ID)
		if err == nil && (actual.Status == "RESOLVED" || actual.Status == "CANCELED") {
			return terminalPrediction(actual, status, winner)
		}
	}
	return requestErr
}

func remoteOutcome(actual apiPrediction, p *Prediction) string {
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

func (e *Engine) commitSettlement(s *State, outcome, source string) error {
	next := *s
	next.LastSettlementAt = time.Now().UnixMilli()
	next.LastResult = fmt.Sprintf("%s (%s)", outcome, filepath.Base(source))
	next.Active = nil
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
		next := *s
		active := *s.Active
		active.Settlement = &SettlementAttempt{Outcome: outcome, Source: source}
		next.Active = &active
		if err := e.updateAndSave(next); err != nil {
			return err
		}
		*s = next
	}
	intent := s.Active.Settlement
	if intent.RetryAt > time.Now().UnixMilli() {
		return errors.New(tr(e.cfg.lang(), "settle.retry_wait"))
	}
	err := e.twitch.end(ctx, s.Active, intent.Outcome)
	var finished *remoteFinishedError
	if errors.As(err, &finished) {
		return e.commitSettlement(s, remoteOutcome(finished.Prediction, s.Active), "Twitch")
	}
	if err != nil {
		next := *s
		active := *s.Active
		retry := *intent
		retry.Failures++
		retry.RetryAt = time.Now().Add(time.Duration(min(retry.Failures, 6)) * 10 * time.Second).UnixMilli()
		var rateLimit *twitchHTTPError
		if errors.As(err, &rateLimit) {
			retry.RetryAt = max(retry.RetryAt, rateLimit.RetryAt)
		}
		active.Settlement = &retry
		next.Active = &active
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
		next.PendingLobby = nil
		if next.Active != nil {
			next.Active = observeActiveLobby(next.Active, snapshot, key)
		}
		fresh := !modified.Before(started.Add(-2*time.Second)) && now.Sub(modified) < 120*time.Second
		sameActive := next.Active != nil && next.Active.Match != nil && next.Active.Match.Lobby != nil && next.Active.Match.Lobby.SHA256 == snapshot.SHA256
		if fresh && snapshot.Status != "read_error" && !sameActive && !lobbyWasHandled(&next, snapshot.SHA256) {
			next.PendingLobby = &LobbyEvent{Key: key, Snapshot: snapshot}
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
	if s.Active != nil || s.PendingLobby == nil || ctx.Err() != nil {
		return
	}
	event := s.PendingLobby
	next := *s
	next.PendingLobby = nil
	if event.Snapshot == nil || now.Sub(time.UnixMilli(event.Snapshot.ModifiedAt)) > 120*time.Second || time.UnixMilli(event.Snapshot.ModifiedAt).After(now) {
		if err := e.updateAndSave(next); err == nil {
			*s = next
		}
		return
	}
	next.LastCreatedLobbySHA = event.Snapshot.SHA256
	if event.Snapshot.SHA256 != "" {
		next.RecentLobbySHAs = append(append([]string(nil), s.RecentLobbySHAs...), event.Snapshot.SHA256)
		if len(next.RecentLobbySHAs) > 64 {
			next.RecentLobbySHAs = next.RecentLobbySHAs[len(next.RecentLobbySHAs)-64:]
		}
	}
	next.LastLobby = event.Snapshot
	if err := e.updateAndSave(next); err != nil {
		e.report("error", "lobby_save", err.Error())
		return
	}
	*s = next
	if err := e.doCreate(ctx, s, event.Key); err != nil {
		e.setReplayScan("create_error", "", err.Error())
		e.report("error", "prediction_create", tr(e.cfg.lang(), "lifecycle.create_unconfirmed", err.Error()))
		return
	}
	e.log(tr(e.cfg.lang(), "log.new_game", event.Snapshot.File))
}
