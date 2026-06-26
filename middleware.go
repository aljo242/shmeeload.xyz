package main

import "net/http"

// securityHeaders sets baseline security headers on every response. The CSP is
// restrictive (same-origin only) but allows inline styles, which the pages use
// for background images, and data: images. HSTS is intentionally omitted here:
// browsers ignore it over a self-signed/LAN setup, and it belongs at the public
// edge once a real public cert is in play.
func securityHeaders(next http.Handler) http.Handler {
	const csp = "default-src 'self'; " +
		"img-src 'self' data:; " +
		"style-src 'self' 'unsafe-inline'; " +
		"script-src 'self'; " +
		"object-src 'none'; " +
		"base-uri 'self'; " +
		"frame-ancestors 'none'"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", csp)
		next.ServeHTTP(w, r)
	})
}

// altSvc advertises HTTP/3 availability on the given port via the Alt-Svc header
// so capable clients upgrade from TCP (h1/h2) to QUIC (h3).
func altSvc(port string) func(http.Handler) http.Handler {
	value := `h3=":` + port + `"; ma=86400`
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Alt-Svc", value)
			next.ServeHTTP(w, r)
		})
	}
}
