// Package frontend embeds the compiled Nuxt SPA and serves it from the
// backend, so a single binary ships both the API and the UI.
//
// In development we don't build the SPA — the dist/ directory only contains
// a placeholder. In that case Handler() returns a 404 and developers reach
// the Nuxt dev server on its own port. In a production build the multi-stage
// Dockerfile drops the generated SPA into dist/ before `go build`, and the
// embed picks it up.
package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// HasBuild reports whether a real Nuxt build was embedded (i.e. an index.html
// is present). Used by the router to decide whether to mount the SPA handler.
func HasBuild() bool {
	root, err := fs.Sub(distFS, "dist")
	if err != nil {
		return false
	}
	_, err = fs.Stat(root, "index.html")
	return err == nil
}

// Handler serves the SPA. Requests for files that exist in dist/ get the file
// directly; anything else (Nuxt client routes like /people, /schedule) falls
// back to index.html so client-side routing can take over.
func Handler() http.Handler {
	root, err := fs.Sub(distFS, "dist")
	if err != nil {
		return http.NotFoundHandler()
	}
	indexBytes, err := fs.ReadFile(root, "index.html")
	if err != nil {
		return http.NotFoundHandler()
	}
	indexHTML := indexBytes
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip leading slash for fs.Open lookups.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			serveIndex(w, indexHTML)
			return
		}
		if f, err := root.Open(path); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// SPA fallback for unknown paths.
		serveIndex(w, indexHTML)
	})
}

func serveIndex(w http.ResponseWriter, html []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(html)
}
