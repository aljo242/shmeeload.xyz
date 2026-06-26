package handlers

import "net/http"

// ScriptsHandler serves compiled JavaScript and source maps.
func ScriptsHandler(cacheMaxAge int) http.HandlerFunc {
	cc := cacheControl(cacheMaxAge)
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ScriptsHandler", dirJS, cc, scriptContentTypes)
	}
}

// CSSHandler serves stylesheets.
func CSSHandler(cacheMaxAge int) http.HandlerFunc {
	cc := cacheControl(cacheMaxAge)
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "CSSHandler", dirCSS, cc, cssContentTypes)
	}
}

// HTMLHandler serves prepared HTML fragments.
func HTMLHandler(cacheMaxAge int) http.HandlerFunc {
	cc := cacheControl(cacheMaxAge)
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "HTMLHandler", dirHTML, cc, htmlContentTypes)
	}
}

// TypeScriptHandler serves the TypeScript sources as plain text.
func TypeScriptHandler(cacheMaxAge int) http.HandlerFunc {
	cc := cacheControl(cacheMaxAge)
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "TypeScriptHandler", dirSrc, cc, tsContentTypes)
	}
}

// ManifestHandler serves manifest.json from the site root.
func ManifestHandler(cacheMaxAge int) http.HandlerFunc {
	cc := cacheControl(cacheMaxAge)
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ManifestHandler", dirRoot, cc, jsonContentTypes)
	}
}

// ServiceWorkerHandler serves serviceWorker.js (and its map) from the site root.
func ServiceWorkerHandler(cacheMaxAge int) http.HandlerFunc {
	cc := cacheControl(cacheMaxAge)
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ServiceWorkerHandler", dirRoot, cc, scriptContentTypes)
	}
}

// ImageHandler serves image files.
func ImageHandler(cacheMaxAge int) http.HandlerFunc {
	cc := cacheControl(cacheMaxAge)
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ImageHandler", dirImg, cc, imageContentTypes)
	}
}

// ModelHandler serves 3D model files.
func ModelHandler(cacheMaxAge int) http.HandlerFunc {
	cc := cacheControl(cacheMaxAge)
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "ModelHandler", dirModel, cc, modelContentTypes)
	}
}

// MiscFileHandler serves miscellaneous downloadable files.
func MiscFileHandler(cacheMaxAge int) http.HandlerFunc {
	cc := cacheControl(cacheMaxAge)
	return func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, "MiscFileHandler", dirFiles, cc, miscContentTypes)
	}
}
