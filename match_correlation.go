package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// Evidence, not a global Match ID. Never rebind an existing Prediction.
type MatchBinding struct {
	Version    int            `json:"version"`
	Lobby      *LobbySnapshot `json:"lobby"`
	LobbyKey   string         `json:"lobbyKey"`
	PlayerToon string         `json:"playerToon"`
	Ambiguous  bool           `json:"ambiguous,omitempty"`
}

func bindPredictionMatch(lobby *LobbySnapshot, key, toon string) *MatchBinding {
	binding := &MatchBinding{Version: 1, LobbyKey: key, PlayerToon: strings.TrimSpace(toon)}
	if lobby != nil {
		copyLobby := *lobby
		if lobby.Data != nil {
			data := *lobby.Data
			data.Players = append([]LobbyPlayer(nil), data.Players...)
			copyLobby.Data = &data
		}
		binding.Lobby = &copyLobby
	}
	return binding
}

func observeActiveLobby(active *Prediction, next *LobbySnapshot, key string) *Prediction {
	m := active.Match
	if m == nil || m.Lobby == nil || next == nil || m.Lobby.SHA256 == "" || m.Lobby.SHA256 != next.SHA256 || key == m.LobbyKey {
		return active
	}
	// The same bytes were observed again as a new lobby event. We cannot prove
	// whether this was a rewrite or a repeated match; keep that uncertainty durable.
	return ambiguousMatch(active)
}

func ambiguousMatch(active *Prediction) *Prediction {
	copyPrediction := *active
	copyMatch := *active.Match
	copyMatch.Ambiguous = true
	copyPrediction.Match = &copyMatch
	return &copyPrediction
}

type correlatedReplay struct {
	Path   string
	Player Player
}

// Use a complete bounded inventory, not History's newest twelve files. Errors or
// limits fail closed so an unseen second candidate cannot be silently ignored.
func correlationReplayFiles(cfg Config, after time.Time) ([]ReplayFile, error) {
	var files []ReplayFile
	visited := 0
	for _, root := range replayRoots(cfg) {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				if path == root && errors.Is(err, os.ErrNotExist) && cfg.ReplayRoot == "" {
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
			st, err := d.Info()
			if err != nil {
				return err
			}
			if !st.Mode().IsRegular() {
				return errors.New("replay is not a regular file")
			}
			if !st.ModTime().Before(after) {
				files = append(files, ReplayFile{Path: path, Modified: st.ModTime(), Size: st.Size()})
			}
			if len(files) > 128 {
				return errors.New("more than 128 candidate replays")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func readStableReplay(expected ReplayFile) ([]byte, error) {
	f, err := os.Open(expected.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() < 32 || before.Size() > maxReplay {
		return nil, errors.New("invalid replay size or type")
	}
	if before.Size() != expected.Size || !before.ModTime().Equal(expected.Modified) {
		return nil, errors.New("replay changed since inventory")
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxReplay+1))
	if err != nil {
		return nil, err
	}
	after, err := os.Stat(expected.Path)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, after) || before.Size() != int64(len(raw)) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, errors.New("replay changed during capture")
	}
	return raw, nil
}

func contentSHA(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }

// Deliberately narrow automatic support: 5v5 lobbies. Matchmaking games
// must field ten human participants; custom lobbies may substitute AI
// participants, while every human slot is still verified by ToonHandle and
// observers stay excluded by explicit role, never by slot position.
func participantToons(lobby *BattleLobby) (map[string]bool, error) {
	if lobby == nil || lobby.TeamSize != "5v5" {
		return nil, errors.New("automatic correlation requires a 5v5 lobby")
	}
	if !lobby.Custom {
		switch lobby.AMMID {
		case 50001, 50051, 50091, 50101:
		default:
			return nil, errors.New("unsupported matchmaking mode")
		}
	}
	toons := map[string]bool{}
	humans := 0
	for _, p := range lobby.Players {
		if p.Role == "Watc" {
			continue
		}
		if p.Role != "Part" {
			return nil, errors.New("unexpected lobby role")
		}
		if p.SlotType == "Comp" {
			if p.ToonHandle != "" {
				return nil, errors.New("AI slot carries a ToonHandle")
			}
			continue
		}
		if p.SlotType != "Humn" || p.ToonHandle == "" || !strings.Contains(p.ToonHandle, "-Hero-") || toons[p.ToonHandle] {
			return nil, errors.New("incomplete or non-human roster")
		}
		toons[p.ToonHandle] = true
		humans++
	}
	if !lobby.Custom && humans != 10 {
		return nil, errors.New("expected ten unique participants")
	}
	if humans == 0 {
		return nil, errors.New("no human participants")
	}
	return toons, nil
}

// Blizzard ToonHandles are "<region>-Hero-<realm>-<id>". AI participants get
// synthetic handles without the Hero program and carry no verified identity.
func isHeroHandle(toon string) bool { return strings.Contains(toon, "-Hero-") }

func correlatedOutcome(replay Replay, lobby *BattleLobby, toon string) (Player, error) {
	toons, err := participantToons(lobby)
	if err != nil {
		return Player{}, err
	}
	if !toons[toon] {
		return Player{}, errors.New("selected ToonHandle is not a participant")
	}
	seen := map[string]bool{}
	counts := [2]int{}
	results := [2]string{}
	var selected Player
	record := func(p Player) error {
		if p.Team < 0 || p.Team > 1 || (p.Result != "win" && p.Result != "loss") {
			return errors.New("invalid replay participant, team or outcome")
		}
		if results[p.Team] != "" && results[p.Team] != p.Result {
			return errors.New("inconsistent team outcome")
		}
		results[p.Team] = p.Result
		counts[p.Team]++
		return nil
	}
	for _, p := range replay.Players {
		if toons[p.ToonHandle] {
			if seen[p.ToonHandle] {
				return Player{}, errors.New("duplicate replay participant")
			}
			seen[p.ToonHandle] = true
			if err := record(p); err != nil {
				return Player{}, err
			}
			if p.ToonHandle == toon {
				selected = p
			}
			continue
		}
		// The replay is already bound to this exact lobby by byte-identical
		// embedded BattleLobby content. In custom lobbies remaining entries
		// are AI participants: identity is not verifiable, team and outcome
		// still are. Any additional Hero handle means an unaccounted player.
		if !lobby.Custom || isHeroHandle(p.ToonHandle) {
			return Player{}, errors.New("replay participant is not in the lobby roster")
		}
		if err := record(p); err != nil {
			return Player{}, err
		}
	}
	if len(seen) != len(toons) {
		return Player{}, errors.New("replay and lobby participant counts differ")
	}
	if lobby.Custom {
		if counts[0] == 0 || counts[1] == 0 || results[0] == results[1] {
			return Player{}, errors.New("inconsistent custom-game result")
		}
	} else if counts[0] != 5 || counts[1] != 5 || results[0] == results[1] {
		return Player{}, errors.New("inconsistent 5v5 result")
	}
	if selected.ToonHandle == "" {
		return Player{}, errors.New("selected participant is missing from the replay")
	}
	return selected, nil
}

func correlatePrediction(cfg Config, p *Prediction, now time.Time) (*correlatedReplay, string, string, string) {
	m := p.Match
	blocked := func(code, file string, err error) (*correlatedReplay, string, string, string) {
		detail := ""
		if err != nil {
			detail = err.Error()
		}
		return nil, code, file, detail
	}
	if m == nil || m.Version != 1 || m.Lobby == nil || m.Lobby.Status != "roster_parsed" || len(m.Lobby.SHA256) != 64 || m.Lobby.ModifiedAt <= 0 || m.Lobby.Data == nil {
		return blocked("correlation_unavailable", "", nil)
	}
	if m.Ambiguous {
		return blocked("correlation_ambiguous", "", nil)
	}
	roster, err := participantToons(m.Lobby.Data)
	if err != nil || !roster[m.PlayerToon] {
		return blocked("correlation_unavailable", "", err)
	}
	observed := time.UnixMilli(m.Lobby.ModifiedAt)
	if p.StartedAt <= 0 || observed.After(time.UnixMilli(p.StartedAt).Add(2*time.Second)) || time.UnixMilli(p.StartedAt).Sub(observed) > 2*time.Minute {
		return blocked("correlation_unavailable", "", errors.New("lobby observation is outside Prediction creation window"))
	}
	files, err := correlationReplayFiles(cfg, observed.Add(-2*time.Second))
	if err != nil {
		return blocked("correlation_incomplete", "", err)
	}
	if len(files) == 0 {
		return blocked("waiting_replay", "", nil)
	}
	matches := map[string]*correlatedReplay{}
	pendingCode, pendingFile, pendingDetail := "", "", ""
	for _, f := range files {
		name := filepath.Base(f.Path)
		if now.Sub(f.Modified) < 8*time.Second {
			pendingCode, pendingFile = "writing", name
			continue
		}
		raw, err := readStableReplay(f)
		if err != nil {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, err.Error()
			continue
		}
		member, err := readMPQMember(raw, "replay.server.battlelobby")
		if err != nil {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, err.Error()
			continue
		}
		// This exact-byte equality is demonstrated by the paired real fixture.
		// A different hash is never accepted by falling back to seed/roster/time.
		if contentSHA(member) != m.Lobby.SHA256 {
			continue
		}
		lobby, err := parseBattleLobby(member)
		if err != nil || !reflect.DeepEqual(&lobby, m.Lobby.Data) {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, "embedded lobby differs from bound evidence"
			continue
		}
		details, err := readMPQMember(raw, "replay.details")
		if err != nil {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, err.Error()
			continue
		}
		replay, err := parseReplayDetails(details)
		if err != nil {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, err.Error()
			continue
		}
		// The game's recorded clock must also be fresh. Touching an old replay's
		// filesystem timestamp must never make it eligible for a new Prediction.
		if replay.RecordedAt < observed.Add(-2*time.Minute).UnixMilli() || replay.RecordedAt > now.Add(2*time.Minute).UnixMilli() {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, "replay clock is missing or outside observation window"
			continue
		}
		player, err := correlatedOutcome(replay, &lobby, m.PlayerToon)
		if err != nil {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, err.Error()
			continue
		}
		matches[contentSHA(raw)] = &correlatedReplay{Path: f.Path, Player: player}
	}
	if len(matches) > 1 {
		return blocked("correlation_ambiguous", "", fmt.Errorf("%d distinct replays match the bound lobby", len(matches)))
	}
	if pendingCode != "" {
		return nil, pendingCode, pendingFile, pendingDetail
	}
	for _, match := range matches {
		return match, "", "", ""
	}
	return blocked("correlation_mismatch", "", nil)
}
