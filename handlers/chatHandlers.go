package handlers

import (
	"net/http"
	"path/filepath"
)

// ChatHomeHandler serves the chat home page where users can get assigned unique
// identifiers. It currently only serves static resources.
func ChatHomeHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servePage(w, r, "ChatHomeHandler", "chat.html", cacheMaxAge,
			filepath.Join(cssDir(), "chat.css"),
			filepath.Join(jsDir(), "chat.js"),
		)
	}
}
