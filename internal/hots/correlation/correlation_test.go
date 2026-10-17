package correlation

import (
 "testing"
 "hots-twitch-companion/internal/hots/lobby"
 "hots-twitch-companion/internal/prediction"
)

func TestMatchBindingOwnsDetachedLobbyRoster(t *testing.T) {
 original:=&lobby.Snapshot{SHA256:"sha",Data:&lobby.BattleLobby{
  Players:[]lobby.LobbyPlayer{{ToonHandle:"1-Hero-1-42"}},
 }}
 binding:=BindPredictionMatch(original,"key"," 1-Hero-1-42 ")
 if binding.Lobby==original || binding.Lobby.Data==original.Data ||
  binding.PlayerToon!="1-Hero-1-42"{t.Fatalf("binding did not copy snapshot: %+v",binding)}
 original.Data.Players[0].ToonHandle="replaced"
 if binding.Lobby.Data.Players[0].ToonHandle!="1-Hero-1-42"{
  t.Fatal("binding roster changed after snapshot mutation")
 }
}

func TestUnknownRosterAndIdentityRejected(t *testing.T) {
 for _,scenario:=range []*lobby.BattleLobby{nil,{TeamSize:"1v1"},{TeamSize:"5v5",AMMID:424242}} {
  if _,err:=ParticipantToons(scenario);err==nil{t.Fatalf("unsupported roster accepted: %+v",scenario)}
 }
}

func TestObserveDuplicateLobbyMarksMatchAmbiguousWithoutMutation(t *testing.T) {
 active:=&prediction.Prediction{ID:"remote",Match:&prediction.MatchBinding{LobbyKey:"old",Lobby:&lobby.Snapshot{SHA256:"abc"}}}
 observed:=ObserveActiveLobby(active,&lobby.Snapshot{SHA256:"abc"},"new")
 if observed==active || observed.Match==active.Match || !observed.Match.Ambiguous {
  t.Fatalf("ambiguous lobby not detected: %+v",observed)
 }
 if active.Match.Ambiguous {t.Fatal("previous state mutated")}
}
