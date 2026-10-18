// A subprocess-only updater fixture: it never opens UI, loads a real profile,
// contacts Twitch or starts monitoring.
package main

import (
	"os"
	"path/filepath"
	"time"
)

func main() {
	if len(os.Args) != 3 && len(os.Args) != 5 {
		return
	}
	if os.Args[1] == "--wait" {
		until := time.Now().Add(30 * time.Second)
		for time.Now().Before(until) {
			if _, err := os.Stat(os.Args[2]); err == nil {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		return
	}
	if os.Args[1] == "--data-dir" {
		if len(os.Args) == 5 && os.Getenv("COMPANION_TEST_FAIL_START") == "1" {
			os.Exit(1)
		}
		_ = os.WriteFile(filepath.Join(os.Args[2], "restarted"), []byte("started"), 0600)
		if len(os.Args) == 5 && os.Args[3] == "--update-ready-file" {
			_ = os.WriteFile(os.Args[4], []byte("started"), 0600)
			time.Sleep(time.Second)
		}
	}
}
