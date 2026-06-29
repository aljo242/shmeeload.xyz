package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"unicode/utf8"
)

// FuzzSanitizeName checks the name sanitizer always returns a safe value:
// non-empty, printable, single-line, and within the rune cap.
func FuzzSanitizeName(f *testing.F) {
	for _, s := range []string{"", "   ", "alex", "a\x00b\nc\td", strings.Repeat("x", 200), "日本語のなまえ", "\x7f"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		out := sanitizeName(in)
		if out == "" {
			t.Fatalf("sanitizeName(%q) = empty", in)
		}
		if n := utf8.RuneCountInString(out); n > maxNameLen {
			t.Errorf("sanitizeName(%q) = %q is %d runes, over the %d cap", in, out, n, maxNameLen)
		}
		if strings.TrimSpace(out) != out {
			t.Errorf("sanitizeName(%q) = %q has surrounding whitespace", in, out)
		}
		for _, r := range out {
			if r < 0x20 || r == 0x7f {
				t.Errorf("sanitizeName(%q) = %q contains a control char %U", in, out, r)
			}
		}
	})
}

// FuzzStaticServe checks the static handler never panics on arbitrary paths or
// negotiation headers.
func FuzzStaticServe(f *testing.F) {
	site, err := newStaticSite(fstest.MapFS{
		"home.html":        {Data: []byte("<!doctype html><title>h</title><p>hi</p>")},
		"static/css/a.css": {Data: []byte("body { color: red }")},
		"static/img/x.png": {Data: []byte("\x89PNG\r\n not a real image")},
	}, 3600)
	if err != nil {
		f.Fatal(err)
	}
	for _, p := range []string{"/home", "/static/css/a.css", "/static/img/x.png", "/missing", "/../etc/passwd", "//x", ""} {
		f.Add(p, "br, gzip, zstd", "image/avif,image/webp")
	}
	f.Fuzz(func(t *testing.T, path, acceptEncoding, accept string) {
		req := httptest.NewRequest(http.MethodGet, "http://example/", nil)
		req.URL.Path = path
		req.Header.Set("Accept-Encoding", acceptEncoding)
		req.Header.Set("Accept", accept)
		// The only requirement is that it does not panic on any input.
		site.serve(httptest.NewRecorder(), req, path)
	})
}

// FuzzMinify checks minification never panics and never grows the input.
func FuzzMinify(f *testing.F) {
	m := newMinifier()
	for _, ct := range []string{"text/html", "text/css", "text/javascript", "application/json", "image/png", ""} {
		f.Add(ct, []byte("<p> x </p>"))
	}
	f.Fuzz(func(t *testing.T, contentType string, data []byte) {
		out := minifyBytes(m, contentType, data)
		if len(out) > len(data) {
			t.Errorf("minifyBytes(%q) grew the input: %d > %d", contentType, len(out), len(data))
		}
	})
}

func BenchmarkStaticServe(b *testing.B) {
	site, err := newStaticSite(fstest.MapFS{
		"home.html": {Data: []byte(strings.Repeat("<p>hello world</p>\n", 200))},
	}, 3600)
	if err != nil {
		b.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://example/home.html", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		site.serve(httptest.NewRecorder(), req, "/home.html")
	}
}

func BenchmarkSanitizeName(b *testing.B) {
	const in = "  some\x00 user\tname here that is fairly long  "
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sanitizeName(in)
	}
}

func BenchmarkChatLine(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		chatLine("alex: hello world, check out https://djinntek.space")
	}
}
