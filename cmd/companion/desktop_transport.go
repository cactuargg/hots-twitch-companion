package main

import (
	"net/http"
	"strings"
)

const desktopHost = "wails.localhost"

// Wails serves requests inside WebView2 rather than exposing a TCP listener.
// Adapt only the path; preserve the real Host, Origin, headers and body so the
// existing dashboard guards still decide which requests are allowed.
func (d *dashboard) nativeAssetHandler() (http.Handler, error) {
	secret, err := randomSegment()
	if err != nil {
		return nil, err
	}
	routes := d.routes(desktopHost, secret)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cloned := r.Clone(r.Context())
		u := *r.URL
		cloned.URL = &u
		cloned.URL.Path = "/" + secret + "/" + strings.TrimPrefix(r.URL.Path, "/")
		cloned.URL.RawPath = ""
		routes.ServeHTTP(w, cloned)
	}), nil
}

func (d *dashboard) closeRequested() bool {
	select {
	case <-d.shutdown:
		return false
	default:
	}
	d.engine.mu.Lock()
	defer d.engine.mu.Unlock()
	return d.engine.running || d.engine.state.Active != nil
}
