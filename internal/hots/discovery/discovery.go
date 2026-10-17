// Package discovery scans on-disk Heroes of the Storm lobbies and replays.
// All access is read-only and bounded by depth/file-count limits.
package discovery

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var ErrLobbyNotFound = errors.New("battlelobby not found yet")

type ReplayFile struct {
	Path     string
	Modified time.Time
	Size     int64
}

func DefaultReplayRoots() []string {
	home, _ := os.UserHomeDir()
	return []string{filepath.Join(home, "Documents", "Heroes of the Storm"), filepath.Join(home, "OneDrive", "Documents", "Heroes of the Storm")}
}
func ReplayRoots(customRoot string) []string {
	if customRoot != "" {
		return []string{customRoot}
	}
	return DefaultReplayRoots()
}
func ReplaysAfter(customRoot string, after time.Time, max int) []ReplayFile {
	var found []ReplayFile
	visited := 0
	for _, root := range ReplayRoots(customRoot) {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return filepath.SkipDir
			}
			visited++
			if visited > 20000 {
				return filepath.SkipAll
			}
			if d.IsDir() {
				rel, er := filepath.Rel(root, path)
				if er == nil && strings.Count(rel, string(os.PathSeparator)) > 8 {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(strings.ToLower(d.Name()), ".stormreplay") {
				st, e := d.Info()
				if e == nil && st.ModTime().After(after) {
					found = append(found, ReplayFile{path, st.ModTime(), st.Size()})
				}
			}
			return nil
		})
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Modified.After(found[j].Modified) })
	if len(found) > max {
		return found[:max]
	}
	return found
}
func FindLobby() (string, string, time.Time, error) {
	root := filepath.Join(os.TempDir(), "Heroes of the Storm")
	var newest string
	var st os.FileInfo
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return filepath.SkipDir
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(root, p)
			if strings.Count(rel, string(os.PathSeparator)) >= 3 {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(d.Name(), "replay.server.battlelobby") {
			s, err := d.Info()
			if err == nil && s.Size() > 0 && (st == nil || s.ModTime().After(st.ModTime())) {
				newest = p
				st = s
			}
		}
		return nil
	})
	if st == nil {
		return "", "", time.Time{}, ErrLobbyNotFound
	}
	key := fmt.Sprintf("%s|%d|%d", newest, st.ModTime().UnixNano(), st.Size())
	return newest, key, st.ModTime(), nil
}


// CountStormReplays counts replay files in a bounded, read-only directory walk.
func CountStormReplays(root string) int {
 count := 0
 visited := 0
 _ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
  if err != nil { return filepath.SkipDir }
  visited++
  if visited > 20000 { return filepath.SkipAll }
  if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".stormreplay") { count++ }
  return nil
 })
 return count
}
