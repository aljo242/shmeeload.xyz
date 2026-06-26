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

// TestMinify checks that text assets are minified before being served.
func TestMinify(t *testing.T) {
	const css = "/* a comment */\nbody {\n    color: red;\n}\n"
	fsys := fstest.MapFS{
		"static/css/a.css": {Data: []byte(css)},
	}
	s, err := newStaticSite(fsys, 3600)
	if err != nil {
		t.Fatalf("newStaticSite: %v", err)
	}

	rr := httptest.NewRecorder()
	s.serve(rr, httptest.NewRequest(http.MethodGet, "/static/css/a.css", nil), "/static/css/a.css")
	got := rr.Body.String()
	if len(got) >= len(css) {
		t.Errorf("served %d bytes, want fewer than the %d-byte source", len(got), len(css))
	}
	if strings.Contains(got, "comment") {
		t.Errorf("minified CSS still contains the comment: %q", got)
	}
}

// TestWebPSiblingPairing checks that newStaticSite attaches a build-time
// "<name>.webp" sibling to its source image and does not expose the sibling at
// its own URL.
func TestWebPSiblingPairing(t *testing.T) {
	fsys := fstest.MapFS{
		"static/img/y.png":      {Data: []byte("raw-png-bytes")},
		"static/img/y.png.webp": {Data: []byte("webp-bytes")},
	}
	s, err := newStaticSite(fsys, 3600)
	if err != nil {
		t.Fatalf("newStaticSite: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static/img/y.png", nil)
	req.Header.Set("Accept", "image/webp")
	s.serve(rr, req, req.URL.Path)
	if got := rr.Header().Get("Content-Type"); got != mimeWebP {
		t.Errorf("content-type = %q, want image/webp", got)
	}
	if got := rr.Body.String(); got != "webp-bytes" {
		t.Errorf("body = %q, want webp-bytes", got)
	}

	// The sibling must not be servable on its own URL.
	if s.serve(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/static/img/y.png.webp", nil), "/static/img/y.png.webp") {
		t.Error("the .webp sibling should not be a standalone asset")
	}
}

// TestImageNegotiation checks that serve() picks the smallest image variant the
// client accepts, across AVIF, WebP, and the original.
func TestImageNegotiation(t *testing.T) {
	asset := func() *staticAsset {
		return &staticAsset{
			contentType: mimePNG,
			etag:        `"png-etag"`,
			raw:         []byte("the-raw-png-bytes"),
			webp:        &variant{body: []byte("webp-bytes"), etag: `"webp-etag"`}, // 10 bytes
			avif:        &variant{body: []byte("avif"), etag: `"avif-etag"`},       // 4 bytes (smallest)
		}
	}
	s := &staticSite{
		cacheControl: "public, max-age=3600",
		assets:       map[string]*staticAsset{"static/img/x.png": asset()},
	}

	cases := []struct {
		name, accept, wantCT, wantBody string
	}{
		{"avif preferred when accepted and smallest", "image/avif,image/webp,*/*", mimeAVIF, "avif"},
		{"webp when avif not accepted", "image/webp,*/*", mimeWebP, "webp-bytes"},
		{"original when neither accepted", "image/png,*/*", mimePNG, "the-raw-png-bytes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/static/img/x.png", nil)
			req.Header.Set("Accept", c.accept)
			s.serve(rr, req, req.URL.Path)
			if got := rr.Header().Get("Content-Type"); got != c.wantCT {
				t.Errorf("content-type = %q, want %q", got, c.wantCT)
			}
			if got := rr.Body.String(); got != c.wantBody {
				t.Errorf("body = %q, want %q", got, c.wantBody)
			}
		})
	}

	t.Run("serves webp when it is smaller than avif even if both accepted", func(t *testing.T) {
		a := asset()
		a.avif = &variant{body: []byte("a-much-larger-avif-body"), etag: `"avif-etag"`}
		small := &staticSite{cacheControl: "x", assets: map[string]*staticAsset{"static/img/x.png": a}}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/img/x.png", nil)
		req.Header.Set("Accept", "image/avif,image/webp,*/*")
		small.serve(rr, req, req.URL.Path)
		if got := rr.Header().Get("Content-Type"); got != mimeWebP {
			t.Errorf("content-type = %q, want image/webp (the smaller one)", got)
		}
	})
}
