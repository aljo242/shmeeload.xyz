package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildRouter(t *testing.T) {
	r := buildRouter(Config{CacheMaxAge: 0}, newHub())

	cases := []struct {
		name string
		path string
		want int
	}{
		{"root redirects to /home", "/", http.StatusPermanentRedirect},
		{"home page served from embed", "/home", http.StatusOK},
		{"static css served from embed", "/static/css/home.css", http.StatusOK},
		{"unknown donate currency 404s", "/donate/doge", http.StatusNotFound},
		{"placeholder redirects to under-construction", "/tunes/home", http.StatusTemporaryRedirect},
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
