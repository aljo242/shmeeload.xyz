package handlers

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
)

// serveFile serves the file named by the request path's final element from dir,
// choosing a Content-Type by extension and setting a Cache-Control max-age. It
// writes 400 for non-GET requests and 404 when the file is absent.
//
// filepath.Base strips any directory components from the request path, so a
// crafted name like "../../etc/passwd" collapses to "passwd" and cannot escape
// dir. This is the single choke point every static asset handler shares.
func serveFile(w http.ResponseWriter, r *http.Request, handlerName, dir string, cacheMaxAge int, contentTypes map[string]string) {
	name := filepath.Base(r.URL.Path)
	log.Debug().Str("Handler", handlerName).Str("Filename", name).Msg("incoming request")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	wantFile := filepath.Join(dir, name)
	if !fileExists(wantFile) {
		w.WriteHeader(http.StatusNotFound)
		log.Debug().Str("Filename", wantFile).Msg("file not found")
		return
	}

	setContentType(w, wantFile, contentTypes)
	w.Header().Set("Cache-Control", cacheControl(cacheMaxAge))
	http.ServeFile(w, r, wantFile)
}

// servePage serves a fixed HTML page from the html directory. It writes 400 for
// non-GET requests and 404 when the page is missing.
func servePage(w http.ResponseWriter, r *http.Request, handlerName, page string, cacheMaxAge int) {
	log.Debug().Str("Handler", handlerName).Msg("incoming request")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	wantFile := filepath.Join(htmlDir(), page)
	if !fileExists(wantFile) {
		w.WriteHeader(http.StatusNotFound)
		log.Error().Str("Filename", wantFile).Msg("page not found")
		return
	}

	w.Header().Set("Content-Type", ctHTML)
	w.Header().Set("Cache-Control", cacheControl(cacheMaxAge))
	http.ServeFile(w, r, wantFile)
}

// fileExists reports whether path resolves to something on disk. A stat error
// other than "not found" (e.g. a permission problem) is treated as existing and
// left for http.ServeFile to surface.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

func setContentType(w http.ResponseWriter, path string, contentTypes map[string]string) {
	if ct := contentTypes[filepath.Ext(path)]; ct != "" {
		w.Header().Set("Content-Type", ct)
	}
}

func cacheControl(maxAge int) string {
	return "max-age=" + strconv.Itoa(maxAge)
}
