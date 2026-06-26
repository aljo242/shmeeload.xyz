package handlers

import "net/http"

// ChatHomeHandler serves the chat home page where users can get assigned unique
// identifiers. It currently only serves static resources.
func ChatHomeHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servePage(w, r, "ChatHomeHandler", "chat.html", cacheMaxAge)
	}
}
