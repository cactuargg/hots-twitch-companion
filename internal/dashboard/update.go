package dashboard

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "strconv"
 "strings"
)

type ReleaseInfo struct {
 Tag string `json:"tag_name"`
 HTMLURL string `json:"html_url"`
}

// FetchLatestRelease reads bounded public release metadata; failures are reported to callers.
func FetchLatestRelease(ctx context.Context, endpoint string) (ReleaseInfo, error) {
 req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
 if err != nil { return ReleaseInfo{}, err }
 req.Header.Set("User-Agent", "HotS-Twitch-Companion")
 req.Header.Set("Accept", "application/vnd.github+json")
 resp, err := http.DefaultClient.Do(req)
 if err != nil { return ReleaseInfo{}, err }
 defer resp.Body.Close()
 if resp.StatusCode != http.StatusOK { return ReleaseInfo{}, fmt.Errorf("release check: HTTP %d", resp.StatusCode) }
 var info ReleaseInfo
 if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&info); err != nil { return ReleaseInfo{}, err }
 if !strings.HasPrefix(info.Tag, "v") || info.HTMLURL == "" { return ReleaseInfo{}, errors.New("release check: unexpected response") }
 return info, nil
}

// IsNewerRelease rejects malformed or overflowing version components rather than treating them as zero.
func IsNewerRelease(latest, current string) bool {
 parse := func(v string) ([3]uint64, bool) {
  var out [3]uint64
  v = strings.TrimPrefix(strings.TrimSpace(v), "v")
  parts := strings.Split(v, ".")
  if len(parts) != 3 { return out, false }
  for i, part := range parts {
   if part == "" { return out, false }
   for _, c := range part { if c < '0' || c > '9' { return out, false } }
   n, err := strconv.ParseUint(part, 10, 64)
   if err != nil { return out, false }
   out[i] = n
  }
  return out, true
 }
 l, okL := parse(latest)
 c, okC := parse(current)
 if !okL || !okC { return false }
 for i := range l { if l[i] != c[i] { return l[i] > c[i] } }
 return false
}
