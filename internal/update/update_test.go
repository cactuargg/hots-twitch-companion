package update

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(code int, b []byte) *http.Response {
	return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(b))}
}
func executable() []byte {
	b := make([]byte, 512)
	copy(b, "MZ")
	binary.LittleEndian.PutUint32(b[60:], 128)
	copy(b[128:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(b[132:], 0x8664)
	binary.LittleEndian.PutUint16(b[148:], 240)
	binary.LittleEndian.PutUint16(b[150:], 2)
	binary.LittleEndian.PutUint16(b[152:], 0x20b)
	binary.LittleEndian.PutUint32(b[260:], 16)
	return b
}
func checksum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func release(data []byte, edition string) Release {
	name := "HotS-Twitch-Companion-v9.0.0.exe"
	if edition == "browser" {
		name = "HotS-Twitch-Companion-Browser-v9.0.0.zip"
	}
	return Release{Tag: "v9.0.0", HTMLURL: "https://github.com/" + Repository + "/releases/tag/v9.0.0", Assets: []Asset{{Name: name, URL: "https://github.com/" + Repository + "/releases/download/v9.0.0/" + name, Size: int64(len(data)), Digest: "sha256:" + checksum(data)}}}
}
func TestDownloadVerifiesOfficialAssetsAndExtractsOnlyBrowserExecutable(t *testing.T) {
	for _, edition := range []string{"desktop", "browser"} {
		t.Run(edition, func(t *testing.T) {
			exe := executable()
			data := exe
			if edition == "browser" {
				var b bytes.Buffer
				z := zip.NewWriter(&b)
				f, _ := z.Create("HotS-Twitch-Companion-Browser-v9.0.0.exe")
				f.Write(exe)
				f, _ = z.Create("../../must-not-extract")
				f.Write([]byte("ignored"))
				z.Close()
				data = b.Bytes()
			}
			c := NewClient()
			c.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) { return response(200, data), nil })
			dir := t.TempDir()
			p, err := c.Download(context.Background(), release(data, edition), edition, dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			if p.SHA256 != checksum(exe) {
				t.Fatal("wrong executable digest")
			}
			entries, _ := os.ReadDir(dir)
			if len(entries) != 1 || entries[0].Name() != "next.exe" {
				t.Fatalf("unexpected extracted files: %v", entries)
			}
		})
	}
}
func TestRejectsBadOriginChecksumSizeAndMalformedExecutable(t *testing.T) {
	for _, kind := range []string{"origin", "checksum", "size", "body", "pe"} {
		t.Run(kind, func(t *testing.T) {
			data := executable()
			r := release(data, "desktop")
			switch kind {
			case "origin":
				r.Assets[0].URL = "https://evil.example/update.exe"
			case "checksum":
				r.Assets[0].Digest = "sha256:bad"
			case "size":
				r.Assets[0].Size = maxDownload + 1
			case "body":
				data = append(data, 1)
			case "pe":
				data = []byte("not executable")
				r = release(data, "desktop")
			}
			calls := 0
			c := NewClient()
			c.HTTP.Transport = roundTrip(func(req *http.Request) (*http.Response, error) { calls++; return response(200, data), nil })
			dir := t.TempDir()
			if _, err := c.Download(context.Background(), r, "desktop", dir, nil); err == nil {
				t.Fatal("invalid package accepted")
			}
			if (kind == "origin" || kind == "checksum" || kind == "size") && calls != 0 {
				t.Fatal("invalid metadata caused a download")
			}
			if _, err := os.Stat(filepath.Join(dir, "next.exe")); !os.IsNotExist(err) {
				t.Fatal("failed package left executable")
			}
		})
	}
	c := NewClient()
	u, _ := http.NewRequest("GET", "https://evil.example/file", nil)
	if err := c.HTTP.CheckRedirect(u, []*http.Request{{}}); err == nil {
		t.Fatal("untrusted redirect accepted")
	}
}
func TestChecksExposeFailureRespectRateLimitAndRecover(t *testing.T) {
	now := time.Now()
	calls := 0
	c := NewClient()
	c.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			resp := response(403, nil)
			resp.Header.Set("X-RateLimit-Remaining", "0")
			resp.Header.Set("X-RateLimit-Reset", fmtUnix(now.Add(time.Hour)))
			return resp, nil
		}
		b, _ := json.Marshal(release(executable(), "desktop"))
		return response(200, b), nil
	})
	m := NewManager(c, "0.9.9", "desktop", t.TempDir(), true)
	defer m.Close()
	m.now = func() time.Time { return now }
	if err := m.Check(); err != nil {
		t.Fatal(err)
	}
	s := m.Snapshot()
	if s.Phase != "error" || s.Error == "" || s.CheckedAt == 0 || s.NextCheckAt < now.Add(59*time.Minute).UnixMilli() {
		t.Fatalf("lost failure/rate limit: %+v", s)
	}
	if err := m.Check(); !errors.Is(err, ErrCooldown) || calls != 1 {
		t.Fatal("rate-limit retry not suppressed")
	}
	now = now.Add(2 * time.Hour)
	if err := m.Check(); err != nil {
		t.Fatal(err)
	}
	if s = m.Snapshot(); s.Phase != "available" || s.Version != "9.0.0" || !s.Downloadable {
		t.Fatalf("did not recover: %+v", s)
	}
}
func fmtUnix(t time.Time) string { return strconv.FormatInt(t.Unix(), 10) }

func TestDisabledChecksMakeNoRequestsAndDiscardLateResults(t *testing.T) {
	entered, finish := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	c := NewClient()
	c.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		close(entered)
		<-finish
		b, _ := json.Marshal(release(executable(), "desktop"))
		return response(200, b), nil
	})
	m := NewManager(c, "0.9.9", "desktop", t.TempDir(), false)
	defer m.Close()
	if err := m.Check(); !errors.Is(err, ErrDisabled) || calls.Load() != 0 {
		t.Fatal("disabled check used network")
	}
	m.SetEnabled(true)
	done := make(chan struct{})
	go func() { _ = m.Check(); close(done) }()
	<-entered
	m.SetEnabled(false)
	close(finish)
	<-done
	s := m.Snapshot()
	if s.Phase != "disabled" || s.Version != "" || s.CheckedAt != 0 {
		t.Fatalf("late check changed disabled state: %+v", s)
	}
	if err := m.RequestDownload(); !errors.Is(err, ErrDisabled) {
		t.Fatal("download enabled while checks disabled")
	}
}
func TestReplacementRestoresBackupWhenLaunchFails(t *testing.T) {
	root := t.TempDir()
	stage := filepath.Join(root, ".companion-update-test")
	os.Mkdir(stage, 0700)
	old, new := append(executable(), byte(1)), append(executable(), byte(2))
	target, payload := filepath.Join(root, "Companion.exe"), filepath.Join(stage, "next.exe")
	os.WriteFile(target, old, 0700)
	os.WriteFile(payload, new, 0700)
	calls := 0
	err := Replace(target, Prepared{payload, checksum(new), "9.0.0"}, func(path string) error {
		calls++
		b, _ := os.ReadFile(path)
		if calls == 1 {
			if !bytes.Equal(b, new) {
				t.Error("new executable not installed")
			}
			return errors.New("launch failed")
		}
		if !bytes.Equal(b, old) {
			t.Error("backup not restored")
		}
		return nil
	})
	if err == nil || calls != 2 {
		t.Fatal("launch failure was not recovered")
	}
	b, _ := os.ReadFile(target)
	if !bytes.Equal(b, old) {
		t.Fatal("previous executable lost")
	}
}
func TestChangedStagedExecutableNeverReplacesCurrentApp(t *testing.T) {
	root := t.TempDir()
	stage := filepath.Join(root, ".companion-update-test")
	os.Mkdir(stage, 0700)
	target, payload := filepath.Join(root, "Companion.exe"), filepath.Join(stage, "next.exe")
	old := executable()
	os.WriteFile(target, old, 0700)
	os.WriteFile(payload, append(old, 1), 0700)
	if err := Replace(target, Prepared{payload, checksum(old), "9.0.0"}, func(string) error { t.Fatal("unverified executable launched"); return nil }); err == nil {
		t.Fatal("changed payload accepted")
	}
	b, _ := os.ReadFile(target)
	if !bytes.Equal(b, old) {
		t.Fatal("old executable was changed")
	}
}

func TestManagerDownloadPublishesOnlyVerifiedPayloadAndAllowsInstallRetry(t *testing.T) {
	data := executable()
	r := release(data, "desktop")
	c := NewClient()
	c.HTTP.Transport = roundTrip(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == LatestURL {
			b, _ := json.Marshal(r)
			return response(200, b), nil
		}
		return response(200, data), nil
	})
	m := NewManager(c, "0.9.9", "desktop", t.TempDir(), true)
	defer m.Close()
	if err := m.Check(); err != nil {
		t.Fatal(err)
	}
	if err := m.RequestDownload(); err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(2 * time.Second)
	for {
		state := m.Snapshot()
		if state.Phase == "ready" {
			break
		}
		if state.Phase == "error" {
			t.Fatal(state.Error)
		}
		if time.Now().After(until) {
			t.Fatal("download never became ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := m.Install(func(p Prepared) error {
		if p.SHA256 != checksum(data) {
			t.Error("unverified prepared payload")
		}
		return errors.New("idle gate refused")
	}); err == nil {
		t.Fatal("failed handoff accepted")
	}
	if m.Snapshot().Phase != "ready" {
		t.Fatal("failed handoff discarded prepared update")
	}
	if err := m.Install(func(Prepared) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if m.Snapshot().Phase != "installing" {
		t.Fatal("handoff not recorded")
	}
}

func TestWatcherStartsDisabledAndStopsOnClose(t *testing.T) {
	var requests atomic.Int32
	c := NewClient()
	seen := make(chan struct{}, 1)
	c.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		b, _ := json.Marshal(release(executable(), "desktop"))
		select {
		case seen <- struct{}{}:
		default:
		}
		return response(200, b), nil
	})
	m := NewManager(c, "0.9.9", "desktop", t.TempDir(), false)
	done := make(chan struct{})
	go func() { m.Run(); close(done) }()
	if err := m.Check(); !errors.Is(err, ErrDisabled) || requests.Load() != 0 {
		t.Fatal("disabled startup sent a request")
	}
	m.SetEnabled(true)
	select {
	case <-seen:
	case <-time.After(time.Second):
		t.Fatal("enabling checks did not wake watcher")
	}
	m.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watcher did not stop")
	}
}
