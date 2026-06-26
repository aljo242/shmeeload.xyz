package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
)

// staticAsset is one served file, prepared once at startup: its content type, a
// content-hash ETag, the raw bytes, and (for compressible types) a precomputed
// gzip variant so responses never compress on the fly.
type staticAsset struct {
	contentType string
	etag        string
	raw         []byte
	gz          []byte // gzip-compressed bytes; nil when compression doesn't help
}

// staticSite serves an embedded file tree with ETag-based revalidation and
// precompressed responses.
type staticSite struct {
	assets       map[string]*staticAsset // keyed by clean relative path, e.g. "static/css/home.css"
	cacheControl string
}

// newStaticSite indexes every file in fsys, precomputing ETags and gzip variants.
func newStaticSite(fsys fs.FS, cacheMaxAge int) (*staticSite, error) {
	s := &staticSite{
		assets:       make(map[string]*staticAsset),
		cacheControl: "public, max-age=" + strconv.Itoa(cacheMaxAge),
	}
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		ct := mime.TypeByExtension(path.Ext(p))
		if ct == "" {
			ct = http.DetectContentType(raw)
		}
		sum := sha256.Sum256(raw)
		a := &staticAsset{
			contentType: ct,
			etag:        `"` + hex.EncodeToString(sum[:]) + `"`,
			raw:         raw,
		}
		if compressible(ct) {
			if gz := gzipBytes(raw); len(gz) > 0 && len(gz) < len(raw) {
				a.gz = gz
			}
		}
		s.assets[p] = a
		return nil
	})
	return s, err
}

// serve writes the asset at urlPath if present and returns true; it returns
// false (writing nothing) when there is no such asset, so callers can 404.
func (s *staticSite) serve(w http.ResponseWriter, r *http.Request, urlPath string) bool {
	key := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	a, ok := s.assets[key]
	if !ok {
		return false
	}

	h := w.Header()
	h.Set("ETag", a.etag)
	h.Set("Cache-Control", s.cacheControl)
	h.Set("Content-Type", a.contentType)
	h.Add("Vary", "Accept-Encoding")

	if r.Header.Get("If-None-Match") == a.etag {
		w.WriteHeader(http.StatusNotModified)
		return true
	}

	body := a.raw
	if a.gz != nil && acceptsGzip(r) {
		h.Set("Content-Encoding", "gzip")
		body = a.gz
	}
	h.Set("Content-Length", strconv.Itoa(len(body)))

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return true
	}
	_, _ = w.Write(body)
	return true
}

func compressible(contentType string) bool {
	ct := contentType
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	switch {
	case strings.HasPrefix(ct, "text/"):
		return true
	case ct == "application/javascript", ct == "application/json",
		ct == "image/svg+xml", ct == "application/xml", ct == "application/wasm":
		return true
	default:
		return false
	}
}

func acceptsGzip(r *http.Request) bool {
	for _, enc := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		if strings.EqualFold(strings.TrimSpace(strings.SplitN(enc, ";", 2)[0]), "gzip") {
			return true
		}
	}
	return false
}

func gzipBytes(raw []byte) []byte {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil
	}
	if _, err := zw.Write(raw); err != nil {
		return nil
	}
	if err := zw.Close(); err != nil {
		return nil
	}
	return buf.Bytes()
}
