package main

import (
 correlation "hots-twitch-companion/internal/hots/correlation"
	predstate "hots-twitch-companion/internal/prediction"
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
type MatchBinding = predstate.MatchBinding

func bindPredictionMatch(lobby *LobbySnapshot,key,toon string)*MatchBinding {
 return correlation.BindPredictionMatch(lobby,key,toon)
}
func observeActiveLobby(active *Prediction,next *LobbySnapshot,key string)*Prediction {
 return correlation.ObserveActiveLobby(active,next,key)
}
func ambiguousMatch(active *Prediction)*Prediction {
 return correlation.AmbiguousMatch(active)
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
func participantToons(lobby *BattleLobby)(map[string]bool,error) {return correlation.ParticipantToons(lobby)}
func isHeroHandle(toon string)bool{return correlation.IsHeroHandle(toon)}
func correlatedOutcome(replay Replay,lobby *BattleLobby,toon string)(Player,error) {
 return correlation.CorrelatedOutcome(replay,lobby,toon)
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
