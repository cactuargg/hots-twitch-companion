package correlation

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
 "hots-twitch-companion/internal/hots/discovery"
 "hots-twitch-companion/internal/hots/lobby"
 "hots-twitch-companion/internal/hots/replay"
 "hots-twitch-companion/internal/prediction"
)

// Match automation fails closed on incomplete evidence, contradictory replay
// metadata, inventory overflows or modified files. No external Twitch calls.
type CorrelatedReplay struct {
	Path   string
	Player replay.Player
}

// Use a complete bounded inventory, not History's newest twelve files. Errors or
// limits fail closed so an unseen second candidate cannot be silently ignored.
func CorrelationReplayFiles(replayRoot string, after time.Time) ([]discovery.ReplayFile, error) {
	var files []discovery.ReplayFile
	visited := 0
	for _, root := range discovery.ReplayRoots(replayRoot) {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				if path == root && errors.Is(err, os.ErrNotExist) && replayRoot == "" {
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
				files = append(files, discovery.ReplayFile{Path: path, Modified: st.ModTime(), Size: st.Size()})
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

func ReadStableReplay(expected discovery.ReplayFile) ([]byte, error) {
	f, err := os.Open(expected.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() < 32 || before.Size() > replay.MaxReplay {
		return nil, errors.New("invalid replay size or type")
	}
	if before.Size() != expected.Size || !before.ModTime().Equal(expected.Modified) {
		return nil, errors.New("replay changed since inventory")
	}
	raw, err := io.ReadAll(io.LimitReader(f, replay.MaxReplay+1))
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

func ContentSHA(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }

func CorrelatePrediction(replayRoot string, p *prediction.Prediction, now time.Time) (*CorrelatedReplay, string, string, string) {
	m := p.Match
	blocked := func(code, file string, err error) (*CorrelatedReplay, string, string, string) {
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
	roster, err := ParticipantToons(m.Lobby.Data)
	if err != nil || !roster[m.PlayerToon] {
		return blocked("correlation_unavailable", "", err)
	}
	observed := time.UnixMilli(m.Lobby.ModifiedAt)
	if p.StartedAt <= 0 || observed.After(time.UnixMilli(p.StartedAt).Add(2*time.Second)) || time.UnixMilli(p.StartedAt).Sub(observed) > 2*time.Minute {
		return blocked("correlation_unavailable", "", errors.New("lobby observation is outside Prediction creation window"))
	}
	files, err := CorrelationReplayFiles(replayRoot, observed.Add(-2*time.Second))
	if err != nil {
		return blocked("correlation_incomplete", "", err)
	}
	if len(files) == 0 {
		return blocked("waiting_replay", "", nil)
	}
	matches := map[string]*CorrelatedReplay{}
	pendingCode, pendingFile, pendingDetail := "", "", ""
	for _, f := range files {
		name := filepath.Base(f.Path)
		if now.Sub(f.Modified) < 8*time.Second {
			pendingCode, pendingFile = "writing", name
			continue
		}
		raw, err := ReadStableReplay(f)
		if err != nil {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, err.Error()
			continue
		}
		member, err := replay.ReadMPQMember(raw, "replay.server.battlelobby")
		if err != nil {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, err.Error()
			continue
		}
		// This exact-byte equality is demonstrated by the paired real fixture.
		// A different hash is never accepted by falling back to seed/roster/time.
		if ContentSHA(member) != m.Lobby.SHA256 {
			continue
		}
		actualLobby, err := lobby.Parse(member)
		if err != nil || !reflect.DeepEqual(&actualLobby, m.Lobby.Data) {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, "embedded lobby differs from bound evidence"
			continue
		}
		details, err := replay.ReadMPQMember(raw, "replay.details")
		if err != nil {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, err.Error()
			continue
		}
		replayDetails, err := replay.ParseDetails(details)
		if err != nil {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, err.Error()
			continue
		}
		// The game's recorded clock must also be fresh. Touching an old replay's
		// filesystem timestamp must never make it eligible for a new Prediction.
		if replayDetails.RecordedAt < observed.Add(-2*time.Minute).UnixMilli() || replayDetails.RecordedAt > now.Add(2*time.Minute).UnixMilli() {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, "replay clock is missing or outside observation window"
			continue
		}
		player, err := CorrelatedOutcome(replayDetails, &actualLobby, m.PlayerToon)
		if err != nil {
			pendingCode, pendingFile, pendingDetail = "correlation_incomplete", name, err.Error()
			continue
		}
		matches[ContentSHA(raw)] = &CorrelatedReplay{Path: f.Path, Player: player}
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
