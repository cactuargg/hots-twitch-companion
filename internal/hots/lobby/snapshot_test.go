package lobby

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureFixtureHashAndRoster(t *testing.T) {
	path:=filepath.Join("..","..","..","testdata","battlelobby","replay.server.ARAM1_89754.battlelobby")
	raw,err:=os.ReadFile(path)
	if err!=nil{t.Fatal(err)}
	s:=Capture(path)
	sum:=sha256.Sum256(raw)
	if s.Status!="roster_parsed" || s.SHA256!=hex.EncodeToString(sum[:]) ||
		s.Data==nil || s.Data.Build!=89754 || s.Size!=int64(len(raw)) {
		t.Fatalf("captured roster is inconsistent: %+v",s)
	}
}

func TestCaptureInvalidBytesKeepsHashAndRejectsRoster(t *testing.T) {
	file:=filepath.Join(t.TempDir(),"replay.server.battlelobby")
	if err:=os.WriteFile(file,[]byte("broken lobby data"),0600);err!=nil {t.Fatal(err)}
	s:=Capture(file)
	if s.Status!="parse_error" || s.SHA256=="" || s.Data!=nil || s.Error=="" {
		t.Fatalf("invalid snapshot was treated as a usable roster: %+v",s)
	}
	if absent:=Capture(filepath.Join(t.TempDir(),"missing"));absent.Status!="read_error" || absent.Data!=nil {
		t.Fatalf("missing snapshot was accepted: %+v",absent)
	}
}
