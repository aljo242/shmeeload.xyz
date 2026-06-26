package main

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// realPNG returns a small but compressible PNG (a flat color block) whose
// lossless WebP encoding is smaller than the PNG.
func realPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for i := range img.Pix {
		img.Pix[i] = 0xAA
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	return buf.Bytes()
}

func newTestSite(t *testing.T) *staticSite {
	t.Helper()
	fsys := fstest.MapFS{
		"static/css/home.css": {Data: []byte(strings.Repeat("body{color:red}\n", 40))},
		"static/img/x.png":    {Data: realPNG(t)},
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

	t.Run("serves WebP for a PNG when the client accepts it", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/img/x.png", nil)
		req.Header.Set("Accept", "image/avif,image/webp,image/*,*/*")
		s.serve(rr, req, req.URL.Path)
		if got := rr.Header().Get("Content-Type"); got != "image/webp" {
			t.Errorf("content-type = %q, want image/webp", got)
		}
	})

	t.Run("serves the original PNG when WebP is not accepted", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/img/x.png", nil)
		req.Header.Set("Accept", "image/png,*/*")
		s.serve(rr, req, req.URL.Path)
		if got := rr.Header().Get("Content-Type"); got != "image/png" {
			t.Errorf("content-type = %q, want image/png", got)
		}
	})
}
