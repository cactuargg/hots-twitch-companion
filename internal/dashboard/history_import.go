package dashboard

import (
	"errors"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/hots/correlation"
	"hots-twitch-companion/internal/hots/discovery"
	"hots-twitch-companion/internal/hots/lobby"
	"hots-twitch-companion/internal/hots/replay"
	"hots-twitch-companion/internal/sessions"
	"path/filepath"
	"strings"
	"time"
)

// Import only reads the selected replay roots. It never changes session boundaries,
// existing outcomes, assignments, exclusions, or Twitch state.
func (d *Dashboard) startHistoryImport() error {
	o := d.overlays
	c := d.Config()
	if playerKey(c) == "" {
		return errors.New("select a player first")
	}
	if _, err := o.store.Snapshot(); err != nil {
		return err
	}
	o.mu.Lock()
	if o.importing {
		o.mu.Unlock()
		return errors.New("import is already running")
	}
	o.importing = true
	o.importProcessed = 0
	o.importTotal = 0
	o.importAdded = 0
	o.importErrors = 0
	o.importError = ""
	o.mu.Unlock()
	go d.importHistory(c)
	return nil
}
func (d *Dashboard) importHistory(c config.Data) {
	o := d.overlays
	defer func() { o.mu.Lock(); o.importing = false; o.mu.Unlock() }()
	fail := func(err error) { o.mu.Lock(); o.importError = err.Error(); o.mu.Unlock() }
	files, err := overlayReplayFiles(c.ReplayRoot, time.Time{})
	if err != nil {
		fail(err)
		return
	}
	o.mu.Lock()
	o.importTotal = len(files)
	o.mu.Unlock()
	var pending []sessions.Match
	flush := func() error {
		if len(pending) == 0 {
			return nil
		}
		added := 0
		err := o.store.Update(func(s *sessions.State) error {
			for _, m := range pending {
				if sessions.Add(s, m) {
					added++
				}
			}
			return nil
		})
		if err == nil {
			o.mu.Lock()
			o.importAdded += added
			o.mu.Unlock()
			pending = nil
		}
		return err
	}
	for _, f := range files {
		select {
		case <-d.Done():
			fail(errors.New("import interrupted; repeat it after restart"))
			return
		default:
		}
		m, err := readHistoryMatch(f, c, time.Now())
		if err == nil && m != nil {
			pending = append(pending, *m)
		}
		o.mu.Lock()
		o.importProcessed++
		if err != nil {
			o.importErrors++
			o.importError = err.Error()
		}
		o.mu.Unlock()
		if len(pending) >= 32 {
			if err := flush(); err != nil {
				fail(err)
				return
			}
		}
	}
	if err := flush(); err != nil {
		fail(err)
	}
}
func readHistoryMatch(f discovery.ReplayFile, c config.Data, now time.Time) (*sessions.Match, error) {
	if now.Sub(f.Modified) < 8*time.Second {
		return nil, errors.New("replay is still being written; import it later")
	}
	raw, err := correlation.ReadStableReplay(f)
	if err != nil {
		return nil, err
	}
	member, err := replay.ReadMPQMember(raw, "replay.details")
	if err != nil {
		return nil, err
	}
	details, err := replay.ParseDetails(member)
	if err != nil {
		return nil, err
	}
	var selected []replay.Player
	for _, p := range details.Players {
		if c.PlayerToonHandle != "" && strings.EqualFold(p.ToonHandle, c.PlayerToonHandle) || c.PlayerToonHandle == "" && strings.EqualFold(p.Name, c.PlayerName) {
			selected = append(selected, p)
		}
	}
	if len(selected) != 1 {
		return nil, nil
	}
	if details.RecordedAt <= 0 || details.RecordedAt > now.Add(2*time.Minute).UnixMilli() {
		return nil, errors.New("replay has no valid game timestamp")
	}
	p := selected[0]
	m := &sessions.Match{ID: correlation.ContentSHA(raw), Player: playerKey(c), Map: details.Map, Hero: p.Hero, HeroSource: "replay.details", Result: p.Result, OriginalResult: p.Result, RecordedAt: details.RecordedAt, DiscoveredAt: now.UnixMilli(), Historical: true, File: filepath.Base(f.Path)}
	for _, v := range details.Players {
		m.Players = append(m.Players, sessions.Participant{Name: v.Name, ToonHandle: v.ToonHandle, Hero: v.Hero, Result: v.Result})
	}
	if b, e := replay.ReadMPQMember(raw, "replay.server.battlelobby"); e == nil {
		if l, e := lobby.Parse(b); e == nil {
			m.Mode = l.AMMID
		}
	}
	return m, nil
}

// Repair only existing entries; upgrading must never import unrelated matches.
func (d *Dashboard) repairHistoryHeroes(c config.Data) {
	state, err := d.overlays.store.Snapshot()
	if err != nil {
		return
	}
	pending := map[string]bool{}
	for _, m := range state.Matches {
		if m.Player != playerKey(c) {
			continue
		}
		needsRepair := m.HeroSource != "replay.details"
		for _, p := range m.Players {
			needsRepair = needsRepair || p.Hero == ""
		}
		if needsRepair {
			pending[m.ID] = true
		}
	}
	if len(pending) == 0 {
		return
	}
	files, err := overlayReplayFiles(c.ReplayRoot, time.Time{})
	if err != nil {
		return
	}
	for _, f := range files {
		select {
		case <-d.Done():
			return
		default:
		}
		m, err := readHistoryMatch(f, c, time.Now())
		if err != nil || m == nil || !pending[m.ID] {
			continue
		}
		if err := d.overlays.store.Update(func(s *sessions.State) error { sessions.Add(s, *m); return nil }); err != nil {
			return
		}
		delete(pending, m.ID)
		if len(pending) == 0 {
			return
		}
	}
}
