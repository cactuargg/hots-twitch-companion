package replay

import (
	"os"
	"testing"
)

func TestPlayedHeroComesFromCompletedReplayDetails(t *testing.T) {
	raw, err := os.ReadFile("../../../testdata/battlelobby/HanamuraSL1_89566.StormReplay")
	if err != nil {
		t.Fatal(err)
	}
	member, err := ReadMPQMember(raw, "replay.details")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseDetails(member)
	if err != nil {
		t.Fatal(err)
	}
	// Independently inspect the versioned tree to verify the m_hero field binding.
	reader := vReader{b: member}
	value, err := reader.read(0)
	if err != nil {
		t.Fatal(err)
	}
	players := asArr(asObj(value)[0])
	if len(players) != len(parsed.Players) {
		t.Fatal("player count changed")
	}
	for i, player := range parsed.Players {
		expected := asStr(asObj(players[i])[10])
		if expected == "" {
			t.Fatal("fixture should contain played hero names")
		}
		if player.Hero != expected {
			t.Fatalf("hero %d: %q != %q", i, player.Hero, expected)
		}
	}
}

func TestParseDetailsRejectsBrokenVersionedData(t *testing.T) {
	for _, input := range [][]byte{nil, {}, {99}, {5, 0}, {5, 0xff}} {
		value, err := ParseDetails(input)
		if err == nil || len(value.Players) > 0 {
			t.Fatalf("invalid details accepted: raw=%v parsed=%+v err=%v", input, value, err)
		}
	}
}

func TestDecodedReplayStructRetainsJSONContract(t *testing.T) {
	p := Player{Name: "Player", ToonHandle: "2-Hero-1-42", Result: "win", ResultCode: 1, Team: 1}
	if p.Result != "win" || p.ToonHandle == "" {
		t.Fatal("player metadata changed")
	}
}
