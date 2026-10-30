package lobby

import "testing"

func TestHeroCatalogMatchesReplayNamesAndLocales(t *testing.T) {
	for _, tc := range []struct{ code, name string }{{"Jain", "Jaina"}, {"L90E", "E.T.C."}, {"Amaz", "Cassia"}, {"Tink", "Gazlowe"}, {"Witc", "Nazeebo"}, {"Jain", "Джайна"}} {
		if !HeroMatches(tc.code, tc.name) || HeroName(tc.code, "en") == "" {
			t.Fatalf("unmapped hero: %+v", tc)
		}
	}
	if HeroName("????", "ru") != "" || HeroMatches("Jain", "Valla") {
		t.Fatal("unknown or mismatched hero guessed")
	}
	for code, h := range heroCatalog {
		if h.en == "" || h.ru == "" || !HeroMatches(code, h.en) || !HeroMatches(code, h.ru) {
			t.Fatal("incomplete hero entry", code)
		}
	}
}
