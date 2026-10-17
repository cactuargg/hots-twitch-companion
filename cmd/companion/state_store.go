package main

import (
 "hots-twitch-companion/internal/storage"
)

// StateStore stays as a transitional alias for Engine tests and callers.
type StateStore = storage.StateStore

// This adapter deliberately keeps the reference to Engine paths: tests that
// redirect e.p.state still exercise the production StateStore.
type fileStateStore struct{p *paths}
func (s fileStateStore) Load()(State,error) {
 return storage.FileStateStore{Path:s.p.state}.Load()
}
func (s fileStateStore) Save(state State)error {
 return storage.FileStateStore{Path:s.p.state}.Save(state)
}
