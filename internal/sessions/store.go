// Package sessions owns the durable match journal independently of Twitch.
package sessions

import (
	"encoding/json"
	"errors"
	"fmt"
	"hots-twitch-companion/internal/storage"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"
)

type Preferences struct {
	LobbyWidgetEnabled bool         `json:"lobbyWidgetEnabled"`
	ScoreThemes        []ScoreTheme `json:"scoreThemes,omitempty"`
	CustomCSS          string       `json:"customCSS,omitempty"`
	Enabled            bool         `json:"enabled"`
	RankedOnly         bool         `json:"rankedOnly"`
	Style              string       `json:"style"`
	Scale              int          `json:"scale"`
	Backdrop           bool         `json:"backdrop"`
	WinColor           string       `json:"winColor"`
	LossColor          string       `json:"lossColor"`
	Labels             bool         `json:"labels"`
	MaskEnabled        bool         `json:"maskEnabled"`
	MaskMode           string       `json:"maskMode"`
	MaskRankedOnly     bool         `json:"maskRankedOnly"`
	MaskImage          string       `json:"maskImage,omitempty"`
}

type ScoreTheme struct {
	Name string `json:"name"`
	CSS  string `json:"css"`
}
type Session struct {
	ID        string `json:"id"`
	Player    string `json:"player"`
	StartedAt int64  `json:"startedAt"`
	EndedAt   int64  `json:"endedAt,omitempty"`
}
type Match struct {
	ID              string        `json:"id"`
	Player          string        `json:"player"`
	SessionID       string        `json:"sessionId"`
	Map             string        `json:"map"`
	Hero            string        `json:"hero"`
	HeroSource      string        `json:"heroSource,omitempty"`
	Mode            uint32        `json:"mode"`
	Result          string        `json:"result"`
	OriginalResult  string        `json:"originalResult"`
	RecordedAt      int64         `json:"recordedAt"`
	StartedAt       int64         `json:"startedAt,omitempty"`
	DiscoveredAt    int64         `json:"discoveredAt"`
	Excluded        bool          `json:"excluded"`
	Manual          bool          `json:"manual"`
	NeedsAssignment bool          `json:"needsAssignment"`
	Historical      bool          `json:"historical,omitempty"`
	File            string        `json:"file,omitempty"`
	Players         []Participant `json:"players,omitempty"`
}
type Participant struct {
	Name       string `json:"name"`
	Hero       string `json:"hero,omitempty"`
	ToonHandle string `json:"toonHandle"`
	Result     string `json:"result"`
}
type Observation struct {
	Hash      string `json:"hash"`
	Player    string `json:"player"`
	StartedAt int64  `json:"startedAt"`
	Mode      uint32 `json:"mode"`
	Ambiguous bool   `json:"ambiguous,omitempty"`
}
type State struct {
	Version      int           `json:"version"`
	Token        string        `json:"token"`
	Preferences  Preferences   `json:"preferences"`
	Since        int64         `json:"since"`
	Sessions     []Session     `json:"sessions"`
	Matches      []Match       `json:"matches"`
	Observations []Observation `json:"observations"`
	Tracking     []Period      `json:"tracking"`
	Revision     int           `json:"revision"`
}
type Period struct {
	From int64 `json:"from"`
	To   int64 `json:"to,omitempty"`
}

func TrackingAt(s State, at int64) bool {
	for _, p := range s.Tracking {
		if at >= p.From && (p.To == 0 || at < p.To) {
			return true
		}
	}
	return len(s.Tracking) == 0 && s.Preferences.Enabled && at >= s.Since
}

type Store struct {
	mu    sync.Mutex
	path  string
	state State
	err   error
}

func Open(path string) *Store {
	s := &Store{path: path, state: State{Version: 1, Preferences: Preferences{Style: "nexus", Scale: 100, Backdrop: true, Labels: true, WinColor: "#72dcff", LossColor: "#ff91b2", MaskMode: "auto", MaskRankedOnly: true}}}
	s.err = storage.LoadJSON(path, &s.state)
	if s.err == nil && s.state.Version != 1 {
		s.err = errors.New("unsupported sessions version")
	}
	return s
}
func (s *Store) Snapshot() (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var copy State
	b, _ := json.Marshal(s.state)
	_ = json.Unmarshal(b, &copy)
	return copy, s.err
}

// A failed load is never overwritten. Commit only after atomic persistence succeeds.
func (s *Store) Update(fn func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	b, _ := json.Marshal(s.state)
	var next State
	_ = json.Unmarshal(b, &next)
	if err := fn(&next); err != nil {
		return err
	}
	if reflect.DeepEqual(next, s.state) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	if err := storage.SaveJSON(s.path, next); err != nil {
		return err
	}
	s.state = next
	return nil
}
func Current(s *State, player string) *Session {
	for i := len(s.Sessions) - 1; i >= 0; i-- {
		if s.Sessions[i].Player == player && s.Sessions[i].EndedAt == 0 {
			return &s.Sessions[i]
		}
	}
	return nil
}
func New(s *State, player string, at int64) error {
	if player == "" {
		return errors.New("select a player first")
	}
	if s.Since == 0 {
		s.Since = at
	}
	if cur := Current(s, player); cur != nil {
		if at <= cur.StartedAt {
			return errors.New("session boundary must move forward")
		}
		cur.EndedAt = at
	}
	s.Sessions = append(s.Sessions, Session{ID: fmt.Sprintf("%d-%d", at, len(s.Sessions)), Player: player, StartedAt: at})
	return nil
}
func Assign(s *State, m *Match) {
	if m.StartedAt <= 0 {
		m.NeedsAssignment = true
		return
	}
	for i := len(s.Sessions) - 1; i >= 0; i-- {
		v := s.Sessions[i]
		if v.Player == m.Player && m.StartedAt >= v.StartedAt && (v.EndedAt == 0 || m.StartedAt < v.EndedAt) {
			m.SessionID = v.ID
			m.NeedsAssignment = false
			return
		}
	}
	m.NeedsAssignment = true
}
func Add(s *State, m Match) bool {
	for i, old := range s.Matches {
		if old.ID == m.ID && old.Player == m.Player {
			if old.Historical && !m.Historical && m.StartedAt > 0 && !old.Manual {
				Assign(s, &m)
				m.Excluded = old.Excluded
				s.Matches[i] = m
				return true
			}
			// Enrich older journal entries without touching user decisions.
			if old.File == "" {
				s.Matches[i].File = m.File
			}
			if len(old.Players) == 0 {
				s.Matches[i].Players = m.Players
			} else if m.HeroSource == "replay.details" {
				for j := range s.Matches[i].Players {
					participant := &s.Matches[i].Players[j]
					for _, incoming := range m.Players {
						if participant.ToonHandle != "" && participant.ToonHandle == incoming.ToonHandle {
							participant.Hero = incoming.Hero
							break
						}
					}
				}
			}
			if m.HeroSource == "replay.details" {
				s.Matches[i].Hero = m.Hero
				s.Matches[i].HeroSource = m.HeroSource
			} else if old.Hero == "" {
				s.Matches[i].Hero = m.Hero
			}
			if old.Map == "" {
				s.Matches[i].Map = m.Map
			}
			if old.Mode == 0 {
				s.Matches[i].Mode = m.Mode
			}
			if old.RecordedAt == 0 {
				s.Matches[i].RecordedAt = m.RecordedAt
			}
			return false
		}
	}
	if !m.Historical {
		Assign(s, &m)
	}
	s.Matches = append(s.Matches, m)
	return true
}
func Score(s State, id string) (wins, losses int) {
	for _, m := range s.Matches {
		if m.SessionID != id || m.Excluded || m.NeedsAssignment || s.Preferences.RankedOnly && m.Mode != 50091 {
			continue
		}
		if m.Result == "win" {
			wins++
		}
		if m.Result == "loss" {
			losses++
		}
	}
	return
}
func Undo(s *State, player string) error {
	cur := Current(s, player)
	if cur == nil {
		return errors.New("no current session")
	}
	var previous *Session
	for i := range s.Sessions {
		v := &s.Sessions[i]
		if v.Player == player && v.EndedAt == cur.StartedAt {
			previous = v
		}
	}
	if previous == nil {
		return errors.New("no previous session")
	}
	oldID, newID := cur.ID, previous.ID
	previous.EndedAt = 0
	for i := range s.Matches {
		if s.Matches[i].SessionID == oldID {
			s.Matches[i].SessionID = newID
		}
	}
	for i := range s.Sessions {
		if s.Sessions[i].ID == oldID {
			s.Sessions = append(s.Sessions[:i], s.Sessions[i+1:]...)
			break
		}
	}
	return nil
}
func Boundary(s *State, player, matchID string) error {
	var at int64
	for _, m := range s.Matches {
		if m.Player == player && m.ID == matchID {
			at = m.StartedAt
			if at == 0 {
				return errors.New("match start unknown; assign it to a session manually")
			}
		}
	}
	if at == 0 {
		return errors.New("match not found")
	}
	if err := New(s, player, at); err != nil {
		return err
	}
	for i := range s.Matches {
		m := &s.Matches[i]
		if m.Player == player && m.StartedAt >= at {
			m.SessionID = ""
			Assign(s, m)
		}
	}
	return nil
}
func Now() int64 { return time.Now().UnixMilli() }

// DisplaySettings avoids copying the entire match journal for OBS polling.
func (s *Store) DisplaySettings() (string, Preferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.state.Preferences
	p.ScoreThemes = append([]ScoreTheme(nil), p.ScoreThemes...)
	return s.state.Token, p, s.err
}
