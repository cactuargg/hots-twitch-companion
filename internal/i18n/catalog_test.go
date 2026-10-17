package i18n

import (
	"strings"
	"testing"
)

func TestFallbackAndNormalization(t *testing.T) {
	if Normalize("ru")!="ru"||Normalize("de")!="en"||Normalize("")!="en"{t.Fatal("language fallback changed")}
	if Text("en","undefined.test.key")!="undefined.test.key"{t.Fatal("missing translation key must return key")}
	if Text("ru","automation.need_twitch")==Text("en","automation.need_twitch"){t.Fatal("Russian localization disappeared")}
	for _,lang:=range []string{"en","ru"}{
		if got:=Text(lang,"log.watch_started","Example");!strings.Contains(got,"Example"){t.Fatalf("%s interpolation failed: %s",lang,got)}
	}
}
