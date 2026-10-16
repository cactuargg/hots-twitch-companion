package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Last observed lobby, independent of any Prediction. No settlement logic reads it.
// SHA256 identifies bytes in this file, never a shared lobby/replay Match ID.
type LobbySnapshot struct {
	SHA256     string       `json:"sha256,omitempty"`
	File       string       `json:"file"`
	ModifiedAt int64        `json:"modifiedAt,omitempty"`
	Size       int64        `json:"size,omitempty"`
	Parser     string       `json:"parser"`
	Status     string       `json:"status"`
	Error      string       `json:"error,omitempty"`
	Data       *BattleLobby `json:"data,omitempty"`
}

func captureBattleLobby(path string) *LobbySnapshot {
	snapshot := &LobbySnapshot{File: filepath.Base(path), Parser: "battlelobby-roster-v1", Status: "read_error"}
	fail := func(err error) *LobbySnapshot { snapshot.Error = err.Error(); return snapshot }
	f, err := os.Open(path)
	if err != nil {
		return fail(err)
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return fail(err)
	}
	if !before.Mode().IsRegular() || before.Size() < 1 || before.Size() > maxLobbyBytes {
		return fail(errors.New("battlelobby: invalid file type or size"))
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxLobbyBytes+1))
	if err != nil {
		return fail(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		return fail(err)
	}
	if !os.SameFile(before, after) || int64(len(raw)) != before.Size() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return fail(errors.New("battlelobby changed during capture"))
	}
	hash := sha256.Sum256(raw)
	snapshot.SHA256 = hex.EncodeToString(hash[:])
	snapshot.Size = before.Size()
	snapshot.ModifiedAt = before.ModTime().UnixMilli()
	data, err := parseBattleLobby(raw)
	if err != nil {
		snapshot.Status = "parse_error"
		return fail(err)
	}
	snapshot.Status = "roster_parsed"
	snapshot.Data = &data
	return snapshot
}
