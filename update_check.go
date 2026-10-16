package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Update checks read public GitHub release metadata only: repository name via
// the URL and no account data is sent. One request per app start plus a daily
// recheck; failures stay silent so offline sessions are unaffected.
const releasesLatestURL = "https://api.github.com/repos/cactuargg/hots-twitch-companion/releases/latest"

type releaseInfo struct {
	Tag     string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func fetchLatestRelease(ctx context.Context, endpoint string) (releaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return releaseInfo{}, err
	}
	// GitHub rejects requests without a User-Agent.
	req.Header.Set("User-Agent", "HotS-Twitch-Companion")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return releaseInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return releaseInfo{}, fmt.Errorf("release check: HTTP %d", resp.StatusCode)
	}
	var info releaseInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&info); err != nil {
		return releaseInfo{}, err
	}
	if !strings.HasPrefix(info.Tag, "v") || info.HTMLURL == "" {
		return releaseInfo{}, errors.New("release check: unexpected response")
	}
	return info, nil
}

// isNewerRelease compares dotted numeric versions such as "v0.9.3" against the
// running appVersion. Malformed or equal versions never report an update, so a
// broken response can at worst hide a notice, never nag the user.
func isNewerRelease(latest, current string) bool {
	parse := func(v string) [3]int {
		var out [3]int
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		for i, part := range strings.Split(v, ".") {
			if i >= len(out) {
				break
			}
			n := 0
			digits := true
			for _, r := range part {
				if r < '0' || r > '9' {
					digits = false
					break
				}
				n = n*10 + int(r-'0')
			}
			if !digits {
				return out
			}
			out[i] = n
		}
		return out
	}
	l, c := parse(latest), parse(current)
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// watchUpdates runs until shutdown: one check shortly after start, then daily.
func (d *dashboard) watchUpdates() {
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

func (d *dashboard) checkUpdate() {
	_ = d.checkUpdateAt(releasesLatestURL)
}

func (d *dashboard) checkUpdateAt(endpoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	info, err := fetchLatestRelease(ctx, endpoint)
	if err != nil {
		return err
	}
	if !isNewerRelease(info.Tag, appVersion) {
		return nil
	}
	d.mu.Lock()
	d.updateVersion = strings.TrimPrefix(info.Tag, "v")
	d.updateURL = info.HTMLURL
	d.mu.Unlock()
	return nil
}
