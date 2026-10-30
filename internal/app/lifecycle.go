package app

import (
	"context"
	"errors"
	"fmt"
	"hots-twitch-companion/internal/hots/correlation"
	"hots-twitch-companion/internal/i18n"
	"hots-twitch-companion/internal/prediction"
	twitchpkg "hots-twitch-companion/internal/twitch"
	"hots-twitch-companion/internal/twitch/transport"
	"time"
)

func (e *Engine) commitSettlement(s *prediction.State, outcome, source string) error {
	next, err := prediction.Completed(*s, outcome, source, e.clock.Now())
	if err != nil {
		return err
	}
	if err := e.updateAndSave(next); err != nil {
		return err
	}
	*s = next
	return nil
}

func (e *Engine) settlePrediction(ctx context.Context, s *prediction.State, outcome, source string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Active == nil {
		return errors.New(i18n.Text(e.cfg.Language(), "settle.none_active"))
	}
	if outcome != "win" && outcome != "loss" && outcome != "cancel" {
		return errors.New(i18n.Text(e.cfg.Language(), "manual.invalid"))
	}
	if s.Active.Settlement != nil && s.Active.Settlement.Outcome != outcome {
		return errors.New(i18n.Text(e.cfg.Language(), "settle.pending_conflict"))
	}
	if s.Active.DryRun {
		return e.commitSettlement(s, outcome, source)
	}
	if s.Active.Settlement == nil {
		next, err := prediction.PendingSettlement(*s, outcome, source)
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
		return errors.New(i18n.Text(e.cfg.Language(), "settle.retry_wait"))
	}
	err := e.predictions.End(ctx, s.Active, intent.Outcome)
	if errors.Is(err, twitchpkg.ErrZeroPoints) {
		// End has verified that no resolution PATCH was sent. Latch the refund
		// before trying it, so a network failure cannot revert to paying out.
		next := *s
		active := *s.Active
		refund := *intent
		refund.Outcome = "cancel"
		active.Settlement = &refund
		next.Active = &active
		if saveErr := e.updateAndSave(next); saveErr != nil {
			return saveErr
		}
		*s = next
		intent = s.Active.Settlement
		err = e.predictions.End(ctx, s.Active, intent.Outcome)
	}
	var finished *twitchpkg.RemoteFinishedError
	if errors.As(err, &finished) {
		return e.commitSettlement(s, twitchpkg.RemoteOutcome(finished.Prediction, s.Active), "Twitch")
	}
	if err != nil {
		minRetryAt := int64(0)
		var rateLimit *transport.HTTPError
		if errors.As(err, &rateLimit) {
			minRetryAt = rateLimit.RetryAt
		}
		next := prediction.RetrySettlement(*s, e.clock.Now(), minRetryAt)
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
func (e *Engine) scanLifecycle(ctx context.Context, s *prediction.State, started, now time.Time, lastError *string, lastLog *time.Time) {
	path, key, modified, err := e.source.FindLobby()
	if err == nil && now.Sub(modified) > 2500*time.Millisecond && (key != s.LastLobbyKey || s.LastLobby != nil && s.LastLobby.Status == "read_error") {
		snapshot := e.source.Capture(path)
		if snapshot.ModifiedAt > 0 && (snapshot.ModifiedAt != modified.UnixMilli() || now.Sub(time.UnixMilli(snapshot.ModifiedAt)) < 2500*time.Millisecond) {
			return
		}
		next := *s
		next.LastLobby = snapshot
		next.LastLobbyKey = key
		if next.Active != nil {
			next.Active = correlation.ObserveActiveLobby(next.Active, snapshot, key)
		}
		fresh := !modified.Before(started.Add(-2*time.Second)) && now.Sub(modified) < 120*time.Second
		sameActive := next.Active != nil && next.Active.Match != nil && next.Active.Match.Lobby != nil && next.Active.Match.Lobby.SHA256 == snapshot.SHA256
		if fresh && snapshot.Status != "read_error" && !sameActive && !prediction.LobbyWasHandled(&next, snapshot.SHA256) {
			if !prediction.EnqueueLobby(&next, &prediction.LobbyEvent{Key: key, Snapshot: snapshot}) &&
				!prediction.PendingLobbySeen(&next, snapshot.SHA256) && !prediction.LobbyWasHandled(&next, snapshot.SHA256) {
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
	prediction.PopPendingLobby(&next)
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
		me, err := e.predictions.PreflightCreate(ctx)
		if err != nil {
			e.setReplayScan("create_error", "", err.Error())
			return
		}
		broadcasterID = me.UserID
	}
	next, err = prediction.AttemptedCreation(*s, event, broadcasterID, e.cfg.RealMode, e.clock.Now())
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
		e.report("error", "prediction_create", i18n.Text(e.cfg.Language(), "lifecycle.create_unconfirmed", err.Error()))
		return
	}
	e.log(i18n.Text(e.cfg.Language(), "log.new_game", event.Snapshot.File))
}
