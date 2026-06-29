package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildRouter(t *testing.T) {
	site, err := newStaticSite(siteFS(), 0)
	if err != nil {
		t.Fatalf("newStaticSite: %v", err)
	}
	r := buildRouter(Config{CacheMaxAge: 0}, newHub(nil), site)

	cases := []struct {
		name string
		path string
		want int
	}{
		{"root redirects to /home", "/", http.StatusPermanentRedirect},
		{"home page served from embed", "/home", http.StatusOK},
		{"static css served from embed", "/static/css/home.css", http.StatusOK},
		{"unknown donate currency 404s", "/donate/doge", http.StatusNotFound},
		{"tunes page served from embed", "/tunes/home", http.StatusOK},
		{"shop page served from embed", "/shop/home", http.StatusOK},
		{"robots.txt served", "/robots.txt", http.StatusOK},
		{"sitemap.xml served", "/sitemap.xml", http.StatusOK},
		{"security.txt served", "/.well-known/security.txt", http.StatusOK},
		{"placeholder redirects to under-construction", "/chat/signup", http.StatusTemporaryRedirect},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, c.path, nil))
			if rr.Code != c.want {
				t.Errorf("%s: status = %d, want %d", c.path, rr.Code, c.want)
			}
		})
	}

	// The securityHeaders middleware must be applied through the router.
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/home", nil))
	if rr.Header().Get("Content-Security-Policy") == "" {
		t.Error("expected the security-headers middleware to set Content-Security-Policy")
	}
}
