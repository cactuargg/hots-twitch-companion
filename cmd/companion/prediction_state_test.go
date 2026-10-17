package main

import (
	"reflect"
	"testing"
	"time"
)

func TestPurePredictionTransitionsDoNotMutatePriorSnapshots(t *testing.T) {
	now := time.Unix(1729000000, 0)
	first := &LobbyEvent{Key: "A", Snapshot: &LobbySnapshot{SHA256: "A", ModifiedAt: now.UnixMilli()}}
	second := &LobbyEvent{Key: "B", Snapshot: &LobbySnapshot{SHA256: "B", ModifiedAt: now.UnixMilli()}}
	input := State{PendingLobby: first, PendingLobbies: []*LobbyEvent{second}, RecentLobbySHAs: []string{"prior"}}
	previous := append([]string(nil), input.RecentLobbySHAs...)
	attempted, err := attemptedCreationState(input, first, "channel", true, now)
	if err != nil { t.Fatal(err) }
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
	p := &Prediction{ID: "created", BroadcasterID: "channel", WinID: "w", LossID: "l"}
	confirmed, err := confirmedCreationState(attempted, p, &MatchBinding{Version: 1})
	if err != nil { t.Fatal(err) }
	if confirmed.Active == nil || confirmed.Active.ID != "created" || confirmed.UnconfirmedCreation != nil ||
		p.Match != nil || attempted.Active != nil || attempted.UnconfirmedCreation == nil {
		t.Fatalf("confirmation mutated previous state or lost remote id: %+v / %+v", attempted, confirmed)
	}
	intent, err := pendingSettlementState(confirmed, "win", "manual")
	if err != nil {t.Fatal(err)}
	if confirmed.Active.Settlement != nil || intent.Active.Settlement == nil ||
		intent.Active.Settlement.Outcome != "win" {
		t.Fatalf("intent was not immutable: %+v / %+v",confirmed,intent)
	}
	retry := retrySettlementState(intent, now, now.Add(time.Minute).UnixMilli())
	if retry.Active.Settlement.RetryAt != now.Add(time.Minute).UnixMilli() ||
		retry.Active.Settlement.Failures != 1 || intent.Active.Settlement.Failures != 0 ||
		retry.Active.Settlement.Outcome != "win" || retry.Active.Settlement.Source != "manual" {
		t.Fatalf("retry overwrote chosen outcome or old snapshot: %+v / %+v",retry,intent)
	}
	done,err := completedPredictionState(retry,"win","manual",now.Add(2*time.Minute))
	if err!=nil {t.Fatal(err)}
	if done.Active!=nil || done.LastSettlementAt!=now.Add(2*time.Minute).UnixMilli() ||
		retry.Active==nil || done.LastResult!="win (manual)" {
		t.Fatalf("completion invalid: %+v",done)
	}
}

func TestPredictionStateRejectsUnsafeTransitions(t *testing.T) {
	event := &LobbyEvent{Key:"a",Snapshot:&LobbySnapshot{SHA256:"a"}}
	s := State{PendingLobby:event}
	if _,err := attemptedCreationState(s,event,"",true,time.Now());err==nil {
		t.Fatal("real creation without a verified channel allowed")
	}
	if _,err := attemptedCreationState(State{Active:&Prediction{ID:"old"},PendingLobby:event},event,"channel",true,time.Now());err==nil {
		t.Fatal("new prediction allowed while one is active")
	}
	if _,err := attemptedCreationState(State{UnconfirmedCreation:&CreationAttempt{},PendingLobby:event},event,"channel",true,time.Now());err==nil {
		t.Fatal("new prediction allowed during unconfirmed POST")
	}
	if _,err := attemptedCreationState(State{PendingLobby:event},&LobbyEvent{Snapshot:&LobbySnapshot{SHA256:"other"}},"channel",true,time.Now());err==nil {
		t.Fatal("attempted wrong pending lobby")
	}
	if _,err := confirmedCreationState(State{UnconfirmedCreation:&CreationAttempt{BroadcasterID:"A"}},&Prediction{ID:"new",BroadcasterID:"B"},nil);err==nil {
		t.Fatal("confirmed mismatching channel")
	}
	if _,err := confirmedCreationState(State{Active:&Prediction{ID:"old"}},&Prediction{ID:"new"},nil);err==nil {
		t.Fatal("overwrote active Prediction")
	}
	if _,err := pendingSettlementState(State{Active:&Prediction{ID:"p"}},"unknown","manual");err==nil {
		t.Fatal("accepted unknown outcome")
	}
	intent,_ := pendingSettlementState(State{Active:&Prediction{ID:"p"}},"win","manual")
	if _,err := pendingSettlementState(intent,"loss","manual");err==nil {
		t.Fatal("changed latched settlement decision")
	}
	if _,err := completedPredictionState(State{},"win","manual",time.Now());err==nil {
		t.Fatal("completed without an active Prediction")
	}
}
