package handlers

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

// RedirectHome permanently redirects to /home.
func RedirectHome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Str("Handler", "RedirectHome").Str("Request URL", r.URL.Path).Msg("incoming request")
		http.Redirect(w, r, "/home", http.StatusPermanentRedirect)
	}
}

// RedirectConstructionHandler temporarily redirects to /under-construction.
func RedirectConstructionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Str("Handler", "RedirectConstructionHandler").Str("Request URL", r.URL.Path).Msg("incoming request")
		http.Redirect(w, r, "/under-construction", http.StatusTemporaryRedirect)
	}
}

// ConstructionHandler serves the under-construction page.
func ConstructionHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servePage(w, r, "ConstructionHandler", "construction.html", cacheMaxAge)
	}
}

// HomeHandler serves the home page.
func HomeHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servePage(w, r, "HomeHandler", "home.html", cacheMaxAge)
	}
}

// ResumeHomeHandler serves the resume page.
func ResumeHomeHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servePage(w, r, "ResumeHomeHandler", "resume.html", cacheMaxAge)
	}
}

// HallofArtHomeHandler serves the hall-of-art page.
func HallofArtHomeHandler(cacheMaxAge int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servePage(w, r, "HallofArtHomeHandler", "shadow.html", cacheMaxAge)
	}
}
