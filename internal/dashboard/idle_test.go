package dashboard

import (
	"hots-twitch-companion/internal/config"
	"testing"
)

func TestBrowserFallbackInitializesIdleClockBeforeOpeningBrowser(t *testing.T) {
	d := noticeDashboard(t, config.NewPaths(t.TempDir()))
	opened := false
	err := d.RunDesktop(func(url string) error {
		opened = true
		d.mu.Lock()
		activity := d.lastActivity
		d.mu.Unlock()
		if activity.IsZero() || url == "" {
			t.Error("browser fallback must start its idle clock before opening")
		}
		// Exit immediately instead of waiting for the idle watchdog.
		d.Close()
		return nil
	})
	if err != nil || !opened {
		t.Fatalf("browser fallback failed: opened=%v error=%v", opened, err)
	}
}
