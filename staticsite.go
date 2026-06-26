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

const (
	mimePNG  = "image/png"
	mimeGIF  = "image/gif"
	mimeWebP = "image/webp"
)

// variant is an alternate encoding of an asset with its own content-hash ETag.
type variant struct {
	body []byte
	etag string
}

// staticAsset is one served file, prepared once at startup: its content type, a
// content-hash ETag, the raw bytes, precomputed brotli/zstd/gzip variants (for
// compressible types), and a WebP variant (the build-time "<name>.webp" sibling,
// when one was generated for a PNG/GIF).
type staticAsset struct {
	contentType string
	etag        string
	raw         []byte
	br          []byte // brotli; nil when not worth it
	zst         []byte // zstd; nil when not worth it
	gz          []byte // gzip; nil when not worth it
	webp        *variant
}

// staticSite serves an embedded file tree with ETag revalidation, precompressed
// text (brotli > zstd > gzip), and transparent WebP for images that shrink.
type staticSite struct {
	assets       map[string]*staticAsset
	cacheControl string
}

func newStaticSite(fsys fs.FS, cacheMaxAge int) (*staticSite, error) {
	s := &staticSite{
		assets:       make(map[string]*staticAsset),
		cacheControl: "public, max-age=" + strconv.Itoa(cacheMaxAge),
	}
	// First pass: read every file. WebP variants are precomputed at build time
	// (see cmd/genwebp) and embedded as "<name>.webp" siblings, so collect the
	// whole tree before pairing each image with its sibling.
	raw := make(map[string][]byte)
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		raw[p] = b
		return nil
	})
	if err != nil {
		return nil, err
	}

	for p, b := range raw {
		// ".webp" files are attached to their source image below, not served on their own.
		if strings.HasSuffix(p, ".webp") {
			continue
		}
		ct := mime.TypeByExtension(path.Ext(p))
		if ct == "" {
			ct = http.DetectContentType(b)
		}
		a := &staticAsset{contentType: ct, etag: etagOf(b), raw: b}
		if compressible(ct) {
			a.br = smaller(brotliBytes(b), b)
			a.zst = smaller(zstdBytes(b), b)
			a.gz = smaller(gzipBytes(b), b)
		}
		// Pair PNG/GIF with the smaller build-time WebP sibling, when present.
		if ct == mimePNG || ct == mimeGIF {
			if wb, ok := raw[p+".webp"]; ok {
				a.webp = &variant{body: wb, etag: etagOf(wb)}
			}
		}
		s.assets[p] = a
	}
	return s, nil
}

// serve writes the asset at urlPath if present (returning true). It serves WebP
// when the client accepts it and a smaller variant exists, otherwise the raw
// bytes with the best accepted compression. Returns false (writing nothing) on a miss.
func (s *staticSite) serve(w http.ResponseWriter, r *http.Request, urlPath string) bool {
	key := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	a, ok := s.assets[key]
	if !ok {
		return false
	}

	h := w.Header()
	h.Set("Cache-Control", s.cacheControl)
	h.Add("Vary", "Accept-Encoding")

	// Image negotiation: serve the smaller WebP when the client accepts it.
	if a.webp != nil && acceptsWebP(r) {
		h.Add("Vary", "Accept")
		h.Set("Content-Type", mimeWebP)
		h.Set("ETag", a.webp.etag)
		if r.Header.Get("If-None-Match") == a.webp.etag {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
		writeBody(w, r, a.webp.body)
		return true
	}

	h.Set("Content-Type", a.contentType)
	h.Set("ETag", a.etag)
	if r.Header.Get("If-None-Match") == a.etag {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	body, enc := a.negotiate(r.Header.Get("Accept-Encoding"))
	if enc != "" {
		h.Set("Content-Encoding", enc)
	}
	writeBody(w, r, body)
	return true
}

func writeBody(w http.ResponseWriter, r *http.Request, body []byte) {
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	_, _ = w.Write(body)
}

// negotiate picks the best precomputed compression the client accepts (brotli >
// zstd > gzip), falling back to the raw bytes.
func (a *staticAsset) negotiate(acceptEncoding string) (body []byte, encoding string) {
	accepted := parseList(acceptEncoding)
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

func acceptsWebP(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "image/webp")
}

func parseList(header string) map[string]bool {
	out := make(map[string]bool)
	for _, tok := range strings.Split(header, ",") {
		name := strings.TrimSpace(strings.SplitN(tok, ";", 2)[0])
		if name != "" {
			out[strings.ToLower(name)] = true
		}
	}
	return out
}

func etagOf(b []byte) string {
	sum := sha256.Sum256(b)
	return `"` + hex.EncodeToString(sum[:]) + `"`
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
