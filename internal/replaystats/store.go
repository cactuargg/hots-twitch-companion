// Package replaystats owns observed results from local replays, not ratings or predictions.
package replaystats

import (
	"errors"
	"hots-twitch-companion/internal/hots/replay"
	"hots-twitch-companion/internal/storage"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Record struct {
	ID        string        `json:"id"`
	Mode      uint32        `json:"mode"`
	LobbyHash string        `json:"lobbyHash,omitempty"`
	Replay    replay.Replay `json:"replay"`
}
type database struct {
	Version int      `json:"version"`
	Matches []Record `json:"matches"`
}
type Store struct {
	mu     sync.Mutex
	path   string
	data   database
	byID   map[string]int
	cached map[string]Summary
	err    error
}

func Open(path string) *Store {
	s := &Store{path: path, data: database{Version: 1}, byID: map[string]int{}}
	s.err = storage.LoadJSON(path, &s.data)
	if s.err == nil && s.data.Version != 1 {
		s.err = errors.New("unsupported replay statistics version")
	}
	for i, m := range s.data.Matches {
		if (m.ID == "" || m.Replay.RecordedAt <= 0) && s.err == nil {
			s.err = errors.New("invalid replay statistics record")
		}
		if _, ok := s.byID[m.ID]; ok && s.err == nil {
			s.err = errors.New("duplicate replay in statistics database")
		}
		s.byID[m.ID] = i
	}
	return s
}
func (s *Store) Get(id string) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := s.byID[id]
	if !ok || s.err != nil {
		return Record{}, false
	}
	m := s.data.Matches[i]
	m.Replay.Players = append([]replay.Player(nil), m.Replay.Players...)
	return m, true
}
func (s *Store) Add(records []Record) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return 0, s.err
	}
	next := database{Version: 1, Matches: append([]Record(nil), s.data.Matches...)}
	seen := map[string]bool{}
	added := 0
	for _, m := range records {
		if _, ok := s.byID[m.ID]; ok || seen[m.ID] {
			continue
		}
		if m.ID == "" || m.Replay.RecordedAt <= 0 {
			return 0, errors.New("invalid replay statistics record")
		}
		seen[m.ID] = true
		m.Replay.Players = append([]replay.Player(nil), m.Replay.Players...)
		next.Matches = append(next.Matches, m)
		added++
	}
	if added == 0 {
		return 0, nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return 0, err
	}
	if err := storage.SaveJSON(s.path, next); err != nil {
		return 0, err
	}
	s.data = next
	for i, m := range next.Matches {
		s.byID[m.ID] = i
	}
	s.cached = nil
	return added, nil
}

type Row struct {
	Key      string  `json:"key"`
	Name     string  `json:"name"`
	Wins     int     `json:"wins"`
	Losses   int     `json:"losses"`
	Games    int     `json:"games"`
	WinRate  float64 `json:"winRate"`
	LastSeen int64   `json:"lastSeen"`
}
type Summary struct {
	Matches      int              `json:"matches"`
	Observations int              `json:"observations"`
	Skipped      int              `json:"skipped"`
	From         int64            `json:"from"`
	To           int64            `json:"to"`
	Heroes       []Row            `json:"heroes"`
	Players      []Row            `json:"players"`
	PlayerHeroes map[string][]Row `json:"-"`
}

var humanToon = regexp.MustCompile(`^[1-9][0-9]*-hero-[1-9][0-9]*-[1-9][0-9]*$`)

func validResult(result string) bool { return result == "win" || result == "loss" }
func rows(values map[string]*Row) []Row {
	out := make([]Row, 0, len(values))
	for _, v := range values {
		v.Games = v.Wins + v.Losses
		v.WinRate = 100 * float64(v.Wins) / float64(v.Games)
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Games != out[j].Games {
			return out[i].Games > out[j].Games
		}
		return out[i].Key < out[j].Key
	})
	return out
}
func count(values map[string]*Row, key, name, result string, at int64) {
	v := values[key]
	if v == nil {
		v = &Row{Key: key}
		values[key] = v
	}
	if at >= v.LastSeen {
		v.Name = name
		v.LastSeen = at
	}
	if result == "win" {
		v.Wins++
	} else {
		v.Losses++
	}
}

// Summary reuses the aggregate index until new replay records arrive. The
// caller receives owned slices; concurrent UI requests cannot mutate the cache.
func (s *Store) Summary(mode string, player string) (Summary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return Summary{}, s.err
	}
	if s.cached == nil {
		s.cached = map[string]Summary{}
	}
	result, ok := s.cached[mode]
	if !ok {
		heroes, players := map[string]*Row{}, map[string]*Row{}
		ph := map[string]map[string]*Row{}
		result.PlayerHeroes = map[string][]Row{}
		for _, m := range s.data.Matches {
			if mode != "all" && mode != modeKey(m.Mode) {
				continue
			}
			result.Matches++
			at := m.Replay.RecordedAt
			if result.From == 0 || at < result.From {
				result.From = at
			}
			if at > result.To {
				result.To = at
			}
			seen := map[string]int{}
			for _, p := range m.Replay.Players {
				if p.ToonHandle != "" {
					seen[strings.ToLower(p.ToonHandle)]++
				}
			}
			for _, p := range m.Replay.Players {
				id := strings.ToLower(strings.TrimSpace(p.ToonHandle))
				if !humanToon.MatchString(id) || seen[id] != 1 || !validResult(p.Result) {
					result.Skipped++
					continue
				}
				result.Observations++
				name := p.Name
				if name == "" {
					name = p.ToonHandle
				}
				count(players, id, name, p.Result, at)
				hero := strings.TrimSpace(p.Hero)
				if hero == "" {
					continue
				}
				hk := strings.ToLower(hero)
				count(heroes, hk, hero, p.Result, at)
				if ph[id] == nil {
					ph[id] = map[string]*Row{}
				}
				count(ph[id], hk, hero, p.Result, at)
			}
		}
		result.Heroes = rows(heroes)
		result.Players = rows(players)
		for id, h := range ph {
			result.PlayerHeroes[id] = rows(h)
		}
		s.cached[mode] = result
	}
	out := result
	out.PlayerHeroes = nil
	out.Players = append([]Row{}, result.Players...)
	if player != "" {
		out.Heroes = append([]Row{}, result.PlayerHeroes[strings.ToLower(player)]...)
	} else {
		out.Heroes = append([]Row{}, result.Heroes...)
	}
	return out, nil
}

func modeKey(mode uint32) string { return strconv.FormatUint(uint64(mode), 10) }
