package app

import (
	"context"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/hots/lobby"
	"hots-twitch-companion/internal/prediction"
	"hots-twitch-companion/internal/twitch"
	"net/url"
	"testing"
	"time"
)

func TestSnapshotDoesNotExposeMutablePredictionEvidence(t *testing.T) {
	e, err := New(config.NewPaths(t.TempDir()), config.Data{}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	e.state = prediction.State{
		Active:         &prediction.Prediction{ID: "keep", Settlement: &prediction.SettlementAttempt{Outcome: "win"}, Match: &prediction.MatchBinding{Lobby: &lobby.Snapshot{Data: &lobby.BattleLobby{Players: []lobby.LobbyPlayer{{ToonHandle: "keep"}}}}}},
		PendingLobbies: []*prediction.LobbyEvent{{Key: "queued"}}, RecentLobbySHAs: []string{"seen"},
	}
	_, _, s, _ := e.Snapshot()
	s.Active.ID = "changed"
	s.Active.Settlement.Outcome = "loss"
	s.Active.Match.Lobby.Data.Players[0].ToonHandle = "changed"
	s.PendingLobbies[0].Key = "changed"
	s.RecentLobbySHAs[0] = "changed"
	_, _, after, _ := e.Snapshot()
	if after.Active.ID != "keep" || after.Active.Settlement.Outcome != "win" || after.Active.Match.Lobby.Data.Players[0].ToonHandle != "keep" || after.PendingLobbies[0].Key != "queued" || after.RecentLobbySHAs[0] != "seen" {
		t.Fatal("consumer modified Engine state through its snapshot")
	}
}

func TestEngineInstancesKeepInjectedTransportsAndOAuthDefaults(t *testing.T) {
	for _, id := range []string{"first", "second"} {
		request := func(_ context.Context, _, endpoint string, _ url.Values, _ any, _, clientID string, dst any) error {
			if endpoint != twitch.AuthURL+"/validate" {
				t.Fatalf("unexpected request %s", endpoint)
			}
			*dst.(*twitch.Identity) = twitch.Identity{ClientID: id, UserID: id, Scopes: []string{twitch.Scope}}
			return nil
		}
		e, err := New(config.NewPaths(t.TempDir()), config.Data{}, Options{DefaultClientID: id, Request: request})
		if err != nil {
			t.Fatal(err)
		}
		if err := e.twitch.AcceptValidatedToken(twitch.Token{AccessToken: id, ExpiresIn: 3600, SavedAt: time.Now().Unix()}, twitch.Identity{}); err != nil {
			t.Fatal(err)
		}
		who, err := e.Identity(context.Background())
		if err != nil || who.UserID != id || e.ClientID() != id {
			t.Fatalf("instance dependencies mixed: %+v %v", who, err)
		}
	}
}
