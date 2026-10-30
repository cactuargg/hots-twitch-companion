package config

import (
	"reflect"
	"testing"
)

func TestLegacyGameProfileMigrationAndRoundTrip(t *testing.T) {
	legacy := Data{ClientID: "twitch", PlayerName: "Main", PlayerToonHandle: "2-Hero-1-7", ReplayRoot: "D:/HotS", PredictionTitle: "Keep", Lang: "ru", RealMode: true}
	c, err := NormalizeProfiles(legacy)
	if err != nil || len(c.GameProfiles) != 1 || c.ActiveGameProfile != "default" || c.GameProfiles[0].PlayerToonHandle != legacy.PlayerToonHandle || c.ReplayRoot != legacy.ReplayRoot || c.ClientID != legacy.ClientID {
		t.Fatalf("migration: %+v %v", c, err)
	}
	p := NewPaths(t.TempDir())
	if err := Write(p, c); err != nil {
		t.Fatal(err)
	}
	restored, err := Read(p)
	if err != nil || !reflect.DeepEqual(c, restored) {
		t.Fatalf("roundtrip: %+v %v", restored, err)
	}
	c.GameProfiles = append(c.GameProfiles, GameProfile{ID: "second", Name: "Second", PlayerName: "Alt", PlayerToonHandle: "2-Hero-1-8", ReplayRoot: "E:/HotS"})
	c.ActiveGameProfile = "second"
	c.PlayerName = "Alt"
	c.PlayerToonHandle = "2-Hero-1-8"
	c.ReplayRoot = "E:/HotS"
	next, err := NormalizeProfiles(c)
	if err != nil || next.GameProfiles[0].PlayerToonHandle != legacy.PlayerToonHandle || next.ClientID != "twitch" || next.PredictionTitle != "Keep" {
		t.Fatalf("switch: %+v %v", next, err)
	}
	next.GameProfiles[0].Name = "Changed"
	if c.GameProfiles[0].Name != "Main" {
		t.Fatal("normalization mutates caller's slice")
	}
}
func TestGameProfilesRejectInvalidMetadata(t *testing.T) {
	for _, c := range []Data{
		{ActiveGameProfile: "missing"},
		{ActiveGameProfile: "a", GameProfiles: []GameProfile{{ID: "a", Name: "Main"}, {ID: "a", Name: "Alt"}}},
		{ActiveGameProfile: "a", GameProfiles: []GameProfile{{ID: "a", Name: "Main"}, {ID: "b", Name: "MAIN"}}},
		{ActiveGameProfile: "a", GameProfiles: []GameProfile{{ID: "a", Name: " "}}},
		{ActiveGameProfile: "missing", GameProfiles: []GameProfile{{ID: "a", Name: "Main"}}},
		{ActiveGameProfile: "a", GameProfiles: make([]GameProfile, 9)},
	} {
		if _, err := NormalizeProfiles(c); err == nil {
			t.Fatalf("accepted invalid profiles: %+v", c)
		}
	}
}
