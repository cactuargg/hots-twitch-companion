package dashboard

import (
	"context"
	"hots-twitch-companion/internal/app"
	"hots-twitch-companion/internal/config"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestIsNewerRelease(t *testing.T) {
	for _, tt := range []struct {
		latest, current string
		want            bool
	}{
		{"v0.9.3", "0.9.2", true},
		{"v0.10.0", "0.9.9", true},
		{"v1.0.0", "0.9.9", true},
		{"v0.9.2", "0.9.2", false},
		{"v0.9.2", "0.9.3", false},
		{"v0.9", "0.9.2", false},
		{"", "0.9.2", false},
		{"garbage", "0.9.2", false},
		{"v", "0.9.2", false},
	} {
		if got := IsNewerRelease(tt.latest, tt.current); got != tt.want {
			t.Fatalf("IsNewerRelease(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestFetchLatestReleaseLegacyCoverage(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request must carry a User-Agent header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"v0.9.3","html_url":"https://example.com/release"}`))
	}))
	defer ok.Close()
	info, err := FetchLatestRelease(context.Background(), ok.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.Tag != "v0.9.3" || info.HTMLURL != "https://example.com/release" {
		t.Fatalf("unexpected release info: %#v", info)
	}

	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFound.Close()
	if _, err := FetchLatestRelease(context.Background(), notFound.URL); err == nil {
		t.Fatal("404 must fail the release check")
	}

	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"nope"}`))
	}))
	defer broken.Close()
	if _, err := FetchLatestRelease(context.Background(), broken.URL); err == nil {
		t.Fatal("response without a usable tag and URL must fail")
	}

	if _, err := FetchLatestRelease(context.Background(), "http://127.0.0.1:1"); err == nil {
		t.Fatal("unreachable endpoint must fail")
	}
}

func TestStatusReportsAvailableUpdate(t *testing.T) {
	dir := t.TempDir()
	p := config.Paths{Dir: dir, Config: filepath.Join(dir, "config.json"), Secret: filepath.Join(dir, "tokens.dpapi"), State: filepath.Join(dir, "state.json")}
	d := noticeDashboard(t, p)
	var err error
	if err != nil {
		t.Fatal(err)
	}
	if st := d.Status(); st["updateVersion"] != "" || st["updateURL"] != "" {
		t.Fatalf("no update expected by default: %#v", st)
	}
	d.mu.Lock()
	d.updateVersion, d.updateURL = "9.9.9", "https://example.com/new"
	d.mu.Unlock()
	st := d.Status()
	if st["updateVersion"] != "9.9.9" || st["updateURL"] != "https://example.com/new" {
		t.Fatalf("available update missing from status: %#v", st)
	}
}

func TestCheckUpdateStoresOnlyNewerRelease(t *testing.T) {
	same := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v` + "0.9.6" + `","html_url":"https://example.com/same"}`))
	}))
	defer same.Close()
	dir := t.TempDir()
	p := config.Paths{Dir: dir, Config: filepath.Join(dir, "config.json"), Secret: filepath.Join(dir, "tokens.dpapi"), State: filepath.Join(dir, "state.json")}
	d := noticeDashboard(t, p)
	var err error
	if err != nil {
		t.Fatal(err)
	}
	if err := d.checkUpdateAt(same.URL); err != nil {
		t.Fatal(err)
	}
	if st := d.Status(); st["updateVersion"] != "" {
		t.Fatalf("same version must not report an update: %#v", st)
	}

	newer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v99.0.0","html_url":"https://example.com/new"}`))
	}))
	defer newer.Close()
	if err := d.checkUpdateAt(newer.URL); err != nil {
		t.Fatal(err)
	}
	st := d.Status()
	if st["updateVersion"] != "99.0.0" || st["updateURL"] != "https://example.com/new" {
		t.Fatalf("newer release must be reported: %#v", st)
	}
}

func noticeDashboard(t *testing.T, p config.Paths) *Dashboard {
	t.Helper()
	e, err := app.New(p, config.Data{}, app.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return New(e, p, "0.9.6")
}
