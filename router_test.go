package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func newRouterTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	hub := newHub(nil)
	go hub.run()
	site, err := newStaticSite(fstest.MapFS{
		"home.html":        {Data: []byte("<!doctype html><title>home</title>")},
		"static/css/x.css": {Data: []byte("body{color:red}")},
	}, 3600)
	if err != nil {
		t.Fatalf("newStaticSite: %v", err)
	}
	srv := httptest.NewServer(buildRouter(Config{Port: "0"}, hub, site))
	t.Cleanup(func() {
		srv.Close()
		hub.stop()
	})
	return srv
}

func TestRouting(t *testing.T) {
	srv := newRouterTestServer(t)
	// Do not follow redirects, so we can assert the redirect status codes.
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	cases := []struct {
		name, method, path string
		wantCode           int
		wantCacheControl   string
	}{
		{"root redirects to home", http.MethodGet, "/", http.StatusPermanentRedirect, ""},
		{"home page, uncached", http.MethodGet, "/home", http.StatusOK, noCacheControl},
		{"css revalidates", http.MethodGet, "/static/css/x.css", http.StatusOK, noCacheControl},
		{"missing asset", http.MethodGet, "/nope.xyz", http.StatusNotFound, ""},
		{"non-GET is rejected", http.MethodPost, "/home", http.StatusMethodNotAllowed, ""},
		{"donate known currency", http.MethodGet, "/donate/btc", http.StatusOK, ""},
		{"donate unknown currency", http.MethodGet, "/donate/zzz", http.StatusNotFound, ""},
		{"not-yet-built redirects", http.MethodGet, "/chat/signup", http.StatusTemporaryRedirect, ""},
		{"security.txt", http.MethodGet, "/.well-known/security.txt", http.StatusOK, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req, err := http.NewRequest(c.method, srv.URL+c.path, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("do: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != c.wantCode {
				t.Errorf("status = %d, want %d", resp.StatusCode, c.wantCode)
			}
			if c.wantCacheControl != "" {
				if got := resp.Header.Get("Cache-Control"); got != c.wantCacheControl {
					t.Errorf("Cache-Control = %q, want %q", got, c.wantCacheControl)
				}
			}
		})
	}
}
