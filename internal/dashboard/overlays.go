package dashboard

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/hots/correlation"
	"hots-twitch-companion/internal/hots/discovery"
	"hots-twitch-companion/internal/hots/lobby"
	"hots-twitch-companion/internal/hots/replay"
	"hots-twitch-companion/internal/sessions"
)

const overlayAddress = "127.0.0.1:18765"

func readOverlayJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return errors.New("unexpected trailing JSON")
	}
	return nil
}

type overlayRuntime struct {
	importing       bool
	importProcessed int
	importTotal     int
	importAdded     int
	importErrors    int
	importError     string
	store           *sessions.Store
	mu              sync.Mutex
	server          *http.Server
	err             string
	scanError       string
	phase           string
	activeHash      string
	activeMode      uint32
	observedAt      int64
	checkedAt       int64
	completed       map[string]bool
	seen            map[string]bool
	retry           map[string]time.Time
	lastPlayer      string
	lastRevision    int
	findLobby       func() (string, string, time.Time, error)
}

func playerKey(c config.Data) string {
	if c.PlayerToonHandle != "" {
		return strings.ToLower(c.PlayerToonHandle)
	}
	if c.PlayerName != "" {
		return "name:" + strings.ToLower(strings.TrimSpace(c.PlayerName))
	}
	return ""
}
func (d *Dashboard) initOverlays() {
	d.overlays = &overlayRuntime{store: sessions.Open(filepath.Join(d.p.Dir, "sessions.json")), phase: "unknown", seen: map[string]bool{}, retry: map[string]time.Time{}, completed: map[string]bool{}, findLobby: discovery.FindLobby}
}
func (d *Dashboard) startOverlays(ctx context.Context) {
	o := d.overlays
	st, err := o.store.Snapshot()
	if err == nil && st.Token == "" {
		token, e := RandomSegment()
		err = e
		if err == nil {
			err = o.store.Update(func(s *sessions.State) error { s.Token = token; return nil })
		}
	}
	if err == nil {
		listener, e := net.Listen("tcp", overlayAddress)
		err = e
		if e == nil {
			srv := &http.Server{Handler: d.overlayHandler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
			o.mu.Lock()
			o.server = srv
			o.mu.Unlock()
			go func() {
				if e := srv.Serve(listener); e != nil && !errors.Is(e, http.ErrServerClosed) {
					o.mu.Lock()
					o.err = e.Error()
					o.mu.Unlock()
				}
			}()
			go func() { <-ctx.Done(); _ = srv.Close() }()
		}
	}
	if err != nil {
		o.mu.Lock()
		o.err = err.Error()
		o.mu.Unlock()
	}
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			d.scanOverlayMatches()
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
func (d *Dashboard) overlaySnapshot(journal bool) map[string]any {
	o := d.overlays
	if o == nil {
		return map[string]any{}
	}
	s, err := o.store.Snapshot()
	player := playerKey(d.Config())
	cur := sessions.Current(&s, player)
	wins, losses := 0, 0
	if cur != nil {
		wins, losses = sessions.Score(s, cur.ID)
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	problem := o.err
	if err != nil {
		problem = err.Error()
	}
	visible := s.Preferences.MaskEnabled
	if visible {
		switch s.Preferences.MaskMode {
		case "hide":
			visible = false
		case "show":
			visible = true
		default:
			fresh := sessions.Now()-o.checkedAt < 15000
			active := o.phase == "loading_or_game" && fresh && sessions.Now()-o.observedAt < 100*time.Minute.Milliseconds() && o.lastPlayer == player
			visible = !(active || o.phase == "excluded_mode" && fresh && o.lastPlayer == player)
		}
	}
	res := map[string]any{"preferences": s.Preferences, "current": cur, "wins": wins, "losses": losses, "maskVisible": visible, "phase": o.phase, "error": problem, "scanError": o.scanError, "player": player, "checkedAt": o.checkedAt}
	if journal {
		res["import"] = map[string]any{"running": o.importing, "processed": o.importProcessed, "total": o.importTotal, "added": o.importAdded, "errors": o.importErrors, "error": o.importError}
		res["sessions"] = s.Sessions
		res["matches"] = s.Matches
		if s.Token != "" && problem == "" {
			res["scoreURL"] = "http://" + overlayAddress + "/" + s.Token + "/score"
			res["maskURL"] = "http://" + overlayAddress + "/" + s.Token + "/mask"
		}
	}
	return res
}
func (d *Dashboard) overlayHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != overlayAddress {
			http.Error(w, "Unknown Host", 403)
			return
		}
		if r.Method != "GET" {
			http.Error(w, "Read only", 405)
			return
		}
		s, err := d.overlays.store.Snapshot()
		if err != nil || s.Token == "" {
			http.Error(w, "Unavailable", 503)
			return
		}
		prefix := "/" + s.Token + "/"
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		route := strings.TrimPrefix(r.URL.Path, prefix)
		switch route {
		case "state":
			d.mu.Lock()
			d.lastActivity = time.Now()
			d.mu.Unlock()
			// Only public display data; never journal, identities, paths or management endpoints.
			v := d.overlaySnapshot(false)
			p := v["preferences"].(sessions.Preferences)
			imageURL := ""
			if p.MaskImage != "" {
				imageURL = "image?rev=" + correlation.ContentSHA([]byte(p.MaskImage))
			}
			p.MaskImage = ""
			JSONOutput(w, 200, map[string]any{"preferences": p, "maskImageURL": imageURL, "wins": v["wins"], "losses": v["losses"], "maskVisible": v["maskVisible"]})
		case "image":
			parts := strings.SplitN(s.Preferences.MaskImage, ",", 2)
			if len(parts) != 2 {
				http.NotFound(w, r)
				return
			}
			b, e := base64.StdEncoding.DecodeString(parts[1])
			if e != nil {
				http.Error(w, "Invalid image", 500)
				return
			}
			w.Header().Set("Content-Type", http.DetectContentType(b))
			_, _ = w.Write(b)
		case "score", "mask":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			b, _ := webFiles.ReadFile("web/overlay.html")
			_, _ = w.Write(b)
		case "overlay.js", "overlay.css":
			mime := "text/javascript"
			if strings.HasSuffix(route, "css") {
				mime = "text/css"
			}
			w.Header().Set("Content-Type", mime+"; charset=utf-8")
			b, _ := webFiles.ReadFile("web/assets/" + route)
			_, _ = w.Write(b)
		default:
			http.NotFound(w, r)
		}
	})
}

var overlayColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func validPreferences(p sessions.Preferences) error {
	if p.Style != "nexus" && p.Style != "minimal" || p.Scale < 50 || p.Scale > 200 || !overlayColor.MatchString(p.WinColor) || !overlayColor.MatchString(p.LossColor) {
		return errors.New("invalid score appearance")
	}
	if p.MaskMode != "auto" && p.MaskMode != "show" && p.MaskMode != "hide" {
		return errors.New("invalid mask mode")
	}
	if p.MaskImage != "" {
		prefix := "data:image/png;base64,"
		if strings.HasPrefix(p.MaskImage, "data:image/jpeg;base64,") {
			prefix = "data:image/jpeg;base64,"
		}
		if !strings.HasPrefix(p.MaskImage, prefix) {
			return errors.New("use a PNG or JPEG image")
		}
		b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(p.MaskImage, prefix))
		if err != nil || len(b) > 2<<20 {
			return errors.New("image must be under 2 MiB")
		}
		// Check the encoded content as well as the supplied media type.
		mime := http.DetectContentType(b)
		if mime != "image/png" && mime != "image/jpeg" {
			return errors.New("invalid image")
		}
	}
	return nil
}
func (d *Dashboard) overlayAPI(w http.ResponseWriter, r *http.Request, route string) {
	o := d.overlays
	if r.Method == "GET" && route == "sessions" {
		JSONOutput(w, 200, d.overlaySnapshot(true))
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	var body struct {
		Preferences *sessions.Preferences `json:"preferences"`
		MatchID     string                `json:"matchId"`
		SessionID   string                `json:"sessionId"`
		Result      string                `json:"result"`
		Excluded    *bool                 `json:"excluded"`
	}
	// Image uploads have a separate bounded envelope; the general API remains 16 KiB.
	if route == "sessions/preferences" {
		r.Body = http.MaxBytesReader(w, r.Body, 3<<20)
		r.ContentLength = -1
		err := readOverlayJSON(r, &body)
		if err != nil {
			RequestError(w, err)
			return
		}
	} else if err := ReadJSONRequest(r, &body, d.Text("http.request_too_large")); err != nil {
		RequestError(w, err)
		return
	}
	player := playerKey(d.Config())
	if route == "sessions/import" {
		if err := d.startHistoryImport(); err != nil {
			RequestError(w, err)
			return
		}
		JSONOutput(w, 202, d.overlaySnapshot(true))
		return
	}
	now := sessions.Now()
	err := o.store.Update(func(s *sessions.State) error {
		switch route {
		case "sessions/preferences":
			if body.Preferences == nil {
				return errors.New("preferences required")
			}
			p := *body.Preferences
			if err := validPreferences(p); err != nil {
				return err
			}
			if p.Enabled && sessions.Current(s, player) == nil {
				if err := sessions.New(s, player, now); err != nil {
					return err
				}
			}
			if p.Enabled != s.Preferences.Enabled {
				s.Revision++
				if p.Enabled {
					s.Tracking = append(s.Tracking, sessions.Period{From: now})
				} else if len(s.Tracking) > 0 {
					s.Tracking[len(s.Tracking)-1].To = now
				}
			}
			s.Preferences = p
		case "sessions/new":
			return sessions.New(s, player, now)
		case "sessions/undo":
			return sessions.Undo(s, player)
		case "sessions/boundary":
			return sessions.Boundary(s, player, body.MatchID)
		case "sessions/match":
			for i := range s.Matches {
				m := &s.Matches[i]
				if m.ID != body.MatchID || m.Player != player {
					continue
				}
				if body.Result != "" {
					if body.Result != "win" && body.Result != "loss" && body.Result != "unknown" {
						return errors.New("invalid result")
					}
					m.Result = body.Result
					m.Manual = m.Result != m.OriginalResult
				}
				if body.Excluded != nil {
					m.Excluded = *body.Excluded
				}
				if body.SessionID != "" {
					found := false
					for _, v := range s.Sessions {
						if v.ID == body.SessionID && v.Player == player {
							found = true
						}
					}
					if !found {
						return errors.New("session not found")
					}
					m.SessionID = body.SessionID
					m.NeedsAssignment = false
					m.Historical = false
					m.Manual = true
				}
				return nil
			}
			return errors.New("match not found")
		default:
			return errors.New("unknown session action")
		}
		return nil
	})
	if err != nil {
		RequestError(w, err)
		return
	}
	JSONOutput(w, 200, d.overlaySnapshot(true))
}

// Scan in bounded batches, with a cache of stable files. Failed files are retried.
// No Twitch operation, Prediction state or settlement is involved.
func (d *Dashboard) scanOverlayMatches() {
	o := d.overlays
	c := d.Config()
	player := playerKey(c)
	s, err := o.store.Snapshot()
	if err != nil {
		return
	}
	if !s.Preferences.Enabled && !s.Preferences.MaskEnabled {
		return
	}
	if player == "" {
		o.mu.Lock()
		o.scanError = "Select a player in settings"
		o.mu.Unlock()
		return
	}
	if s.Preferences.Enabled && sessions.Current(&s, player) == nil {
		if err = o.store.Update(func(s *sessions.State) error { return sessions.New(s, player, sessions.Now()) }); err != nil {
			o.mu.Lock()
			o.scanError = err.Error()
			o.mu.Unlock()
			return
		}
		s, _ = o.store.Snapshot()
	}
	now := time.Now()
	var scanErr error
	o.mu.Lock()
	if o.lastPlayer != player {
		o.seen = map[string]bool{}
		o.phase = "unknown"
		o.activeHash = ""
		o.lastPlayer = player
	}
	if o.lastRevision != s.Revision {
		o.seen = map[string]bool{}
		o.lastRevision = s.Revision
	}
	o.mu.Unlock()
	if path, _, modified, e := o.findLobby(); e == nil && now.Sub(modified) > 2500*time.Millisecond && now.Sub(modified) < 2*time.Minute {
		snap := lobby.Capture(path)
		if snap.Status == "roster_parsed" && snap.Data != nil && snap.ModifiedAt == modified.UnixMilli() {
			selected := false
			for _, p := range snap.Data.Players {
				if strings.EqualFold(p.ToonHandle, c.PlayerToonHandle) && c.PlayerToonHandle != "" && p.Role != "Watc" {
					selected = true
				}
			}
			if selected {
				o.mu.Lock()
				changed := o.activeHash != snap.SHA256 || o.observedAt != modified.UnixMilli()
				o.mu.Unlock()
				if changed {
					found := false
					ambiguous := false
					for _, v := range s.Observations {
						if v.Hash == snap.SHA256 && v.Player == player {
							found = true
							ambiguous = v.Ambiguous || v.StartedAt != modified.UnixMilli()
						}
					}
					if found {
						err = o.store.Update(func(s *sessions.State) error {
							for i := range s.Observations {
								v := &s.Observations[i]
								if v.Hash == snap.SHA256 && v.Player == player && v.StartedAt != modified.UnixMilli() {
									v.Ambiguous = true
								}
							}
							return nil
						})
						if err != nil {
							scanErr = err
						}
					}
					if !found {
						err = o.store.Update(func(s *sessions.State) error {
							s.Observations = append(s.Observations, sessions.Observation{Hash: snap.SHA256, Player: player, StartedAt: modified.UnixMilli(), Mode: snap.Data.AMMID})
							return nil
						})
						if err != nil {
							scanErr = err
						}
					}
					o.mu.Lock()
					o.activeHash = snap.SHA256
					o.activeMode = snap.Data.AMMID
					o.observedAt = modified.UnixMilli()
					o.phase = "loading_or_game"
					if s.Preferences.MaskRankedOnly && snap.Data.AMMID != 50091 {
						o.phase = "excluded_mode"
					}
					if o.completed[snap.SHA256] {
						o.phase = "between_games"
					}
					if ambiguous {
						o.phase = "unknown"
					}
					if err != nil {
						o.phase = "unknown"
					}
					o.mu.Unlock()
				}
			} else {
				o.mu.Lock()
				o.phase = "unknown"
				o.mu.Unlock()
			}
		} else {
			o.mu.Lock()
			o.phase = "unknown"
			o.mu.Unlock()
		}
	}
	s, _ = o.store.Snapshot()
	after := time.UnixMilli(s.Since).Add(-2 * time.Minute)
	if s.Since == 0 {
		after = now.Add(-2 * time.Hour)
	}
	files, e := overlayReplayFiles(c.ReplayRoot, after)
	if e != nil {
		scanErr = e
	} else {
		sort.Slice(files, func(i, j int) bool { return files[i].Modified.Before(files[j].Modified) })
		processed := 0
		for _, f := range files {
			key := fmt.Sprintf("%s|%d|%d|%s", f.Path, f.Modified.UnixNano(), f.Size, player)
			if o.seen[key] || now.Before(o.retry[key]) || now.Sub(f.Modified) < 8*time.Second {
				continue
			}
			if processed >= 16 {
				break
			}
			processed++
			o.retry[key] = now.Add(time.Minute)
			raw, e := correlation.ReadStableReplay(f)
			if e != nil {
				scanErr = e
				continue
			}
			member, e := replay.ReadMPQMember(raw, "replay.details")
			if e != nil {
				scanErr = fmt.Errorf("%s: %w", filepath.Base(f.Path), e)
				continue
			}
			details, e := replay.ParseDetails(member)
			if e != nil {
				scanErr = e
				continue
			}
			p, e := replay.FindOutcome(details, c.PlayerName, c.PlayerToonHandle, c.Language())
			// Keep unknown outcomes in the journal when the identity itself is unique.
			if e != nil {
				var candidates []replay.Player
				for _, v := range details.Players {
					if c.PlayerToonHandle != "" && strings.EqualFold(v.ToonHandle, c.PlayerToonHandle) || c.PlayerToonHandle == "" && strings.EqualFold(v.Name, c.PlayerName) {
						candidates = append(candidates, v)
					}
				}
				if len(candidates) != 1 {
					o.seen[key] = true
					continue
				}
				p = candidates[0]
			}
			hash := ""
			mode := uint32(0)
			hero := ""
			if b, e := replay.ReadMPQMember(raw, "replay.server.battlelobby"); e == nil {
				hash = correlation.ContentSHA(b)
				if l, e := lobby.Parse(b); e == nil {
					mode = l.AMMID
					for _, v := range l.Players {
						if v.ToonHandle == p.ToonHandle {
							hero = v.Hero
						}
					}
				}
			}
			start := int64(0)
			for _, v := range s.Observations {
				if v.Hash == hash && v.Player == player && !v.Ambiguous {
					start = v.StartedAt
				}
			}
			if details.RecordedAt > now.Add(2*time.Minute).UnixMilli() {
				scanErr = errors.New("replay clock is in the future")
				continue
			}
			m := sessions.Match{ID: correlation.ContentSHA(raw), Player: player, Map: details.Map, Hero: hero, Mode: mode, Result: p.Result, OriginalResult: p.Result, RecordedAt: details.RecordedAt, StartedAt: start, DiscoveredAt: now.UnixMilli()}
			m.File = filepath.Base(f.Path)
			for _, v := range details.Players {
				m.Players = append(m.Players, sessions.Participant{Name: v.Name, ToonHandle: v.ToonHandle, Result: v.Result})
			}
			matchTime := start
			if matchTime == 0 {
				matchTime = details.RecordedAt
			}
			playerSince := int64(0)
			for _, v := range s.Sessions {
				if v.Player == player && (playerSince == 0 || v.StartedAt < playerSince) {
					playerSince = v.StartedAt
				}
			}
			if s.Preferences.Enabled && matchTime >= playerSince && sessions.TrackingAt(s, matchTime) {
				if err = o.store.Update(func(s *sessions.State) error { sessions.Add(s, m); return nil }); err != nil {
					scanErr = err
					continue
				}
			}
			o.seen[key] = true
			delete(o.retry, key)
			if hash != "" {
				o.mu.Lock()
				o.completed[hash] = true
				if o.activeHash == hash {
					o.phase = "between_games"
				}
				o.mu.Unlock()
			}
		}
	}
	o.mu.Lock()
	o.checkedAt = now.UnixMilli()
	o.scanError = ""
	if scanErr != nil {
		o.scanError = scanErr.Error()
	}
	if now.UnixMilli()-o.observedAt > 100*time.Minute.Milliseconds() {
		o.phase = "unknown"
	}
	if o.phase == "loading_or_game" || o.phase == "excluded_mode" {
		o.phase = "loading_or_game"
		if s.Preferences.MaskRankedOnly && o.activeMode != 50091 {
			o.phase = "excluded_mode"
		}
	}
	o.mu.Unlock()
}

// Unlike Prediction correlation, a journal can recover more than 128 matches.
// The complete inventory is still bounded and never silently truncated.
func overlayReplayFiles(root string, after time.Time) ([]discovery.ReplayFile, error) {
	var files []discovery.ReplayFile
	visited := 0
	for _, base := range discovery.ReplayRoots(root) {
		err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				if path == base && root == "" && errors.Is(err, os.ErrNotExist) {
					return filepath.SkipDir
				}
				return err
			}
			visited++
			if visited > 20000 {
				return errors.New("replay inventory exceeds 20000 entries")
			}
			if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".stormreplay") {
				return nil
			}
			st, e := d.Info()
			if e != nil {
				return e
			}
			if !st.Mode().IsRegular() {
				return errors.New("replay is not a regular file")
			}
			if !st.ModTime().Before(after) {
				files = append(files, discovery.ReplayFile{Path: path, Modified: st.ModTime(), Size: st.Size()})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}
