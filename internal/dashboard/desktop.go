package dashboard

import (
	"net/http"
	"strings"
)

const DesktopHost = "wails.localhost"

// Wails serves requests inside WebView2 rather than exposing a TCP listener.
// Adapt only the path; preserve the real Host, Origin, headers and body so the
// existing dashboard guards still decide which requests are allowed.
func (d *Dashboard) NativeAssetHandler() (http.Handler, error) {
	secret, err := RandomSegment()
	if err != nil {
		return nil, err
	}
	routes := d.Routes(DesktopHost, secret)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cloned := r.Clone(r.Context())
		u := *r.URL
		cloned.URL = &u
		cloned.URL.Path = "/" + secret + "/" + strings.TrimPrefix(r.URL.Path, "/")
		cloned.URL.RawPath = ""
		routes.ServeHTTP(w, cloned)
	}), nil
}
