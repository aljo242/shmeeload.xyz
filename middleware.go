package main

import "net/http"

// securityHeaders sets baseline security headers on every response. The CSP is
// restrictive (same-origin only) but allows inline styles, which the pages use
// for background images, and data: images. HSTS is intentionally omitted here:
// TLS is terminated by the Caddy reverse proxy, which is the right place for it.
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
