package main

import (
 "context"
 "strings"
 "time"

 dashview "hots-twitch-companion/internal/dashboard"
)

// Update metadata fetch and version parsing live in internal/dashboard.
const releasesLatestURL = "https://api.github.com/repos/cactuargg/hots-twitch-companion/releases/latest"
type releaseInfo = dashview.ReleaseInfo
func fetchLatestRelease(ctx context.Context, endpoint string) (releaseInfo, error) {
 return dashview.FetchLatestRelease(ctx, endpoint)
}
func isNewerRelease(latest, current string) bool {
 return dashview.IsNewerRelease(latest, current)
}

func (d *dashboard) watchUpdates() {
 d.checkUpdate()
 ticker := time.NewTicker(24 * time.Hour)
 defer ticker.Stop()
 for {
  select {
  case <-d.shutdown: return
  case <-ticker.C: d.checkUpdate()
  }
 }
}
func (d *dashboard) checkUpdate() { _ = d.checkUpdateAt(releasesLatestURL) }
func (d *dashboard) checkUpdateAt(endpoint string) error {
 ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
 defer cancel()
 info, err := fetchLatestRelease(ctx, endpoint)
 if err != nil { return err }
 if !isNewerRelease(info.Tag, appVersion) { return nil }
 d.mu.Lock()
 d.updateVersion = strings.TrimPrefix(info.Tag, "v")
 d.updateURL = info.HTMLURL
 d.mu.Unlock()
 return nil
}
