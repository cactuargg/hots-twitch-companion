package prediction

// The first pending lobby is kept in PendingLobby for backward-compatible
// state serialization. PendingLobbies contains the bounded FIFO remainder.
func LobbyWasHandled(s *State, hash string) bool {
	if hash == "" {
		return false
	}
	if hash == s.LastCreatedLobbySHA {
		return true
	}
	for _, previous := range s.RecentLobbySHAs {
		if previous == hash {
			return true
		}
	}
	return false
}

// PendingLobby is the oldest unconsumed event for compatibility with
// previously saved state. PendingLobbies holds the remaining FIFO entries.
const MaxPendingLobbies = 32

func PendingLobbySeen(s *State, sha string) bool {
	if sha == "" {
		return false
	}
	if s.PendingLobby != nil && s.PendingLobby.Snapshot != nil && s.PendingLobby.Snapshot.SHA256 == sha {
		return true
	}
	for _, pending := range s.PendingLobbies {
		if pending != nil && pending.Snapshot != nil && pending.Snapshot.SHA256 == sha {
			return true
		}
	}
	return false
}

func EnqueueLobby(s *State, event *LobbyEvent) bool {
	if event == nil || event.Snapshot == nil || event.Snapshot.SHA256 == "" ||
		LobbyWasHandled(s, event.Snapshot.SHA256) || PendingLobbySeen(s, event.Snapshot.SHA256) {
		return false
	}
	if s.PendingLobby == nil {
		s.PendingLobby = event
		return true
	}
	if len(s.PendingLobbies) >= MaxPendingLobbies-1 {
		return false
	}
	// Clone to keep the previously persisted State snapshot immutable.
	s.PendingLobbies = append(append([]*LobbyEvent(nil), s.PendingLobbies...), event)
	return true
}



