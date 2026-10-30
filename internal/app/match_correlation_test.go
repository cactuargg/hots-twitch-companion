package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/hots/correlation"
	"hots-twitch-companion/internal/hots/discovery"
	lobbydata "hots-twitch-companion/internal/hots/lobby"
	"hots-twitch-companion/internal/hots/replay"
	predmodel "hots-twitch-companion/internal/prediction"
	twitchpkg "hots-twitch-companion/internal/twitch"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func realLobbyTest(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../testdata/battlelobby/replay.server.StormLeague1_89566.battlelobby")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func bindingTest(t *testing.T, raw []byte, at time.Time) *predmodel.MatchBinding {
	t.Helper()
	lobby, err := lobbydata.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return correlation.BindPredictionMatch(&lobbydata.Snapshot{Status: "roster_parsed", Parser: "battlelobby-roster-v1", SHA256: correlation.ContentSHA(raw), ModifiedAt: at.UnixMilli(), Data: &lobby}, "original", lobby.Players[0].ToonHandle)
}

// Synthetic multi-member MPQ container around real, unmodified lobby bytes.
func correlationMPQTest(details, lobby []byte) []byte {
	members := [][]byte{details, lobby}
	names := []string{"replay.details", "replay.server.battlelobby"}
	head := make([]byte, 32)
	copy(head, "MPQ\x1a")
	binary.LittleEndian.PutUint32(head[4:], 32)
	binary.LittleEndian.PutUint16(head[14:], 3)
	binary.LittleEndian.PutUint32(head[16:], 32)
	binary.LittleEndian.PutUint32(head[20:], 96)
	binary.LittleEndian.PutUint32(head[24:], 4)
	binary.LittleEndian.PutUint32(head[28:], 2)
	hashes := bytes.Repeat([]byte{255}, 64)
	blocks := make([]byte, 32)
	payload := []byte{}
	for i, m := range members {
		h := hashes[i*16:]
		binary.LittleEndian.PutUint32(h, hashMPQ(names[i], 1))
		binary.LittleEndian.PutUint32(h[4:], hashMPQ(names[i], 2))
		binary.LittleEndian.PutUint32(h[12:], uint32(i))
		block := blocks[i*16:]
		binary.LittleEndian.PutUint32(block, uint32(128+len(payload)))
		binary.LittleEndian.PutUint32(block[4:], uint32(len(m)))
		binary.LittleEndian.PutUint32(block[8:], uint32(len(m)))
		binary.LittleEndian.PutUint32(block[12:], 0x81000000)
		payload = append(payload, m...)
	}
	binary.LittleEndian.PutUint32(head[8:], uint32(128+len(payload)))
	return bytes.Join([][]byte{head, encryptTest(hashes, hashMPQ("(hash table)", 3)), encryptTest(blocks, hashMPQ("(block table)", 3)), payload}, nil)
}

func correlationDetailsTest(t *testing.T, lobby []byte, clock time.Time, firstTeamWins bool) []byte {
	t.Helper()
	l, err := lobbydata.Parse(lobby)
	if err != nil {
		t.Fatal(err)
	}
	players := []byte{0}
	players = append(players, vintTest(10)...)
	for i, p := range l.Players {
		if p.Role == "Watc" {
			continue
		}
		parts := strings.Split(p.ToonHandle, "-")
		id, err := strconv.ParseInt(parts[3], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		toon := structTest(vintTest(0), intTest(1), vintTest(1), append([]byte{7}, []byte("Hero")...), vintTest(2), intTest(1), vintTest(4), intTest(id))
		team := int64(i / 5)
		result := int64(2)
		if (team == 0) == firstTeamWins {
			result = 1
		}
		player := structTest(vintTest(0), blobTest(fmt.Sprintf("Player%d", i)), vintTest(1), toon, vintTest(5), intTest(team), vintTest(8), intTest(result))
		players = append(players, player...)
	}
	ticks := clock.UnixMilli()*10000 + 116444736000000000
	return structTest(vintTest(0), players, vintTest(1), blobTest("Test map"), vintTest(5), intTest(ticks))
}

func writeCorrelationFile(t *testing.T, root, name string, raw []byte, at time.Time) string {
	t.Helper()
	path := filepath.Join(root, name+".StormReplay")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
	return path
}

func boundScanFixture(t *testing.T, at time.Time) (*Engine, config.Paths, []byte) {
	t.Helper()
	e, p := scanFixture(t, at)
	raw := realLobbyTest(t)
	e.state.Active.Match = bindingTest(t, raw, at.Add(-time.Second))
	e.cfg.PlayerToonHandle = e.state.Active.Match.PlayerToon
	if err := e.updateAndSave(e.state); err != nil {
		t.Fatal(err)
	}
	return e, p, raw
}

func TestRealPairedLobbyAndReplay(t *testing.T) {
	raw := realLobbyTest(t)
	archive, err := os.ReadFile("../../testdata/battlelobby/HanamuraSL1_89566.StormReplay")
	if err != nil {
		t.Fatal(err)
	}
	member, err := replay.ReadMPQMember(archive, "replay.server.battlelobby")
	if err != nil || !bytes.Equal(member, raw) {
		t.Fatalf("real pair is not byte-identical: %v", err)
	}
	details, err := replay.ReadMPQMember(archive, "replay.details")
	if err != nil {
		t.Fatal(err)
	}
	replay, err := replay.ParseDetails(details)
	if err != nil {
		t.Fatal(err)
	}
	if replay.RecordedAt != 1674204080951 || replay.Map != "Hanamura Temple" {
		t.Fatalf("wrong real replay details: %+v", replay)
	}
	observed := time.UnixMilli(replay.RecordedAt)
	now := observed.Add(20 * time.Minute)
	root := t.TempDir()
	writeCorrelationFile(t, root, "paired", archive, now.Add(-10*time.Second))
	p := &predmodel.Prediction{StartedAt: observed.Add(5 * time.Second).UnixMilli(), Match: bindingTest(t, raw, observed)}
	result, code, _, detail := correlatePrediction(config.Data{ReplayRoot: root}, p, now)
	if result == nil || result.Player.Result != "loss" || code != "" {
		t.Fatalf("paired match failed: %+v %s %s", result, code, detail)
	}
}

func TestCorrelationCandidates(t *testing.T) {
	now := time.Now()
	observed := now.Add(-time.Minute)
	lobby := realLobbyTest(t)
	valid := correlationMPQTest(correlationDetailsTest(t, lobby, observed, true), lobby)
	for _, tc := range []struct {
		name  string
		code  string
		files [][]byte
		fresh bool
	}{
		{"unique", "", [][]byte{valid}, false},
		{"identical copies", "", [][]byte{valid, valid}, false},
		{"two distinct results", "correlation_ambiguous", [][]byte{valid, correlationMPQTest(correlationDetailsTest(t, lobby, observed, false), lobby)}, false},
		{"distinct files with same result", "correlation_ambiguous", [][]byte{valid, append(append([]byte(nil), valid...), 0)}, false},
		{"unreadable second file", "correlation_incomplete", [][]byte{valid, []byte("broken")}, false},
		{"writing second file", "writing", [][]byte{valid, valid}, true},
		{"old clock with fresh mtime", "correlation_incomplete", [][]byte{correlationMPQTest(correlationDetailsTest(t, lobby, now.Add(-24*time.Hour), true), lobby)}, false},
		{"future game clock", "correlation_incomplete", [][]byte{correlationMPQTest(correlationDetailsTest(t, lobby, now.Add(time.Hour), true), lobby)}, false},
		{"missing clock", "correlation_incomplete", [][]byte{correlationMPQTest(detailsTest(), lobby)}, false},
		{"wrong roster", "correlation_incomplete", [][]byte{correlationMPQTest(correlationDetailsTest(t, realARAMLobbyTest(t), observed, true), lobby)}, false},
		{"different lobby", "correlation_mismatch", [][]byte{correlationMPQTest(correlationDetailsTest(t, lobby, observed, true), realARAMLobbyTest(t))}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for i, raw := range tc.files {
				at := now.Add(-10 * time.Second)
				if tc.fresh && i == len(tc.files)-1 {
					at = now
				}
				writeCorrelationFile(t, root, fmt.Sprint(i), raw, at)
			}
			p := &predmodel.Prediction{StartedAt: observed.Add(time.Second).UnixMilli(), Match: bindingTest(t, lobby, observed)}
			result, code, _, detail := correlatePrediction(config.Data{ReplayRoot: root}, p, now)
			if code != tc.code || (result != nil) != (tc.code == "") {
				t.Fatalf("result %+v code %s detail %s", result, code, detail)
			}
		})
	}
}

func realARAMLobbyTest(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../testdata/battlelobby/replay.server.ARAM1_89754.battlelobby")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestBindingIsFrozenAndRepeatedLobbyIsAmbiguous(t *testing.T) {
	now := time.Now()
	m := bindingTest(t, realLobbyTest(t), now)
	p := &predmodel.Prediction{Match: m}
	snapshot := m.Lobby
	frozen := correlation.BindPredictionMatch(snapshot, "original", m.PlayerToon)
	snapshot.Data.Players[0].Hero = "changed"
	if frozen.Lobby.Data.Players[0].Hero == "changed" {
		t.Fatal("binding aliases mutable observed roster")
	}
	p.Match = frozen
	next := bindingTest(t, realARAMLobbyTest(t), now.Add(time.Minute)).Lobby
	got := correlation.ObserveActiveLobby(p, next, "next")
	if got.Match != frozen || got.Match.Ambiguous {
		t.Fatal("new match rebound the old Prediction")
	}
	got = correlation.ObserveActiveLobby(p, frozen.Lobby, "rewritten")
	if !got.Match.Ambiguous || p.Match.Ambiguous || got.Match.Lobby != frozen.Lobby {
		t.Fatal("repeat uncertainty was lost or shared state mutated")
	}
}

func TestSameRosterAndSeedAreInsufficient(t *testing.T) {
	now := time.Now()
	lobby := realLobbyTest(t)
	different := append([]byte(nil), lobby...)
	different[len(different)-1] ^= 1
	original, err := lobbydata.Parse(lobby)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := lobbydata.Parse(different)
	if err != nil {
		t.Fatal(err)
	}
	if original.RandomSeed != parsed.RandomSeed || original.Build != parsed.Build {
		t.Fatal("test must keep seed and build unchanged")
	}
	root := t.TempDir()
	writeCorrelationFile(t, root, "same-roster", correlationMPQTest(correlationDetailsTest(t, lobby, now.Add(-time.Minute), true), different), now.Add(-10*time.Second))
	p := &predmodel.Prediction{StartedAt: now.Add(-time.Minute).UnixMilli(), Match: bindingTest(t, lobby, now.Add(-time.Minute))}
	if result, code, _, _ := correlatePrediction(config.Data{ReplayRoot: root}, p, now); result != nil || code != "correlation_mismatch" {
		t.Fatalf("weak fallback accepted: %+v %s", result, code)
	}
}

func TestInconsistentRosterOrOutcomeCannotPay(t *testing.T) {
	raw := realLobbyTest(t)
	lobby, err := lobbydata.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*replay.Replay)
	}{
		{"duplicate identity", func(r *replay.Replay) { r.Players[1].ToonHandle = r.Players[0].ToonHandle }},
		{"unknown outcome", func(r *replay.Replay) { r.Players[0].Result = "unknown" }},
		{"mixed team outcomes", func(r *replay.Replay) { r.Players[0].Result = "loss" }},
		{"both teams win", func(r *replay.Replay) {
			for i := range r.Players {
				r.Players[i].Result = "win"
			}
		}},
		{"invalid team", func(r *replay.Replay) { r.Players[0].Team = 2 }},
		{"unbalanced teams", func(r *replay.Replay) { r.Players[0].Team = 1; r.Players[0].Result = "loss" }},
		{"missing participant", func(r *replay.Replay) { r.Players = r.Players[1:] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			replay, err := replay.ParseDetails(correlationDetailsTest(t, raw, time.Now(), true))
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(&replay)
			if _, err := correlation.CorrelatedOutcome(replay, &lobby, lobby.Players[0].ToonHandle); err == nil {
				t.Fatal("inconsistent match accepted")
			}
		})
	}
}

func TestVerifiedLiveMatchCallsOnlyBoundPredictionOnce(t *testing.T) {
	now := time.Now()
	e, p, lobby := boundScanFixture(t, now.Add(-time.Minute))
	active := e.state.Active
	active.DryRun = false
	active.ID = "bound-prediction"
	active.BroadcasterID = "channel"
	active.WinID = "winning"
	active.LossID = "losing"
	testTokenPaths.Store(e.twitch, e.p.Secret)
	seedToken(e.twitch, twitchpkg.Token{AccessToken: "fake-token", ExpiresIn: 3600, SavedAt: now.Unix()})
	writeCorrelationFile(t, p.Dir, "valid", correlationMPQTest(correlationDetailsTest(t, lobby, now.Add(-time.Minute), true), lobby), now.Add(-10*time.Second))
	oldClient := httpClient
	defer func() { httpClient = oldClient }()
	requests := 0
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method == "GET" {
			return manualResponse(`{"data":[{"id":"bound-prediction","status":"ACTIVE","outcomes":[{"id":"winning","channel_points":100},{"id":"losing","channel_points":200}]}]}`), nil
		}
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if req.Method != "PATCH" || req.URL.Path != "/helix/predictions" || payload["id"] != "bound-prediction" || payload["broadcaster_id"] != "channel" || payload["winning_outcome_id"] != "winning" || payload["status"] != "RESOLVED" {
			t.Fatalf("wrong target or outcome: %s %s %+v", req.Method, req.URL, payload)
		}
		return manualResponse(`{"data":[{"id":"bound-prediction","status":"RESOLVED","winning_outcome_id":"winning"}]}`), nil
	})}
	started := active.StartedAt
	status, err := e.RecheckReplay(context.Background(), active.ID, started)
	if err != nil || status.Code != "settled" || requests != 2 {
		t.Fatalf("verified settlement: %+v %v requests=%d", status, err, requests)
	}
	if _, err := e.RecheckReplay(context.Background(), active.ID, started); err == nil || requests != 2 {
		t.Fatal("repeated recheck called Twitch twice")
	}
}

func customLobbyTest(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../testdata/battlelobby/replay.server.IndustrialDistrict1_98348.battlelobby")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// Details mirroring the real 2026-10-17 custom game on build 98348: one
// verified human and AI participants carrying synthetic zero handles, with
// teams 5/1 and consistent per-team outcomes.
func customAIDetailsTest(t *testing.T, clock time.Time, humanWin, aiTeammatesWin, intruder bool) []byte {
	t.Helper()
	hero := structTest(vintTest(0), intTest(2), vintTest(1), append([]byte{7}, []byte("Hero")...), vintTest(2), intTest(1), vintTest(4), intTest(7669040))
	otherHero := structTest(vintTest(0), intTest(9), vintTest(1), append([]byte{7}, []byte("Hero")...), vintTest(2), intTest(9), vintTest(4), intTest(9999999))
	ai := structTest(vintTest(0), intTest(0), vintTest(1), append([]byte{7}, []byte("\x00\x00\x00\x00")...), vintTest(2), intTest(0), vintTest(4), intTest(0))
	entry := func(name string, toon []byte, team int64, win bool) []byte {
		result := int64(2)
		if win {
			result = 1
		}
		return structTest(vintTest(0), blobTest(name), vintTest(1), toon, vintTest(5), intTest(team), vintTest(8), intTest(result))
	}
	count := 6
	if intruder {
		count = 7
	}
	players := []byte{0}
	players = append(players, vintTest(int64(count))...)
	players = append(players, entry("Daigert", hero, 0, humanWin)...)
	for _, name := range []string{"Player 2", "Player 3", "Player 4", "Player 5"} {
		players = append(players, entry(name, ai, 0, aiTeammatesWin)...)
	}
	players = append(players, entry("Player 10", ai, 1, !humanWin)...)
	if intruder {
		players = append(players, entry("Intruder", otherHero, 1, !humanWin)...)
	}
	ticks := clock.UnixMilli()*10000 + 116444736000000000
	return structTest(vintTest(0), players, vintTest(1), blobTest("Industrial District"), vintTest(5), intTest(ticks))
}

func customAIBindingTest(t *testing.T, raw []byte, at time.Time, toon string) *predmodel.MatchBinding {
	t.Helper()
	lobby, err := lobbydata.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return correlation.BindPredictionMatch(&lobbydata.Snapshot{Status: "roster_parsed", Parser: "battlelobby-roster-v1", SHA256: correlation.ContentSHA(raw), ModifiedAt: at.UnixMilli(), Size: int64(len(raw)), Data: &lobby}, "original", toon)
}

func TestCustomAILobbySettlesAutomatically(t *testing.T) {
	for _, tc := range []struct {
		name       string
		humanWin   bool
		wantResult string
	}{
		{"win", true, "win"},
		{"loss", false, "loss"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			observed := now.Add(-time.Minute)
			raw := customLobbyTest(t)
			lobby, err := lobbydata.Parse(raw)
			if err != nil {
				t.Fatal(err)
			}
			if !lobby.Custom || lobby.TeamSize != "5v5" {
				t.Fatalf("fixture is not a 5v5 custom lobby: %+v", lobby)
			}
			archive := correlationMPQTest(customAIDetailsTest(t, observed.Add(30*time.Second), tc.humanWin, tc.humanWin, false), raw)
			root := t.TempDir()
			writeCorrelationFile(t, root, "custom-ai-"+tc.name, archive, now.Add(-10*time.Second))
			p := &predmodel.Prediction{StartedAt: observed.Add(10 * time.Second).UnixMilli(), Match: customAIBindingTest(t, raw, observed, "2-Hero-1-7669040")}
			result, code, file, detail := correlatePrediction(config.Data{ReplayRoot: root}, p, now)
			if result == nil || result.Player.Result != tc.wantResult || code != "" {
				t.Fatalf("custom AI game did not correlate: %+v %s %s(%s)", result, code, file, detail)
			}
		})
	}
}

func TestCustomAILobbyReplaySafety(t *testing.T) {
	now := time.Now()
	observed := now.Add(-time.Minute)
	raw := customLobbyTest(t)
	for _, tc := range []struct {
		name     string
		toon     string
		details  func() []byte
		wantCode string
	}{
		{"unaccounted hero handle", "2-Hero-1-7669040", func() []byte {
			return customAIDetailsTest(t, observed.Add(30*time.Second), true, true, true)
		}, "correlation_incomplete"},
		{"binding outside roster", "9-Hero-9-9999999", func() []byte {
			return customAIDetailsTest(t, observed.Add(30*time.Second), true, true, false)
		}, "correlation_unavailable"},
		{"inconsistent team outcome", "2-Hero-1-7669040", func() []byte {
			return customAIDetailsTest(t, observed.Add(30*time.Second), true, false, false)
		}, "correlation_incomplete"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeCorrelationFile(t, root, "custom-ai-"+tc.name, correlationMPQTest(tc.details(), raw), now.Add(-10*time.Second))
			p := &predmodel.Prediction{StartedAt: observed.Add(10 * time.Second).UnixMilli(), Match: customAIBindingTest(t, raw, observed, tc.toon)}
			result, code, _, _ := correlatePrediction(config.Data{ReplayRoot: root}, p, now)
			if result != nil || code != tc.wantCode {
				t.Fatalf("unsafe custom replay accepted: %+v %s", result, code)
			}
		})
	}
}

func TestLegacyAndUnusableBindingsNeverAutoSettle(t *testing.T) {
	now := time.Now()
	for _, modify := range []func(*predmodel.Prediction){
		func(p *predmodel.Prediction) { p.Match = nil },
		func(p *predmodel.Prediction) { p.Match.PlayerToon = "" },
		func(p *predmodel.Prediction) { p.Match.PlayerToon = "different" },
		func(p *predmodel.Prediction) { p.Match.Lobby.Status = "parse_error" },
		func(p *predmodel.Prediction) { p.Match.Lobby.Data.Players[1].SlotType = "Comp" },
		func(p *predmodel.Prediction) {
			p.Match.Lobby.Data.Custom = true
			p.Match.Lobby.Data.Players[0].SlotType = "Comp"
		},
		func(p *predmodel.Prediction) { p.Match.Lobby.Data.Players[0].Role = "Watc" },
		func(p *predmodel.Prediction) { p.Match.Version = 2 },
		func(p *predmodel.Prediction) { p.StartedAt = now.Add(-time.Hour).UnixMilli() },
	} {
		p := &predmodel.Prediction{StartedAt: now.UnixMilli(), Match: bindingTest(t, realLobbyTest(t), now)}
		modify(p)
		if result, code, _, _ := correlatePrediction(config.Data{ReplayRoot: t.TempDir()}, p, now); result != nil || code != "correlation_unavailable" {
			t.Fatalf("unsafe binding accepted: %+v %s", result, code)
		}
	}
}

func TestRecheckCorrelationPersistsAcrossRestart(t *testing.T) {
	now := time.Now()
	e, p, lobby := boundScanFixture(t, now.Add(-time.Minute))
	writeCorrelationFile(t, p.Dir, "match", correlationMPQTest(correlationDetailsTest(t, lobby, now.Add(-time.Minute), true), lobby), now.Add(-10*time.Second))
	restarted, err := newEngine(p, config.Data{ReplayRoot: p.Dir, PlayerToonHandle: "changed-settings"}, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	status, err := restarted.RecheckReplay(context.Background(), e.state.Active.ID, e.state.Active.StartedAt)
	if err != nil || status.Code != "settled" || restarted.state.Active != nil {
		t.Fatalf("restart failed: %+v %v", status, err)
	}
}

func TestAmbiguousLivePredictionNeverCallsTwitchAfterRestart(t *testing.T) {
	now := time.Now()
	e, p, lobby := boundScanFixture(t, now.Add(-time.Minute))
	e.state.Active.DryRun = false
	e.cfg.RealMode = true
	winning := correlationMPQTest(correlationDetailsTest(t, lobby, now.Add(-time.Minute), true), lobby)
	losing := correlationMPQTest(correlationDetailsTest(t, lobby, now.Add(-time.Minute), false), lobby)
	writeCorrelationFile(t, p.Dir, "win", winning, now.Add(-10*time.Second))
	second := writeCorrelationFile(t, p.Dir, "loss", losing, now.Add(-10*time.Second))
	oldClient := httpClient
	defer func() { httpClient = oldClient }()
	requests := 0
	httpClient = &http.Client{Transport: manualTransport(func(req *http.Request) (*http.Response, error) {
		requests++
		t.Errorf("unsafe Twitch request: %s %s", req.Method, req.URL)
		return manualResponse(`{"data":[]}`), nil
	})}
	// No OAuth request should occur either; only the correlation gate is evaluated.
	status, err := e.RecheckReplay(context.Background(), e.state.Active.ID, e.state.Active.StartedAt)
	if err != nil || status.Code != "correlation_ambiguous" {
		t.Fatalf("ambiguity missed: %+v %v", status, err)
	}
	if err := os.Remove(second); err != nil {
		t.Fatal(err)
	}
	restarted, err := newEngine(p, e.cfg, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	status, err = restarted.RecheckReplay(context.Background(), e.state.Active.ID, e.state.Active.StartedAt)
	if err != nil || status.Code != "correlation_ambiguous" || restarted.state.Active == nil || requests != 0 {
		t.Fatalf("restart unlocked payout: %+v %v requests=%d", status, err, requests)
	}
}

func TestInventoryLimitAndChangedReplayBlockCorrelation(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	lobby := realLobbyTest(t)
	for i := 0; i < 129; i++ {
		writeCorrelationFile(t, root, fmt.Sprint(i), []byte("unknown"), now.Add(-10*time.Second))
	}
	p := &predmodel.Prediction{StartedAt: now.Add(-time.Minute).UnixMilli(), Match: bindingTest(t, lobby, now.Add(-time.Minute))}
	if result, code, _, _ := correlatePrediction(config.Data{ReplayRoot: root}, p, now); result != nil || code != "correlation_incomplete" {
		t.Fatalf("incomplete inventory accepted: %+v %s", result, code)
	}
	if _, err := correlationReplayFiles(config.Data{ReplayRoot: filepath.Join(root, "missing")}, time.Time{}); err == nil {
		t.Fatal("missing configured folder silently accepted")
	}
	path := writeCorrelationFile(t, t.TempDir(), "changed", bytes.Repeat([]byte{1}, 64), now)
	if _, err := correlation.ReadStableReplay(discovery.ReplayFile{Path: path, Size: 32, Modified: now}); err == nil {
		t.Fatal("changed replay accepted")
	}
}

func TestCreationBindsExactObservationAndPlayer(t *testing.T) {
	now := time.Now()
	e, p := scanFixture(t, now)
	e.state.Active = nil
	original := bindingTest(t, realLobbyTest(t), now.Add(-time.Second))
	e.state.LastLobby = original.Lobby
	e.cfg.PlayerToonHandle = original.PlayerToon
	if err := e.doCreate(context.Background(), &e.state, "created-key"); err != nil {
		t.Fatal(err)
	}
	restored, err := loadState(p)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Active == nil || restored.Active.Match == nil || restored.Active.Match.LobbyKey != "created-key" || restored.Active.Match.PlayerToon != original.PlayerToon || restored.Active.Match.Lobby.SHA256 != original.Lobby.SHA256 {
		t.Fatalf("creation lost binding: %+v", restored.Active)
	}
	e.state.LastLobby.Data.Players[0].Hero = "later"
	if restored.Active.Match.Lobby.Data.Players[0].Hero == "later" {
		t.Fatal("binding is not frozen")
	}
}

func TestRestartDetectsRepeatedLobbyBeforeWatchBegan(t *testing.T) {
	now := time.Now()
	e, p, lobby := boundScanFixture(t, now.Add(-time.Minute))
	temp := t.TempDir()
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	root := filepath.Join(os.TempDir(), "Heroes of the Storm", "TempWriteReplayP1")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "replay.server.battlelobby")
	if err := os.WriteFile(path, lobby, 0600); err != nil {
		t.Fatal(err)
	}
	changedAt := now.Add(-20 * time.Second)
	if err := os.Chtimes(path, changedAt, changedAt); err != nil {
		t.Fatal(err)
	}
	writeCorrelationFile(t, p.Dir, "match", correlationMPQTest(correlationDetailsTest(t, lobby, now.Add(-time.Minute), true), lobby), now.Add(-10*time.Second))
	lastError := ""
	lastLog := time.Time{}
	e.scanTick(context.Background(), now, &lastError, &lastLog)
	restored, err := loadState(p)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Active == nil || !restored.Active.Match.Ambiguous || e.scan.Code != "correlation_ambiguous" {
		t.Fatalf("startup ignored changed lobby: %+v %+v", restored, e.scan)
	}
}
