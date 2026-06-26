package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeFile(t *testing.T) {
	t.Run("serves an existing asset with content type and cache header", func(t *testing.T) {
		setupAssets(t, map[string]string{"css/home.css": "body{color:red}"})

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/css/home.css", nil)
		CSSHandler(3600)(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if got, want := rr.Header().Get("Content-Type"), "text/css; charset=UTF-8"; got != want {
			t.Errorf("content-type = %q, want %q", got, want)
		}
		if got, want := rr.Header().Get("Cache-Control"), "max-age=3600"; got != want {
			t.Errorf("cache-control = %q, want %q", got, want)
		}
		if got, want := rr.Body.String(), "body{color:red}"; got != want {
			t.Errorf("body = %q, want %q", got, want)
		}
	})

	t.Run("missing asset returns 404", func(t *testing.T) {
		setupAssets(t, nil)
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/css/nope.css", nil)
		CSSHandler(0)(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rr.Code)
		}
	})

	t.Run("non-GET returns 400", func(t *testing.T) {
		setupAssets(t, map[string]string{"css/home.css": "x"})
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/static/css/home.css", nil)
		CSSHandler(0)(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("path traversal collapses to a base name and cannot escape", func(t *testing.T) {
		setupAssets(t, map[string]string{"css/home.css": "safe"})
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/css/placeholder", nil)
		// Try to climb out of the css dir; path.Base reduces this to "home.css",
		// which is only ever looked up inside the in-memory css namespace.
		req.URL.Path = "/static/css/../../../../etc/home.css"
		CSSHandler(0)(rr, req)

		// It must not be possible to read anything outside the css namespace; the
		// only thing that could match is the in-namespace "css/home.css".
		if rr.Code == http.StatusOK && !strings.Contains(rr.Body.String(), "safe") {
			t.Fatalf("traversal returned unexpected content: %q", rr.Body.String())
		}
	})
}
