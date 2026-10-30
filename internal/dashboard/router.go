package dashboard

import (
	"io/fs"
	"net/http"
	"strings"
)

// NewHTTPHandler guards the loopback dashboard with an unguessable URL
// capability, origin checks, and security headers before invoking API routes.
func NewHTTPHandler(addr, secret string, api func(http.ResponseWriter, *http.Request, string), activity func()) http.Handler {
	prefix := "/" + secret + "/"
	web, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(web))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != addr {
			http.Error(w, "Unknown Host", http.StatusForbidden)
			return
		}
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; script-src 'self'; style-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		if strings.HasPrefix(r.URL.Path, prefix+"api/") {
			if activity != nil {
				activity()
			}
		}
		sub := strings.TrimPrefix(r.URL.Path, prefix)
		// Only the trusted widget preview document can be framed by this UI origin.
		if sub == "assets/lobby-widget.html" {
			w.Header().Set("X-Frame-Options", "SAMEORIGIN")
			w.Header().Set("Content-Security-Policy", strings.Replace(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'", "frame-ancestors 'self'", 1))
		}
		if !strings.HasPrefix(sub, "api/") {
			if sub == "" {
				page, err := webFiles.ReadFile("web/index.html")
				if err != nil {
					http.Error(w, "UI unavailable", 500)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-store")
				_, _ = w.Write(page)
				return
			}
			clone := r.Clone(r.Context())
			urlCopy := *clone.URL
			clone.URL = &urlCopy
			clone.URL.Path = "/" + sub
			files.ServeHTTP(w, clone)
			return
		}
		if r.Header.Get("X-Companion") != "1" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if r.Method == "POST" {
			origin := r.Header.Get("Origin")
			if origin != "" && origin != "http://"+addr {
				http.Error(w, "Origin refused", http.StatusForbidden)
				return
			}
			if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
				http.Error(w, "Cross-site requests denied", http.StatusForbidden)
				return
			}
		}
		api(w, r, strings.TrimPrefix(sub, "api/"))
	})
}
