package app

import (
	lobbydata "hots-twitch-companion/internal/hots/lobby"
	predmodel "hots-twitch-companion/internal/prediction"
	"reflect"
	"testing"
	"time"
)

func TestPurePredictionTransitionsDoNotMutatePriorSnapshots(t *testing.T) {
	now := time.Unix(1729000000, 0)
	first := &predmodel.LobbyEvent{Key: "A", Snapshot: &lobbydata.Snapshot{SHA256: "A", ModifiedAt: now.UnixMilli()}}
	second := &predmodel.LobbyEvent{Key: "B", Snapshot: &lobbydata.Snapshot{SHA256: "B", ModifiedAt: now.UnixMilli()}}
	input := predmodel.State{PendingLobby: first, PendingLobbies: []*predmodel.LobbyEvent{second}, RecentLobbySHAs: []string{"prior"}}
	previous := append([]string(nil), input.RecentLobbySHAs...)
	attempted, err := predmodel.AttemptedCreation(input, first, "channel", true, now)
	if err != nil {
		t.Fatal(err)
	}
	if input.PendingLobby != first || len(input.PendingLobbies) != 1 ||
		!reflect.DeepEqual(input.RecentLobbySHAs, previous) {
		t.Fatalf("transition mutated original state: %+v", input)
	}
	if attempted.PendingLobby != second || len(attempted.PendingLobbies) != 0 ||
		attempted.UnconfirmedCreation == nil || attempted.UnconfirmedCreation.LobbySHA != "A" ||
		attempted.UnconfirmedCreation.AttemptedAt != now.UnixMilli() ||
		attempted.LastCreatedLobbySHA != "A" {
		t.Fatalf("attempted state incorrect: %+v", attempted)
	}
	p := &predmodel.Prediction{ID: "created", BroadcasterID: "channel", WinID: "w", LossID: "l"}
	confirmed, err := predmodel.ConfirmedCreation(attempted, p, &predmodel.MatchBinding{Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.Active == nil || confirmed.Active.ID != "created" || confirmed.UnconfirmedCreation != nil ||
		p.Match != nil || attempted.Active != nil || attempted.UnconfirmedCreation == nil {
		t.Fatalf("confirmation mutated previous state or lost remote id: %+v / %+v", attempted, confirmed)
	}
	intent, err := predmodel.PendingSettlement(confirmed, "win", "manual")
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.Active.Settlement != nil || intent.Active.Settlement == nil ||
		intent.Active.Settlement.Outcome != "win" {
		t.Fatalf("intent was not immutable: %+v / %+v", confirmed, intent)
	}
	retry := predmodel.RetrySettlement(intent, now, now.Add(time.Minute).UnixMilli())
	if retry.Active.Settlement.RetryAt != now.Add(time.Minute).UnixMilli() ||
		retry.Active.Settlement.Failures != 1 || intent.Active.Settlement.Failures != 0 ||
		retry.Active.Settlement.Outcome != "win" || retry.Active.Settlement.Source != "manual" {
		t.Fatalf("retry overwrote chosen outcome or old snapshot: %+v / %+v", retry, intent)
	}
	done, err := predmodel.Completed(retry, "win", "manual", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if done.Active != nil || done.LastSettlementAt != now.Add(2*time.Minute).UnixMilli() ||
		retry.Active == nil || done.LastResult != "win (manual)" {
		t.Fatalf("completion invalid: %+v", done)
	}
}

func TestPredictionStateRejectsUnsafeTransitions(t *testing.T) {
	event := &predmodel.LobbyEvent{Key: "a", Snapshot: &lobbydata.Snapshot{SHA256: "a"}}
	s := predmodel.State{PendingLobby: event}
	if _, err := predmodel.AttemptedCreation(s, event, "", true, time.Now()); err == nil {
		t.Fatal("real creation without a verified channel allowed")
	}
	if _, err := predmodel.AttemptedCreation(predmodel.State{Active: &predmodel.Prediction{ID: "old"}, PendingLobby: event}, event, "channel", true, time.Now()); err == nil {
		t.Fatal("new prediction allowed while one is active")
	}
	if _, err := predmodel.AttemptedCreation(predmodel.State{UnconfirmedCreation: &predmodel.CreationAttempt{}, PendingLobby: event}, event, "channel", true, time.Now()); err == nil {
		t.Fatal("new prediction allowed during unconfirmed POST")
	}
	if _, err := predmodel.AttemptedCreation(predmodel.State{PendingLobby: event}, &predmodel.LobbyEvent{Snapshot: &lobbydata.Snapshot{SHA256: "other"}}, "channel", true, time.Now()); err == nil {
		t.Fatal("attempted wrong pending lobby")
	}
	if _, err := predmodel.ConfirmedCreation(predmodel.State{UnconfirmedCreation: &predmodel.CreationAttempt{BroadcasterID: "A"}}, &predmodel.Prediction{ID: "new", BroadcasterID: "B"}, nil); err == nil {
		t.Fatal("confirmed mismatching channel")
	}
	if _, err := predmodel.ConfirmedCreation(predmodel.State{Active: &predmodel.Prediction{ID: "old"}}, &predmodel.Prediction{ID: "new"}, nil); err == nil {
		t.Fatal("overwrote active Prediction")
	}
	if _, err := predmodel.PendingSettlement(predmodel.State{Active: &predmodel.Prediction{ID: "p"}}, "unknown", "manual"); err == nil {
		t.Fatal("accepted unknown outcome")
	}
	intent, _ := predmodel.PendingSettlement(predmodel.State{Active: &predmodel.Prediction{ID: "p"}}, "win", "manual")
	if _, err := predmodel.PendingSettlement(intent, "loss", "manual"); err == nil {
		t.Fatal("changed latched settlement decision")
	}
	if _, err := predmodel.Completed(predmodel.State{}, "win", "manual", time.Now()); err == nil {
		t.Fatal("completed without an active Prediction")
	}
}
