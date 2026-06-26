package handlers

import "time"

// assets holds every served file, keyed by "<dir>/<name>" (and bare names for
// site-root files like manifest.json). The server builds this once at startup
// (package main) and installs it via SetAssets, so handlers serve straight from
// memory and the container needs no writable filesystem.
var (
	assets       = map[string][]byte{}
	assetModTime time.Time
)

// SetAssets installs the in-memory asset map and the modification time reported
// to clients (used for caching/conditional requests).
func SetAssets(m map[string][]byte, modTime time.Time) {
	assets = m
	assetModTime = modTime
}

// Asset directory keys within the asset map.
const (
	dirHTML  = "html"
	dirJS    = "js"
	dirCSS   = "css"
	dirSrc   = "src"
	dirImg   = "img"
	dirModel = "model"
	dirFiles = "files"
	dirRoot  = "" // site root: manifest.json, serviceWorker.js
)

// Common Content-Type values, named so the maps below and the page handlers
// share a single source of truth.
const (
	ctHTML  = "text/html; charset=UTF-8"
	ctCSS   = "text/css; charset=UTF-8"
	ctJS    = "application/javascript; charset=UTF-8"
	ctJSON  = "application/json; charset=UTF-8"
	ctPlain = "text/plain; charset=UTF-8"
)

// Content types keyed by lowercase file extension. A request whose extension is
// absent from the relevant map is served without an explicit Content-Type, so
// net/http sniffs it.
var (
	scriptContentTypes = map[string]string{
		".js":  ctJS,
		".map": ctJSON,
	}
	cssContentTypes   = map[string]string{".css": ctCSS}
	htmlContentTypes  = map[string]string{".html": ctHTML}
	tsContentTypes    = map[string]string{".ts": ctPlain}
	jsonContentTypes  = map[string]string{".json": ctJSON}
	imageContentTypes = map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
	}
	modelContentTypes = map[string]string{
		".dae":  "model/dae",
		".obj":  "model/obj",
		".gltf": "model/gltf",
	}
	miscContentTypes = map[string]string{".pdf": "application/pdf"}
)
