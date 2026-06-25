package handlers

import "path/filepath"

// staticRoot is the base directory prepared assets are served from. siteRoot is
// the working-directory root for top-level files (manifest.json, serviceWorker.js).
//
// They are package vars rather than consts so tests can point them at a temp
// directory (via SetStaticRoot / SetSiteRoot) and exercise the handlers without
// depending on a real ./static tree.
var (
	staticRoot = "./static"
	siteRoot   = "."
)

// SetStaticRoot overrides the base directory for served static assets.
func SetStaticRoot(dir string) { staticRoot = dir }

// SetSiteRoot overrides the base directory for top-level site files.
func SetSiteRoot(dir string) { siteRoot = dir }

func htmlDir() string      { return filepath.Join(staticRoot, "html") }
func jsDir() string        { return filepath.Join(staticRoot, "js") }
func cssDir() string       { return filepath.Join(staticRoot, "css") }
func tsDir() string        { return filepath.Join(staticRoot, "src") }
func imgDir() string       { return filepath.Join(staticRoot, "img") }
func modelDir() string     { return filepath.Join(staticRoot, "model") }
func miscFilesDir() string { return filepath.Join(staticRoot, "files") }

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
