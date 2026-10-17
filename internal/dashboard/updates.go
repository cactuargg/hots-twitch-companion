package dashboard

import (
	"context"
	"strings"
	"time"
)

const releasesLatestURL = "https://api.github.com/repos/cactuargg/hots-twitch-companion/releases/latest"

func (d *Dashboard) watchUpdates() {
	d.checkUpdate()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-d.shutdown:
			return
		case <-ticker.C:
			d.checkUpdate()
		}
	}
}
func (d *Dashboard) checkUpdate() { _ = d.checkUpdateAt(releasesLatestURL) }
func (d *Dashboard) checkUpdateAt(endpoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	info, err := FetchLatestRelease(ctx, endpoint)
	if err != nil {
		return err
	}
	if !IsNewerRelease(info.Tag, d.version) {
		return nil
	}
	d.mu.Lock()
	d.updateVersion = strings.TrimPrefix(info.Tag, "v")
	d.updateURL = info.HTMLURL
	d.mu.Unlock()
	return nil
}
