package storage

import "hots-twitch-companion/internal/prediction"

// StateStore is the durable boundary of the Prediction state machine.
// A failed Save prevents the Engine from sending irreversible Twitch writes.
type StateStore interface {
 Load()(prediction.State,error)
 Save(prediction.State)error
}

// FileStateStore persists Prediction state with the existing atomic JSON
// writer and retains the exact legacy filename chosen by its caller.
type FileStateStore struct {Path string}
func (s FileStateStore) Load()(prediction.State,error){
 var state prediction.State
 err:=LoadJSON(s.Path,&state)
 return state,err
}
func (s FileStateStore) Save(state prediction.State)error{
 return SaveJSON(s.Path,state)
}
var _ StateStore = FileStateStore{}
