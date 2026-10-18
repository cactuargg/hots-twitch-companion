// Package update owns release discovery, bounded verified downloads and portable
// replacement. It never reads Twitch credentials or mutates prediction state.
package update

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const Repository = "cactuargg/hots-twitch-companion"
const LatestURL = "https://api.github.com/repos/" + Repository + "/releases/latest"
const maxDownload = 128 << 20

type Asset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}
type Release struct {
	Tag        string  `json:"tag_name"`
	HTMLURL    string  `json:"html_url"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Assets     []Asset `json:"assets"`
}
type APIError struct {
	Status  int
	RetryAt time.Time
}

func (e *APIError) Error() string { return fmt.Sprintf("GitHub HTTP %d", e.Status) }

type Client struct {
	HTTP     *http.Client
	Endpoint string
}

func NewClient() *Client {
	return &Client{HTTP: &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || !trustedURL(req.URL) {
			return errors.New("update redirect refused")
		}
		return nil
	}}, Endpoint: LatestURL}
}
func trustedURL(u *url.URL) bool {
	if u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	switch u.Hostname() {
	case "api.github.com", "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return true
	}
	return false
}
func (c *Client) get(ctx context.Context, endpoint string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "HotS-Twitch-Companion")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		retry := time.Time{}
		if seconds, e := strconv.Atoi(resp.Header.Get("Retry-After")); e == nil && seconds > 0 {
			retry = time.Now().Add(time.Duration(seconds) * time.Second)
		} else if date, e := http.ParseTime(resp.Header.Get("Retry-After")); e == nil {
			retry = date
		}
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			if n, e := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); e == nil && time.Unix(n, 0).After(retry) {
				retry = time.Unix(n, 0)
			}
		}
		return nil, &APIError{resp.StatusCode, retry}
	}
	return resp, nil
}
func (c *Client) Fetch(ctx context.Context) (Release, error) {
	resp, err := c.get(ctx, c.Endpoint)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	var r Release
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r); err != nil {
		return r, err
	}
	if !r.valid() {
		return r, errors.New("invalid official release metadata")
	}
	return r, nil
}
func (r Release) Package(edition string) (Asset, error) {
	if !r.valid() {
		return Asset{}, errors.New("invalid official release")
	}
	name := "HotS-Twitch-Companion-" + r.Tag + ".exe"
	if edition == "browser" {
		name = "HotS-Twitch-Companion-Browser-" + r.Tag + ".zip"
	} else if edition != "desktop" {
		return Asset{}, errors.New("unsupported update edition")
	}
	for _, a := range r.Assets {
		if a.Name == name {
			hash, err := hex.DecodeString(strings.TrimPrefix(a.Digest, "sha256:"))
			if !strings.HasPrefix(a.Digest, "sha256:") || err != nil || len(hash) != sha256.Size || a.Size <= 0 || a.Size > maxDownload || a.URL != "https://github.com/"+Repository+"/releases/download/"+r.Tag+"/"+name {
				return Asset{}, errors.New("missing or invalid package checksum, size or origin")
			}
			return a, nil
		}
	}
	return Asset{}, errors.New("release package is not available")
}

func (r Release) valid() bool {
	return strings.HasPrefix(r.Tag, "v") && strings.TrimSpace(r.Tag) == r.Tag && IsNewerRelease(r.Tag, "0.0.0") && !r.Draft && !r.Prerelease && r.HTMLURL == "https://github.com/"+Repository+"/releases/tag/"+r.Tag
}

type Prepared struct {
	Path    string
	SHA256  string
	Version string
}

func (c *Client) Download(ctx context.Context, r Release, edition, dir string, progress func(int64, int64)) (prepared Prepared, err error) {
	a, err := r.Package(edition)
	if err != nil {
		return prepared, err
	}
	archive := filepath.Join(dir, "package.download")
	f, err := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return prepared, err
	}
	defer func() {
		f.Close()
		os.Remove(archive)
		if err != nil {
			os.Remove(filepath.Join(dir, "next.exe"))
		}
	}()
	resp, err := c.get(ctx, a.URL)
	if err != nil {
		return prepared, err
	}
	defer resp.Body.Close()
	h := sha256.New()
	counter := &progressWriter{total: a.Size, report: progress}
	n, err := io.Copy(io.MultiWriter(f, h, counter), io.LimitReader(resp.Body, a.Size+1))
	if err != nil {
		return prepared, err
	}
	if n != a.Size || hex.EncodeToString(h.Sum(nil)) != strings.TrimPrefix(a.Digest, "sha256:") {
		return prepared, errors.New("update size or SHA-256 mismatch")
	}
	if err = f.Sync(); err != nil {
		return prepared, err
	}
	if err = f.Close(); err != nil {
		return prepared, err
	}
	dest := filepath.Join(dir, "next.exe")
	if edition == "desktop" {
		err = os.Rename(archive, dest)
	} else {
		err = extractBrowser(archive, dest, r.Tag)
	}
	if err != nil {
		return prepared, err
	}
	digest, err := VerifyExecutable(dest)
	if err != nil {
		return prepared, err
	}
	return Prepared{dest, digest, strings.TrimPrefix(r.Tag, "v")}, nil
}

type progressWriter struct {
	n, total int64
	report   func(int64, int64)
}

func (w *progressWriter) Write(b []byte) (int, error) {
	w.n += int64(len(b))
	if w.report != nil {
		w.report(w.n, w.total)
	}
	return len(b), nil
}
func extractBrowser(archive, dest, tag string) error {
	z, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer z.Close()
	name := "HotS-Twitch-Companion-Browser-" + tag + ".exe"
	var candidate *zip.File
	for _, f := range z.File {
		if f.Name == name {
			if candidate != nil {
				return errors.New("duplicate executable in update archive")
			}
			candidate = f
		}
	}
	if candidate == nil || !candidate.Mode().IsRegular() || candidate.UncompressedSize64 > maxDownload {
		return errors.New("invalid Browser executable in archive")
	}
	src, err := candidate.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(dst, io.LimitReader(src, maxDownload+1))
	syncErr := dst.Sync()
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	if n > maxDownload {
		return errors.New("executable too large")
	}
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}
func VerifyExecutable(path string) (string, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() || st.Size() > maxDownload {
		return "", errors.New("invalid executable file")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	image, err := pe.NewFile(f)
	if err != nil {
		return "", err
	}
	defer image.Close()
	if image.FileHeader.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || image.FileHeader.Characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE == 0 || image.FileHeader.Characteristics&pe.IMAGE_FILE_DLL != 0 {
		return "", errors.New("update is not a Windows x64 executable")
	}
	if _, ok := image.OptionalHeader.(*pe.OptionalHeader64); !ok {
		return "", errors.New("missing Windows x64 optional header")
	}
	if _, err = f.Seek(0, 0); err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
