package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReleaseVersionValidation(t *testing.T) {
	tests := []struct {
		latest, current string
		newer           bool
	}{
		{"v0.9.3", "0.9.2", true},
		{"v0.10.0", "0.9.9", true},
		{"v0.9.2", "0.9.2", false},
		{"v0.9", "0.9.2", false},
		{"v0.9.3.1", "0.9.2", false},
		{"v0.x.3", "0.9.2", false},
		{"v999999999999999999999999999999.0.0", "0.9.2", false},
		{"v0.9.3", "bad", false},
		{"v", "0.9.2", false},
	}
	for _, tt := range tests {
		if got := IsNewerRelease(tt.latest, tt.current); got != tt.newer {
			t.Errorf("IsNewerRelease(%q,%q) = %v, want %v", tt.latest, tt.current, got, tt.newer)
		}
	}
}
func TestFetchLatestRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "HotS-Twitch-Companion" {
			t.Error("missing user agent")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v0.9.3","html_url":"https://github.com/cactuargg/hots-twitch-companion/releases/tag/v0.9.3"}`))
	}))
	defer server.Close()
	info, err := FetchLatestRelease(context.Background(), server.URL)
	if err != nil || info.Tag != "v0.9.3" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}
