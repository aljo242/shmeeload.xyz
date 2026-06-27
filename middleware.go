package main

import "net/http"

// redirectToApex 301s a request for "www.<apex>" to the bare apex host, so the
// site has one canonical hostname. The destination host is the configured apex
// (not the request's Host header), so it cannot be turned into an open redirect.
// An empty apex (no public domain configured, e.g. LAN) disables it.
func redirectToApex(apex string) func(http.Handler) http.Handler {
	www := "www." + apex
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apex != "" && r.Host == www {
				// #nosec G710 -- host is the fixed configured apex; only the
				// path/query is carried, and it targets our own site.
				http.Redirect(w, r, "https://"+apex+r.URL.RequestURI(), http.StatusMovedPermanently)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

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

// hsts tells browsers to use HTTPS for this host for two years. Only safe with a
// publicly-trusted cert, so it is gated behind the HSTS config flag and applied
// only then (see buildRouter).
func hsts(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
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
