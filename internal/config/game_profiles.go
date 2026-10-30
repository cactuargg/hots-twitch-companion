package config

import (
	"errors"
	"regexp"
	"strings"
)

// Game profiles only select a local HotS player. Twitch and OBS remain shared.
type GameProfile struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	PlayerName       string `json:"playerName"`
	PlayerToonHandle string `json:"playerToonHandle"`
	ReplayRoot       string `json:"replayRoot"`
}

func (c Data) Clone() Data {
	c.GameProfiles = append([]GameProfile(nil), c.GameProfiles...)
	return c
}

var profileID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// NormalizeProfiles keeps the legacy active-player fields as the engine-facing
// configuration and copies edits into the active profile. Legacy configs gain
// a deterministic first profile; no credentials or journal data are changed.
func NormalizeProfiles(c Data) (Data, error) {
	c = c.Clone()
	if len(c.GameProfiles) == 0 {
		if c.ActiveGameProfile != "" {
			return c, errors.New("active game profile is missing")
		}
		if c.PlayerName == "" && c.PlayerToonHandle == "" {
			return c, nil
		}
		name := c.PlayerName
		if name == "" {
			name = "HotS"
		}
		c.GameProfiles = []GameProfile{{ID: "default", Name: name}}
		c.ActiveGameProfile = "default"
	}
	if len(c.GameProfiles) > 8 {
		return c, errors.New("at most 8 game profiles are allowed")
	}
	ids, names := map[string]bool{}, map[string]bool{}
	found := false
	for i := range c.GameProfiles {
		p := &c.GameProfiles[i]
		p.Name = strings.TrimSpace(p.Name)
		if !profileID.MatchString(p.ID) || ids[p.ID] || p.Name == "" || len(p.Name) > 160 || strings.ContainsAny(p.Name, "\r\n\t") || names[strings.ToLower(p.Name)] {
			return c, errors.New("game profiles need unique IDs and names (up to 160 bytes)")
		}
		ids[p.ID] = true
		names[strings.ToLower(p.Name)] = true
		if p.ID == c.ActiveGameProfile {
			found = true
			p.PlayerName = c.PlayerName
			p.PlayerToonHandle = c.PlayerToonHandle
			p.ReplayRoot = c.ReplayRoot
		}
		if len(p.PlayerName) > 160 || len(p.PlayerToonHandle) > 240 || len(p.ReplayRoot) > 1000 {
			return c, errors.New("game profile fields are too long")
		}
	}
	if !found {
		return c, errors.New("active game profile is missing")
	}
	return c, nil
}

func PlayerSelectionChanged(a, b Data) bool {
	return a.ActiveGameProfile != b.ActiveGameProfile || a.PlayerName != b.PlayerName || a.PlayerToonHandle != b.PlayerToonHandle || a.ReplayRoot != b.ReplayRoot
}
