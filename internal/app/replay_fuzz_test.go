package app

import (
	"bytes"
	"compress/zlib"
	"hots-twitch-companion/internal/hots/replay"
	"testing"
)

// FuzzMPQMember subjects archive table indices, sector offsets, compression
// headers and nested replay.details parsing to arbitrary input. Go's normal
// go test executes these deterministic seed cases; go test -fuzz=FuzzMPQMember
// explores mutated inputs locally.
func FuzzMPQMember(f *testing.F) {
	details := detailsTest()
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	_, _ = zw.Write(details)
	_ = zw.Close()
	f.Add(buildTestMPQ(append([]byte{0x02}, compressed.Bytes()...), details))
	f.Add([]byte{})
	f.Add([]byte{'M', 'P', 'Q', 0x1a})
	f.Add(bytes.Repeat([]byte{0xff}, 32))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		member, err := replay.ReadMPQMember(data, "replay.details")
		if err == nil {
			_, _ = replay.ParseDetails(member)
		}
	})
}

// FuzzReplayDetails exercises the Blizzard versioned-data decoder with
// malformed lengths, nested structs and partial UTF-8 without requiring
// filesystem access or Twitch credentials.
func FuzzReplayDetails(f *testing.F) {
	f.Add(detailsTest())
	f.Add([]byte{})
	f.Add([]byte{5, 0})
	f.Add(bytes.Repeat([]byte{0xff}, 40))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 1<<20 {
			t.Skip()
		}
		_, _ = replay.ParseDetails(raw)
	})
}
