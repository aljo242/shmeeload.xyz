package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeFile(t *testing.T) {
	root := setupStatic(t)
	writeTestFile(t, filepath.Join(root, "css", "home.css"), "body{color:red}")

	t.Run("serves existing file with content type and cache header", func(t *testing.T) {
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

	t.Run("missing file returns 404", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/css/nope.css", nil)
		CSSHandler(0)(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rr.Code)
		}
	})

	t.Run("non-GET returns 400", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/static/css/home.css", nil)
		CSSHandler(0)(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("path traversal cannot escape the static root", func(t *testing.T) {
		// A secret file outside the static root.
		secret := filepath.Join(t.TempDir(), "secret.txt")
		writeTestFile(t, secret, "TOPSECRET")

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/css/placeholder", nil)
		// Attempt to break out of cssDir up to the secret's absolute path.
		req.URL.Path = "/static/css/../../../../.." + secret

		CSSHandler(0)(rr, req)

		if rr.Code == http.StatusOK && strings.Contains(rr.Body.String(), "TOPSECRET") {
			t.Fatalf("path traversal served a file outside the static root: %q", rr.Body.String())
		}
	})
}
