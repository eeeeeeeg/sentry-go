package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func registerStaticUI(r http.Handler, staticDir string) http.Handler {
	if staticDir == "" {
		staticDir = "/ui"
	}
	info, err := os.Stat(staticDir)
	if err != nil || !info.IsDir() {
		return r
	}

	fileServer := http.FileServer(http.Dir(staticDir))
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			r.ServeHTTP(w, req)
			return
		}
		if strings.HasPrefix(req.URL.Path, "/api/") ||
			req.URL.Path == "/healthz" ||
			req.URL.Path == "/readyz" ||
			req.URL.Path == "/metrics" {
			r.ServeHTTP(w, req)
			return
		}

		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		requestedPath := filepath.Join(staticDir, filepath.Clean(req.URL.Path))
		if info, err := os.Stat(requestedPath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, req)
			return
		}

		http.ServeFile(w, req, filepath.Join(staticDir, "index.html"))
	})
}
