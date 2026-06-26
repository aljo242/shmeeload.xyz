package handlers

import (
	"bytes"
	"net/http"
	"path"
	"strconv"

	"github.com/aljo242/shmeeload.xyz/internal/log"
)

// serveFile serves the asset named by the request path's final element from the
// given asset directory, choosing a Content-Type by extension and setting a
// Cache-Control max-age. It writes 400 for non-GET requests and 404 when the
// asset is absent.
//
// path.Base strips any directory components from the request path, so a crafted
// name like "../../etc/passwd" collapses to "passwd" and can only ever hit the
// in-memory map. This is the single choke point every static asset handler shares.
func serveFile(w http.ResponseWriter, r *http.Request, handlerName, dir string, cacheMaxAge int, contentTypes map[string]string) {
	name := path.Base(r.URL.Path)
	log.Debug("incoming request", "handler", handlerName, "filename", name)

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	content, ok := assets[path.Join(dir, name)]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		log.Debug("asset not found", "handler", handlerName, "filename", name)
		return
	}

	setContentType(w, name, contentTypes)
	w.Header().Set("Cache-Control", cacheControl(cacheMaxAge))
	http.ServeContent(w, r, name, assetModTime, bytes.NewReader(content))
}

// servePage serves a fixed HTML page from the html asset directory. It writes
// 400 for non-GET requests and 404 when the page is missing.
func servePage(w http.ResponseWriter, r *http.Request, handlerName, page string, cacheMaxAge int) {
	log.Debug("incoming request", "handler", handlerName)

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	content, ok := assets[path.Join(dirHTML, page)]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		log.Error("page not found", "handler", handlerName, "page", page)
		return
	}

	w.Header().Set("Content-Type", ctHTML)
	w.Header().Set("Cache-Control", cacheControl(cacheMaxAge))
	http.ServeContent(w, r, page, assetModTime, bytes.NewReader(content))
}

func setContentType(w http.ResponseWriter, name string, contentTypes map[string]string) {
	if ct := contentTypes[path.Ext(name)]; ct != "" {
		w.Header().Set("Content-Type", ct)
	}
}

func cacheControl(maxAge int) string {
	return "max-age=" + strconv.Itoa(maxAge)
}
