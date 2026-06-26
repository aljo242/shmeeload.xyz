package handlers

import (
	"net/http"
	"net/http/httptest"
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

	t.Run("path traversal collapses to a base name within the namespace", func(t *testing.T) {
		setupAssets(t, map[string]string{"css/home.css": "safe"})
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/css/placeholder", nil)
		// path.Base reduces this to "home.css", looked up only inside the css
		// namespace, so it serves the in-namespace asset and never escapes.
		req.URL.Path = "/static/css/../../../../etc/home.css"
		CSSHandler(0)(rr, req)
		if rr.Code != http.StatusOK || rr.Body.String() != "safe" {
			t.Fatalf("status=%d body=%q, want 200 %q", rr.Code, rr.Body.String(), "safe")
		}
	})

	t.Run("traversal to an out-of-namespace basename is 404", func(t *testing.T) {
		setupAssets(t, map[string]string{"css/home.css": "safe"})
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/css/x", nil)
		// Collapses to "passwd", which is not in the css namespace.
		req.URL.Path = "/static/css/../../../../etc/passwd"
		CSSHandler(0)(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d, want 404", rr.Code)
		}
	})
}
