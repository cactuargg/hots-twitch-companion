package discovery

import (
 "os"
 "path/filepath"
 "testing"
)

func TestCountStormReplays(t *testing.T) {
 root := t.TempDir()
 nested := filepath.Join(root,"Accounts","Example","Replays")
 if err := os.MkdirAll(nested,0700); err != nil { t.Fatal(err) }
 for _, name := range []string{"one.StormReplay","two.stormreplay","not-a-replay.txt"} {
  if err := os.WriteFile(filepath.Join(nested,name), []byte("test"),0600); err != nil { t.Fatal(err) }
 }
 if got := CountStormReplays(root); got != 2 { t.Fatalf("got %d replays, want 2",got) }
 if got := CountStormReplays(filepath.Join(root,"does-not-exist")); got != 0 {t.Fatalf("missing root count=%d",got)}
}
