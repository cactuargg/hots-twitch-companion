package prediction

import (
  "encoding/json"
  "testing"
  "time"
  "hots-twitch-companion/internal/hots/lobby"
)

func TestPersistedJSONFieldNamesUnchanged(t *testing.T) {
  at:=time.Date(2026,10,17,14,0,0,0,time.UTC)
  event:=&LobbyEvent{Key:"lobby",Snapshot:&lobby.Snapshot{SHA256:"sha",File:"replay.server.battlelobby"}}
  following:=&LobbyEvent{Key:"following",Snapshot:&lobby.Snapshot{SHA256:"next"}}
  s,err:=AttemptedCreation(State{PendingLobby:event,PendingLobbies:[]*LobbyEvent{following}},event,"broadcaster",true,at)
  if err!=nil {t.Fatal(err)}
  raw,err:=json.Marshal(s)
  if err!=nil{t.Fatal(err)}
  var decoded map[string]json.RawMessage
  if err:=json.Unmarshal(raw,&decoded);err!=nil{t.Fatal(err)}
  for _,name:=range []string{"pendingLobby","active","unconfirmedCreation","lastCreatedLobbySHA","recentLobbySHAs","lastLobby"}{
    if _,ok:=decoded[name];!ok {t.Fatalf("missing persisted JSON field %q in %s",name,raw)}
  }
  var restored State
  if err:=json.Unmarshal(raw,&restored);err!=nil{t.Fatal(err)}
  if restored.UnconfirmedCreation==nil || restored.UnconfirmedCreation.BroadcasterID!="broadcaster" || restored.LastLobby.SHA256!="sha" {
    t.Fatalf("state roundtrip changed: %+v",restored)
  }
}
