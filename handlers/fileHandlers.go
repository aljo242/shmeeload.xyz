package handlers

import "net/http"

// ScriptsHandler serves compiled JavaScript and source maps.
func ScriptsHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ScriptsHandler", dirJS, cacheMaxAge, scriptContentTypes)
	}
}

// CSSHandler serves stylesheets.
func CSSHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "CSSHandler", dirCSS, cacheMaxAge, cssContentTypes)
	}
}

// HTMLHandler serves prepared HTML fragments.
func HTMLHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "HTMLHandler", dirHTML, cacheMaxAge, htmlContentTypes)
	}
}

// TypeScriptHandler serves the TypeScript sources as plain text.
func TypeScriptHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "TypeScriptHandler", dirSrc, cacheMaxAge, tsContentTypes)
	}
}

// ManifestHandler serves manifest.json from the site root.
func ManifestHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ManifestHandler", dirRoot, cacheMaxAge, jsonContentTypes)
	}
}

// ServiceWorkerHandler serves serviceWorker.js (and its map) from the site root.
func ServiceWorkerHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ServiceWorkerHandler", dirRoot, cacheMaxAge, scriptContentTypes)
	}
}

// ImageHandler serves image files.
func ImageHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ImageHandler", dirImg, cacheMaxAge, imageContentTypes)
	}
}

// ModelHandler serves 3D model files.
func ModelHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ModelHandler", dirModel, cacheMaxAge, modelContentTypes)
	}
}

// MiscFileHandler serves miscellaneous downloadable files.
func MiscFileHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "MiscFileHandler", dirFiles, cacheMaxAge, miscContentTypes)
	}
}
