package handlers

import "net/http"

// ScriptsHandler serves compiled JavaScript and source maps from the js dir.
func ScriptsHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ScriptsHandler", jsDir(), cacheMaxAge, scriptContentTypes)
	}
}

// CSSHandler serves stylesheets from the css dir.
func CSSHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "CSSHandler", cssDir(), cacheMaxAge, cssContentTypes)
	}
}

// HTMLHandler serves prepared HTML fragments from the html dir.
func HTMLHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "HTMLHandler", htmlDir(), cacheMaxAge, htmlContentTypes)
	}
}

// TypeScriptHandler serves the TypeScript sources from the src dir as plain text.
func TypeScriptHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "TypeScriptHandler", tsDir(), cacheMaxAge, tsContentTypes)
	}
}

// ManifestHandler serves manifest.json from the site root.
func ManifestHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ManifestHandler", siteRoot, cacheMaxAge, jsonContentTypes)
	}
}

// ServiceWorkerHandler serves serviceWorker.js (and its map) from the site root.
func ServiceWorkerHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ServiceWorkerHandler", siteRoot, cacheMaxAge, scriptContentTypes)
	}
}

// ImageHandler serves image files from the img dir.
func ImageHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ImageHandler", imgDir(), cacheMaxAge, imageContentTypes)
	}
}

// ModelHandler serves 3D model files from the model dir.
func ModelHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ModelHandler", modelDir(), cacheMaxAge, modelContentTypes)
	}
}

// MiscFileHandler serves miscellaneous downloadable files from the files dir.
func MiscFileHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "MiscFileHandler", miscFilesDir(), cacheMaxAge, miscContentTypes)
	}
}
