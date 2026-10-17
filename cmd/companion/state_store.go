package main

// StateStore is the durable boundary for the Prediction lifecycle. The caller
// commits state before irreversible Twitch requests; Save must report failures
// rather than making an operation appear persisted.
type StateStore interface {
	Load() (State, error)
	Save(State) error
}

type fileStateStore struct {
	p *paths
}

func (s fileStateStore) Load() (State, error) {
	return loadState(*s.p)
}

func (s fileStateStore) Save(state State) error {
	return saveState(*s.p, state)
}
