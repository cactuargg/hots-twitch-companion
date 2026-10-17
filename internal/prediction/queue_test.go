package prediction

import (
 "fmt"
 "testing"
 "hots-twitch-companion/internal/hots/lobby"
)

func TestLobbyQueueFIFOAndDuplicateSuppression(t *testing.T){
 s:=State{LastCreatedLobbySHA:"settled",RecentLobbySHAs:[]string{"old"}}
 makeEvent:=func(sha string)*LobbyEvent{return &LobbyEvent{Key:sha,Snapshot:&lobby.Snapshot{SHA256:sha}}}
 if EnqueueLobby(&s,makeEvent("old"))||EnqueueLobby(&s,makeEvent("settled")){t.Fatal("replayed lobby was requeued")}
 if !EnqueueLobby(&s,makeEvent("a"))||!EnqueueLobby(&s,makeEvent("b")){t.Fatal("distinct lobby not queued")}
 if EnqueueLobby(&s,makeEvent("b"))||EnqueueLobby(&s,makeEvent("a")){t.Fatal("duplicate lobby requeued")}
 if s.PendingLobby.Key!="a"||len(s.PendingLobbies)!=1||s.PendingLobbies[0].Key!="b"{t.Fatalf("incorrect FIFO state: %+v",s)}
 prev:=append([]*LobbyEvent(nil),s.PendingLobbies...)
 PopPendingLobby(&s)
 if s.PendingLobby.Key!="b"||len(s.PendingLobbies)!=0||len(prev)!=1||prev[0].Key!="b"{t.Fatal("queue pop violated persistence order")}
 PopPendingLobby(&s)
 if s.PendingLobby!=nil{t.Fatal("empty queue retained head")}
}

func TestLobbyQueueBoundAndNoSharedSliceMutation(t *testing.T){
 s:=State{}
 makeEvent:=func(i int)*LobbyEvent{return &LobbyEvent{Key:fmt.Sprint(i),Snapshot:&lobby.Snapshot{SHA256:fmt.Sprint(i)}}}
 for i:=0;i<MaxPendingLobbies;i++{if !EnqueueLobby(&s,makeEvent(i)){t.Fatalf("rejected entry %d",i)}}
 if EnqueueLobby(&s,makeEvent(MaxPendingLobbies)){t.Fatal("unbounded pending queue")}
 if len(s.PendingLobbies)!=MaxPendingLobbies-1{t.Fatalf("wrong size %d",len(s.PendingLobbies))}
 old:=s
 oldList:=append([]*LobbyEvent(nil),s.PendingLobbies...)
 PopPendingLobby(&s)
 if old.PendingLobby.Key!="0"||len(old.PendingLobbies)!=MaxPendingLobbies-1{
  t.Fatalf("old state mutated: %+v",old)
 }
 if oldList[0].Key!="1"||s.PendingLobby.Key!="1"{t.Fatal("first-in-first-out invariant broke")}
}
