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

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

// staticAsset is one served file, prepared once at startup: its content type, a
// content-hash ETag, the raw bytes, and (for compressible types) precomputed
// brotli/zstd/gzip variants so responses never compress on the fly.
type staticAsset struct {
	contentType string
	etag        string
	raw         []byte
	br          []byte // brotli; nil when not worth it
	zst         []byte // zstd; nil when not worth it
	gz          []byte // gzip; nil when not worth it
}

// staticSite serves an embedded file tree with ETag revalidation and
// precompressed responses (brotli > zstd > gzip, by Accept-Encoding).
type staticSite struct {
	assets       map[string]*staticAsset
	cacheControl string
}

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
			a.br = smaller(brotliBytes(raw), raw)
			a.zst = smaller(zstdBytes(raw), raw)
			a.gz = smaller(gzipBytes(raw), raw)
		}
		s.assets[p] = a
		return nil
	})
	return s, err
}

// serve writes the asset at urlPath if present (returning true), negotiating the
// best available encoding; it returns false (writing nothing) on a miss.
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

	body, enc := a.negotiate(r.Header.Get("Accept-Encoding"))
	if enc != "" {
		h.Set("Content-Encoding", enc)
	}
	h.Set("Content-Length", strconv.Itoa(len(body)))

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return true
	}
	_, _ = w.Write(body)
	return true
}

// negotiate picks the best precomputed encoding the client accepts, preferring
// brotli, then zstd, then gzip, falling back to the raw bytes.
func (a *staticAsset) negotiate(acceptEncoding string) (body []byte, encoding string) {
	accepted := parseAcceptEncoding(acceptEncoding)
	switch {
	case a.br != nil && accepted["br"]:
		return a.br, "br"
	case a.zst != nil && accepted["zstd"]:
		return a.zst, "zstd"
	case a.gz != nil && accepted["gzip"]:
		return a.gz, "gzip"
	default:
		return a.raw, ""
	}
}

func parseAcceptEncoding(header string) map[string]bool {
	out := make(map[string]bool)
	for _, tok := range strings.Split(header, ",") {
		name := strings.TrimSpace(strings.SplitN(tok, ";", 2)[0])
		if name != "" {
			out[strings.ToLower(name)] = true
		}
	}
	return out
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

// smaller returns compressed only when it actually beats raw, else nil.
func smaller(compressed, raw []byte) []byte {
	if len(compressed) > 0 && len(compressed) < len(raw) {
		return compressed
	}
	return nil
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

func brotliBytes(raw []byte) []byte {
	var buf bytes.Buffer
	bw := brotli.NewWriterLevel(&buf, brotli.BestCompression)
	if _, err := bw.Write(raw); err != nil {
		return nil
	}
	if err := bw.Close(); err != nil {
		return nil
	}
	return buf.Bytes()
}

func zstdBytes(raw []byte) []byte {
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil
	}
	defer enc.Close()
	return enc.EncodeAll(raw, nil)
}
