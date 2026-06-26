package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func newTestSite(t *testing.T) *staticSite {
	t.Helper()
	fsys := fstest.MapFS{
		"static/css/home.css": {Data: []byte(strings.Repeat("body{color:red}\n", 40))},
		"static/img/x.png":    {Data: []byte("\x89PNG not a decodable image")},
	}
	s, err := newStaticSite(fsys, 3600)
	if err != nil {
		t.Fatalf("newStaticSite: %v", err)
	}
	return s
}

func TestStaticSiteServe(t *testing.T) {
	s := newTestSite(t)

	t.Run("prefers brotli when the client accepts br/gzip/zstd", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/css/home.css", nil)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
		if !s.serve(rr, req, req.URL.Path) {
			t.Fatal("expected the asset to be served")
		}
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if rr.Header().Get("ETag") == "" {
			t.Error("missing ETag")
		}
		if got := rr.Header().Get("Content-Encoding"); got != "br" {
			t.Errorf("content-encoding = %q, want br", got)
		}
		if got := rr.Header().Get("Cache-Control"); got != "public, max-age=3600" {
			t.Errorf("cache-control = %q", got)
		}
	})

	t.Run("falls back to gzip when only gzip is accepted", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/css/home.css", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		s.serve(rr, req, req.URL.Path)
		if got := rr.Header().Get("Content-Encoding"); got != "gzip" {
			t.Errorf("content-encoding = %q, want gzip", got)
		}
	})

	t.Run("serves raw when no encoding is accepted", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/css/home.css", nil)
		s.serve(rr, req, req.URL.Path)
		if got := rr.Header().Get("Content-Encoding"); got != "" {
			t.Errorf("content-encoding = %q, want empty", got)
		}
	})

	t.Run("matching If-None-Match yields 304", func(t *testing.T) {
		first := httptest.NewRecorder()
		s.serve(first, httptest.NewRequest(http.MethodGet, "/static/css/home.css", nil), "/static/css/home.css")
		etag := first.Header().Get("ETag")

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/css/home.css", nil)
		req.Header.Set("If-None-Match", etag)
		s.serve(rr, req, req.URL.Path)
		if rr.Code != http.StatusNotModified {
			t.Fatalf("status = %d, want 304", rr.Code)
		}
	})

	t.Run("images are not gzipped", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/img/x.png", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		s.serve(rr, req, req.URL.Path)
		if rr.Header().Get("Content-Encoding") != "" {
			t.Error("image should not be gzip-encoded")
		}
	})

	t.Run("missing asset returns false", func(t *testing.T) {
		rr := httptest.NewRecorder()
		if s.serve(rr, httptest.NewRequest(http.MethodGet, "/nope.css", nil), "/nope.css") {
			t.Fatal("expected serve to report a miss")
		}
	})
}

// TestWebPNegotiation drives the serve() image branch with a hand-built asset
// so the test does not depend on whether a real encoder's WebP happens to beat
// the source PNG (a flat-color fixture compresses smaller as PNG, so no variant
// would be generated).
func TestWebPNegotiation(t *testing.T) {
	s := &staticSite{
		cacheControl: "public, max-age=3600",
		assets: map[string]*staticAsset{
			"static/img/x.png": {
				contentType: mimePNG,
				etag:        `"png-etag"`,
				raw:         []byte("raw-png-bytes"),
				webp:        &variant{body: []byte("webp-bytes"), etag: `"webp-etag"`},
			},
		},
	}

	t.Run("serves WebP when the client accepts it", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/img/x.png", nil)
		req.Header.Set("Accept", "image/avif,image/webp,image/*,*/*")
		s.serve(rr, req, req.URL.Path)
		if got := rr.Header().Get("Content-Type"); got != mimeWebP {
			t.Errorf("content-type = %q, want image/webp", got)
		}
		if got := rr.Header().Get("ETag"); got != `"webp-etag"` {
			t.Errorf("etag = %q, want webp etag", got)
		}
		if got := rr.Body.String(); got != "webp-bytes" {
			t.Errorf("body = %q, want webp-bytes", got)
		}
	})

	t.Run("serves the original PNG when WebP is not accepted", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/img/x.png", nil)
		req.Header.Set("Accept", "image/png,*/*")
		s.serve(rr, req, req.URL.Path)
		if got := rr.Header().Get("Content-Type"); got != mimePNG {
			t.Errorf("content-type = %q, want image/png", got)
		}
		if got := rr.Body.String(); got != "raw-png-bytes" {
			t.Errorf("body = %q, want raw-png-bytes", got)
		}
	})

	t.Run("matching WebP If-None-Match yields 304", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/img/x.png", nil)
		req.Header.Set("Accept", "image/webp")
		req.Header.Set("If-None-Match", `"webp-etag"`)
		s.serve(rr, req, req.URL.Path)
		if rr.Code != http.StatusNotModified {
			t.Fatalf("status = %d, want 304", rr.Code)
		}
	})
}
